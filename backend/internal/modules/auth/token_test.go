package auth

import (
	"testing"
	"time"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"
)

func testTokenManager(t *testing.T) *TokenManager {
	t.Helper()
	manager, err := NewTokenManager(config.AuthConfig{
		Issuer:               "bwims-test",
		AccessTokenTTL:       15 * time.Minute,
		RefreshTokenTTL:      24 * time.Hour,
		AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	return manager
}

func TestTokenManagerIssuesAndParsesRS256AccessToken(t *testing.T) {
	manager := testTokenManager(t)
	user := User{
		ID:       "56ce0a73-031c-48e9-bb5b-a0877462145c",
		Email:    "admin@bwims.local",
		FullName: "Admin",
		Role:     RoleAdmin,
		IsActive: true,
	}

	pair, err := manager.Issue(user)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if pair.RefreshToken == "" || pair.TokenType != "Bearer" {
		t.Fatalf("unexpected token pair: %+v", pair)
	}

	claims, err := manager.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.Subject != user.ID || claims.Role != RoleAdmin || claims.Email != user.Email {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTokenManagerRejectsTamperedToken(t *testing.T) {
	manager := testTokenManager(t)
	pair, err := manager.Issue(User{ID: "user-id", Role: RoleViewer})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	lastCharacter := pair.AccessToken[len(pair.AccessToken)-1]
	replacement := byte('A')
	if lastCharacter == replacement {
		replacement = 'B'
	}
	tampered := pair.AccessToken[:len(pair.AccessToken)-1] + string(replacement)
	if _, err := manager.ParseAccessToken(tampered); err == nil {
		t.Fatal("ParseAccessToken() error = nil, want tampered token error")
	}
}

func TestProductionRequiresConfiguredRSAKeys(t *testing.T) {
	_, err := NewTokenManager(config.AuthConfig{
		Issuer:          "bwims",
		AccessTokenTTL:  time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err == nil {
		t.Fatal("NewTokenManager() error = nil, want missing key error")
	}
}
