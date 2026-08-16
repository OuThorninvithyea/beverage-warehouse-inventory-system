package users

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func TestPostgresUsersLifecycle(t *testing.T) {
	databaseURL := os.Getenv("BWIMS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("BWIMS_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("database ping error = %v", err)
	}

	repository := NewPostgresRepository(pool)
	suffix := strings.ToUpper(fmt.Sprintf("%x", time.Now().UnixNano()))
	emailDomain := "users-test-" + strings.ToLower(suffix) + ".bwims.test"

	var warehouseID string
	err = pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, is_active) VALUES ($1, $2, TRUE) RETURNING id::text`,
		"WH-"+suffix, "Users Test Warehouse "+suffix).Scan(&warehouseID)
	if err != nil {
		t.Fatalf("seed warehouse error = %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE email LIKE $1)", "%"+emailDomain)
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE email LIKE $1", "%"+emailDomain)
		_, _ = pool.Exec(context.Background(), "DELETE FROM warehouses WHERE id = $1", warehouseID)
	})

	passwordHash := "$2a$10$abcdefghijklmnopqrstuvABCDEFGHIJKLMNOPQRSTUVWXYZ012345"

	admin1, err := repository.CreateUser(ctx, UserCreateInput{
		Email: "admin1@" + emailDomain, FullName: "Admin One", Role: auth.RoleAdmin,
	}, passwordHash)
	if err != nil {
		t.Fatalf("CreateUser(admin1) error = %v", err)
	}
	if admin1.ID == "" || admin1.Role != auth.RoleAdmin || admin1.WarehouseID != nil {
		t.Fatalf("admin1 = %#v, want a persisted admin with no warehouse", admin1)
	}

	warehouseIDCopy := warehouseID
	picker, err := repository.CreateUser(ctx, UserCreateInput{
		Email: "picker1@" + emailDomain, FullName: "Picker One", Role: auth.RolePicker,
		WarehouseID: OptionalString{Set: true, Value: &warehouseIDCopy},
	}, passwordHash)
	if err != nil {
		t.Fatalf("CreateUser(picker) error = %v", err)
	}
	if picker.WarehouseID == nil || *picker.WarehouseID != warehouseID {
		t.Fatalf("picker.WarehouseID = %v, want %s", picker.WarehouseID, warehouseID)
	}

	_, err = repository.CreateUser(ctx, UserCreateInput{
		Email: strings.ToUpper("admin1@" + emailDomain), FullName: "Duplicate", Role: auth.RolePicker,
	}, passwordHash)
	if !errors.Is(err, ErrEmailConflict) {
		t.Fatalf("duplicate email (case-insensitive) error = %v, want ErrEmailConflict", err)
	}

	missingWarehouse := "00000000-0000-0000-0000-000000000000"
	_, err = repository.CreateUser(ctx, UserCreateInput{
		Email: "nowhere@" + emailDomain, FullName: "Nowhere", Role: auth.RolePicker,
		WarehouseID: OptionalString{Set: true, Value: &missingWarehouse},
	}, passwordHash)
	if !errors.Is(err, ErrWarehouseNotFound) {
		t.Fatalf("missing warehouse error = %v, want ErrWarehouseNotFound", err)
	}

	_, err = repository.CreateUser(ctx, UserCreateInput{
		Email: "badrole@" + emailDomain, FullName: "Bad Role", Role: "supervisor",
	}, passwordHash)
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("invalid role error = %v, want ErrInvalidRole", err)
	}

	listed, err := repository.ListUsers(ctx, ListFilter{Limit: 10, Search: strings.ToLower(suffix)})
	if err != nil || len(listed) != 2 {
		t.Fatalf("ListUsers() = %#v, %v, want 2 users (admin1, picker)", listed, err)
	}

	adminOnly, err := repository.ListUsers(ctx, ListFilter{Limit: 10, Role: auth.RoleAdmin, Search: strings.ToLower(suffix)})
	if err != nil || len(adminOnly) != 1 || adminOnly[0].ID != admin1.ID {
		t.Fatalf("ListUsers(role=admin) = %#v, %v, want [admin1]", adminOnly, err)
	}
}
