package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid or expired")
)

type Repository interface {
	FindActiveUserByEmail(context.Context, string) (User, error)
	StoreRefreshToken(context.Context, string, string, time.Time) error
	RotateRefreshToken(context.Context, string, string, time.Time) (User, error)
	RevokeRefreshToken(context.Context, string) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const userColumns = `
	SELECT u.id::text, u.email, u.password_hash, u.full_name, r.code,
	       u.warehouse_id::text, u.is_active
	FROM users u
	JOIN roles r ON r.id = u.role_id`

func (r *PostgresRepository) FindActiveUserByEmail(ctx context.Context, email string) (User, error) {
	row := r.pool.QueryRow(ctx, userColumns+`
		WHERE LOWER(u.email) = LOWER($1) AND u.is_active = TRUE`, email)

	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("find active user: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) StoreRefreshToken(
	ctx context.Context,
	userID string,
	tokenHash string,
	expiresAt time.Time,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RotateRefreshToken(
	ctx context.Context,
	oldHash string,
	newHash string,
	newExpiresAt time.Time,
) (User, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return User{}, fmt.Errorf("begin refresh rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		RETURNING user_id::text`, oldHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrRefreshTokenInvalid
	}
	if err != nil {
		return User{}, fmt.Errorf("consume refresh token: %w", err)
	}

	user, err := scanUser(tx.QueryRow(ctx, userColumns+`
		WHERE u.id = $1 AND u.is_active = TRUE`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrRefreshTokenInvalid
	}
	if err != nil {
		return User{}, fmt.Errorf("find refresh user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, user.ID, newHash, newExpiresAt); err != nil {
		return User{}, fmt.Errorf("store rotated refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit refresh rotation: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRefreshTokenInvalid
	}
	return nil
}

func (r *PostgresRepository) EnsureDevelopmentAdmin(
	ctx context.Context,
	email string,
	passwordHash string,
	fullName string,
) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO users (role_id, email, password_hash, full_name)
		SELECT id, $1, $2, $3
		FROM roles
		WHERE code = 'admin'
		ON CONFLICT ((LOWER(email))) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    full_name = EXCLUDED.full_name,
		    role_id = EXCLUDED.role_id,
		    is_active = TRUE,
		    updated_at = NOW()`, email, passwordHash, fullName)
	if err != nil {
		return fmt.Errorf("seed development admin: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("seed development admin: admin role is missing")
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.WarehouseID,
		&user.IsActive,
	)
	return user, err
}
