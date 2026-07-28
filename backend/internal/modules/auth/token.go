package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"
)

type Claims struct {
	Email       string  `json:"email"`
	FullName    string  `json:"full_name"`
	Role        string  `json:"role"`
	WarehouseID *string `json:"warehouse_id,omitempty"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewTokenManager(cfg config.AuthConfig) (*TokenManager, error) {
	privateKey, publicKey, err := loadRSAKeys(cfg)
	if err != nil {
		return nil, err
	}

	return &TokenManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     cfg.Issuer,
		accessTTL:  cfg.AccessTokenTTL,
		refreshTTL: cfg.RefreshTokenTTL,
		now:        time.Now,
	}, nil
}

func loadRSAKeys(cfg config.AuthConfig) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	if cfg.PrivateKeyBase64 == "" || cfg.PublicKeyBase64 == "" {
		if !cfg.AllowDevelopmentKeys {
			return nil, nil, fmt.Errorf("JWT_PRIVATE_KEY_BASE64 and JWT_PUBLIC_KEY_BASE64 are required")
		}

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, nil, fmt.Errorf("generate development RSA key: %w", err)
		}
		return privateKey, &privateKey.PublicKey, nil
	}

	privatePEM, err := base64.StdEncoding.DecodeString(cfg.PrivateKeyBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("decode JWT private key: %w", err)
	}
	publicPEM, err := base64.StdEncoding.DecodeString(cfg.PublicKeyBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("decode JWT public key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse JWT private key: %w", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse JWT public key: %w", err)
	}
	return privateKey, publicKey, nil
}

func (m *TokenManager) Issue(user User) (TokenPair, error) {
	now := m.now().UTC()
	accessExpiry := now.Add(m.accessTTL)
	accessToken, err := m.issueAccessToken(user, now, accessExpiry)
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, refreshExpiry, err := m.NewRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		TokenType:             "Bearer",
		AccessTokenExpiresAt:  accessExpiry,
		RefreshTokenExpiresAt: refreshExpiry,
		User:                  user.Public(),
	}, nil
}

func (m *TokenManager) IssueAccessToken(user User) (string, time.Time, error) {
	now := m.now().UTC()
	expiry := now.Add(m.accessTTL)
	token, err := m.issueAccessToken(user, now, expiry)
	return token, expiry, err
}

func (m *TokenManager) issueAccessToken(user User, now, expiry time.Time) (string, error) {
	claims := Claims{
		Email:       user.Email,
		FullName:    user.FullName,
		Role:        user.Role,
		WarehouseID: user.WarehouseID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.ID,
			Audience:  jwt.ClaimStrings{"bwims-web"},
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(m.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return accessToken, nil
}

func (m *TokenManager) NewRefreshToken() (string, time.Time, error) {
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(refreshBytes), m.now().UTC().Add(m.refreshTTL), nil
}

func (m *TokenManager) ParseAccessToken(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		raw,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodRS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return m.publicKey, nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience("bwims-web"),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	if err != nil {
		return nil, fmt.Errorf("validate access token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.Subject == "" {
		return nil, fmt.Errorf("invalid access token claims")
	}
	return claims, nil
}

func HashRefreshToken(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
