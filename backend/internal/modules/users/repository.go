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
	panic("not implemented until Task 6")
}

func (repository *PostgresRepository) GetUser(ctx context.Context, id string) (User, error) {
	panic("not implemented until Task 7")
}

func (repository *PostgresRepository) UpdateUser(ctx context.Context, id string, input UserUpdateInput) (User, error) {
	panic("not implemented until Task 7")
}

func (repository *PostgresRepository) DeactivateUser(ctx context.Context, id string) error {
	panic("not implemented until Task 8")
}

func (repository *PostgresRepository) ResetPassword(ctx context.Context, id string, passwordHash string) error {
	panic("not implemented until Task 9")
}
