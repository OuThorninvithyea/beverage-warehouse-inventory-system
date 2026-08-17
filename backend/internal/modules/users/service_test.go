package users

import (
	"context"
	"errors"
	"testing"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

type fakeRepository struct {
	createFn     func(UserCreateInput, string) (User, error)
	listFn       func(ListFilter) ([]User, error)
	getFn        func(string) (User, error)
	updateFn     func(string, UserUpdateInput) (User, error)
	deactivateFn func(string) error
	resetFn      func(string, string) error
}

func (f *fakeRepository) CreateUser(_ context.Context, input UserCreateInput, hash string) (User, error) {
	return f.createFn(input, hash)
}
func (f *fakeRepository) ListUsers(_ context.Context, filter ListFilter) ([]User, error) {
	return f.listFn(filter)
}
func (f *fakeRepository) GetUser(_ context.Context, id string) (User, error) {
	return f.getFn(id)
}
func (f *fakeRepository) UpdateUser(_ context.Context, id string, input UserUpdateInput) (User, error) {
	return f.updateFn(id, input)
}
func (f *fakeRepository) DeactivateUser(_ context.Context, id string) error {
	return f.deactivateFn(id)
}
func (f *fakeRepository) ResetPassword(_ context.Context, id string, hash string) error {
	return f.resetFn(id, hash)
}

func adminActor() Actor { return Actor{ID: "admin-1", Role: auth.RoleAdmin} }

func TestCreateUserRejectsNonAdmin(t *testing.T) {
	service := NewService(&fakeRepository{})
	for _, role := range []string{auth.RoleWarehouseManager, auth.RolePicker, auth.RoleViewer} {
		_, err := service.CreateUser(context.Background(), Actor{ID: "x", Role: role}, UserCreateInput{
			Email: "a@bwims.test", FullName: "A", Role: auth.RolePicker, Password: "at-least-12-chars",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
		}
	}
}

func TestCreateUserValidatesRequiredFields(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.CreateUser(context.Background(), adminActor(), UserCreateInput{
		Email: "", FullName: "A", Role: auth.RolePicker, Password: "at-least-12-chars",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestCreateUserRejectsUnknownRole(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.CreateUser(context.Background(), adminActor(), UserCreateInput{
		Email: "a@bwims.test", FullName: "A", Role: "supervisor", Password: "at-least-12-chars",
	})
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("err = %v, want ErrInvalidRole", err)
	}
}

func TestCreateUserRejectsShortPassword(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.CreateUser(context.Background(), adminActor(), UserCreateInput{
		Email: "a@bwims.test", FullName: "A", Role: auth.RolePicker, Password: "short",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestCreateUserPassesHashedPasswordToRepository(t *testing.T) {
	var capturedHash string
	repo := &fakeRepository{
		createFn: func(input UserCreateInput, hash string) (User, error) {
			capturedHash = hash
			return User{ID: "new-1", Email: input.Email, Role: input.Role}, nil
		},
	}
	service := NewService(repo)
	user, err := service.CreateUser(context.Background(), adminActor(), UserCreateInput{
		Email: "A@BWIMS.test", FullName: "A", Role: auth.RolePicker, Password: "at-least-12-chars",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if capturedHash == "" || capturedHash == "at-least-12-chars" {
		t.Fatalf("capturedHash = %q, want a bcrypt hash, not the raw password", capturedHash)
	}
	if user.Email != "a@bwims.test" {
		t.Fatalf("Email = %q, want lower-cased email", user.Email)
	}
}

func TestGetUserRejectsInvalidID(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.GetUser(context.Background(), adminActor(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestListUsersRejectsUnknownRoleFilter(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.ListUsers(context.Background(), adminActor(), ListFilter{Role: "supervisor"})
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("err = %v, want ErrInvalidRole", err)
	}
}

func TestListUsersAppliesDefaultAndOverflowLimit(t *testing.T) {
	var capturedLimit int
	repo := &fakeRepository{
		listFn: func(filter ListFilter) ([]User, error) {
			capturedLimit = filter.Limit
			return []User{}, nil
		},
	}
	service := NewService(repo)
	if _, err := service.ListUsers(context.Background(), adminActor(), ListFilter{}); err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if capturedLimit != 21 {
		t.Fatalf("capturedLimit = %d, want 21 (default 20 + 1 overflow probe)", capturedLimit)
	}
}
