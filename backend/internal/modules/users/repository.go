package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound       = errors.New("user was not found")
	ErrWarehouseNotFound  = errors.New("warehouse was not found or is not active")
	ErrEmailConflict      = errors.New("email is already in use")
	ErrInvalidRole        = errors.New("role is not a known role code")
	ErrLastAdminProtected = errors.New("this action would leave zero active administrators")
)

type Repository interface {
	CreateUser(ctx context.Context, input UserCreateInput, passwordHash string) (User, error)
	ListUsers(ctx context.Context, filter ListFilter) ([]User, error)
	GetUser(ctx context.Context, id string) (User, error)
	UpdateUser(ctx context.Context, id string, input UserUpdateInput) (User, error)
	DeactivateUser(ctx context.Context, id string) error
	ResetPassword(ctx context.Context, id string, passwordHash string) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &PostgresRepository{pool: pool}
}

type rowScanner interface {
	Scan(...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Email, &user.FullName, &user.Role,
		&user.WarehouseID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalStringValue(value OptionalString) any {
	if !value.Set || value.Value == nil {
		return nil
	}
	return *value.Value
}

func cursorArguments(cursor *Cursor) (any, any) {
	if cursor == nil {
		return nil, nil
	}
	return cursor.CreatedAt, cursor.ID
}

func resolveRoleID(ctx context.Context, tx pgx.Tx, code string) (string, error) {
	var roleID string
	err := tx.QueryRow(ctx, `SELECT id::text FROM roles WHERE code = $1`, code).Scan(&roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidRole
	}
	if err != nil {
		return "", fmt.Errorf("resolve role id: %w", err)
	}
	return roleID, nil
}

func lockActiveWarehouse(ctx context.Context, tx pgx.Tx, id string) error {
	var lockedID string
	err := tx.QueryRow(ctx,
		`SELECT id::text FROM warehouses WHERE id = $1 AND is_active = TRUE FOR UPDATE`, id,
	).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWarehouseNotFound
	}
	if err != nil {
		return fmt.Errorf("lock active warehouse: %w", err)
	}
	return nil
}

func mapUsersWriteError(operation string, err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		if pgError.ConstraintName == "users_email_unique_idx" {
			return ErrEmailConflict
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func (repository *PostgresRepository) CreateUser(
	ctx context.Context,
	input UserCreateInput,
	passwordHash string,
) (User, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin create user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	roleID, err := resolveRoleID(ctx, tx, input.Role)
	if err != nil {
		return User{}, err
	}

	if input.WarehouseID.Set && input.WarehouseID.Value != nil {
		if err := lockActiveWarehouse(ctx, tx, *input.WarehouseID.Value); err != nil {
			return User{}, err
		}
	}

	user, err := scanUser(tx.QueryRow(ctx, `
		INSERT INTO users (role_id, warehouse_id, email, password_hash, full_name, is_active)
		VALUES ($1, $2, LOWER($3), $4, $5, TRUE)
		RETURNING id::text, email, full_name, $6::text, warehouse_id::text, is_active, created_at, updated_at`,
		roleID, optionalStringValue(input.WarehouseID), input.Email, passwordHash, input.FullName, input.Role))
	if err != nil {
		return User{}, mapUsersWriteError("create user", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit create user: %w", err)
	}
	return user, nil
}

func (repository *PostgresRepository) ListUsers(ctx context.Context, filter ListFilter) ([]User, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := repository.pool.Query(ctx, `
		SELECT u.id::text, u.email, u.full_name, r.code, u.warehouse_id::text, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE ($1 = '' OR u.email ILIKE '%' || $1 || '%' OR u.full_name ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR r.code = $2)
		  AND ($3::uuid IS NULL OR u.warehouse_id = $3::uuid)
		  AND ($4::boolean IS NULL OR u.is_active = $4)
		  AND (
			$5::timestamptz IS NULL
			OR (u.created_at, u.id) < ($5::timestamptz, $6::uuid)
		  )
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT $7`,
		filter.Search, filter.Role, nullableString(filter.WarehouseID), nullableBool(filter.IsActive),
		afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (repository *PostgresRepository) GetUser(ctx context.Context, id string) (User, error) {
	user, err := scanUser(repository.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.full_name, r.code, u.warehouse_id::text, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// requireMoreThanOneActiveAdmin locks every currently-active admin row and fails
// with ErrLastAdminProtected if fewer than two exist. It must be called inside the
// same transaction as the write that would reduce the admin count, so a concurrent
// deactivation of a different admin cannot race past this check.
func requireMoreThanOneActiveAdmin(ctx context.Context, tx pgx.Tx) error {
	rows, err := tx.Query(ctx, `
		SELECT u.id::text
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.code = 'admin' AND u.is_active = TRUE
		FOR UPDATE OF u`)
	if err != nil {
		return fmt.Errorf("lock active admins: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("count active admins: %w", err)
	}
	if count <= 1 {
		return ErrLastAdminProtected
	}
	return nil
}

func (repository *PostgresRepository) UpdateUser(
	ctx context.Context,
	id string,
	input UserUpdateInput,
) (User, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin update user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentRoleCode string
	var currentIsActive bool
	err = tx.QueryRow(ctx, `
		SELECT r.code, u.is_active
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1
		FOR UPDATE OF u`, id,
	).Scan(&currentRoleCode, &currentIsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("lock user for update: %w", err)
	}

	roleID, err := resolveRoleID(ctx, tx, input.Role)
	if err != nil {
		return User{}, err
	}

	demotingActiveAdmin := currentRoleCode == "admin" && currentIsActive && input.Role != "admin"
	deactivatingActiveAdmin := currentRoleCode == "admin" && currentIsActive &&
		input.IsActive != nil && !*input.IsActive
	if demotingActiveAdmin || deactivatingActiveAdmin {
		if err := requireMoreThanOneActiveAdmin(ctx, tx); err != nil {
			return User{}, err
		}
	}

	if input.WarehouseID.Set && input.WarehouseID.Value != nil {
		if err := lockActiveWarehouse(ctx, tx, *input.WarehouseID.Value); err != nil {
			return User{}, err
		}
	}

	user, err := scanUser(tx.QueryRow(ctx, `
		UPDATE users
		SET full_name = $2,
		    role_id = $3,
		    warehouse_id = CASE WHEN $4 THEN $5::uuid ELSE warehouse_id END,
		    is_active = COALESCE($6, is_active),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, email, full_name, $7::text, warehouse_id::text, is_active, created_at, updated_at`,
		id, input.FullName, roleID,
		input.WarehouseID.Set, optionalStringValue(input.WarehouseID),
		nullableBool(input.IsActive), input.Role))
	if err != nil {
		return User{}, mapUsersWriteError("update user", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit update user: %w", err)
	}
	return user, nil
}

func (repository *PostgresRepository) DeactivateUser(ctx context.Context, id string) error {
	panic("not implemented until Task 8")
}

func (repository *PostgresRepository) ResetPassword(ctx context.Context, id string, passwordHash string) error {
	panic("not implemented until Task 9")
}
