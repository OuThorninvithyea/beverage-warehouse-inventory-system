package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"
)

type fakeRepository struct {
	user             User
	findErr          error
	storedUserID     string
	storedHash       string
	rotateOldHash    string
	rotateNewHash    string
	rotatedUser      User
	rotateErr        error
	revokedTokenHash string
}

func (r *fakeRepository) FindActiveUserByEmail(context.Context, string) (User, error) {
	return r.user, r.findErr
}

func (r *fakeRepository) StoreRefreshToken(
	_ context.Context,
	userID string,
	tokenHash string,
	_ time.Time,
) error {
	r.storedUserID = userID
	r.storedHash = tokenHash
	return nil
}

func (r *fakeRepository) RotateRefreshToken(
	_ context.Context,
	oldHash string,
	newHash string,
	_ time.Time,
) (User, error) {
	r.rotateOldHash = oldHash
	r.rotateNewHash = newHash
	return r.rotatedUser, r.rotateErr
}

func (r *fakeRepository) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	r.revokedTokenHash = tokenHash
	return nil
}

func newTestService(t *testing.T, repository Repository) Service {
	t.Helper()
	tokens, err := NewTokenManager(config.AuthConfig{
		Issuer:               "bwims-test",
		AccessTokenTTL:       time.Minute,
		RefreshTokenTTL:      time.Hour,
		AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	return NewService(repository, tokens)
}

func TestServiceLoginStoresHashedRefreshToken(t *testing.T) {
	password := "CorrectHorse1!"
	passwordHash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	repository := &fakeRepository{user: User{
		ID:           "user-id",
		Email:        "admin@bwims.local",
		PasswordHash: passwordHash,
		FullName:     "Admin",
		Role:         RoleAdmin,
		IsActive:     true,
	}}
	service := newTestService(t, repository)

	pair, err := service.Login(context.Background(), repository.user.Email, password)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if repository.storedUserID != repository.user.ID {
		t.Fatalf("stored user ID = %q", repository.storedUserID)
	}
	if repository.storedHash != HashRefreshToken(pair.RefreshToken) {
		t.Fatal("repository did not receive hashed refresh token")
	}
	if repository.storedHash == pair.RefreshToken {
		t.Fatal("raw refresh token was stored")
	}
}

func TestServiceLoginUsesGenericCredentialError(t *testing.T) {
	service := newTestService(t, &fakeRepository{findErr: ErrUserNotFound})
	if _, err := service.Login(context.Background(), "missing@example.com", "anything"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceRotatesRefreshToken(t *testing.T) {
	repository := &fakeRepository{rotatedUser: User{
		ID:       "user-id",
		Email:    "picker@bwims.local",
		FullName: "Picker",
		Role:     RolePicker,
		IsActive: true,
	}}
	service := newTestService(t, repository)

	pair, err := service.Refresh(context.Background(), "old-refresh-token")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if repository.rotateOldHash != HashRefreshToken("old-refresh-token") {
		t.Fatal("old refresh token was not hashed")
	}
	if repository.rotateNewHash != HashRefreshToken(pair.RefreshToken) {
		t.Fatal("new refresh token was not atomically persisted")
	}
	if pair.User.Role != RolePicker {
		t.Fatalf("pair user role = %q", pair.User.Role)
	}
}

func TestHashPasswordRequiresTwelveCharacters(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("HashPassword() error = nil, want length error")
	}
}
