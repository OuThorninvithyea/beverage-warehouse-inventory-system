package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type Service interface {
	Login(context.Context, string, string) (TokenPair, error)
	Refresh(context.Context, string) (TokenPair, error)
	Logout(context.Context, string) error
}

type service struct {
	repository Repository
	tokens     *TokenManager
}

func NewService(repository Repository, tokens *TokenManager) Service {
	return &service{repository: repository, tokens: tokens}
}

func (s *service) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := s.repository.FindActiveUserByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, ErrUserNotFound) {
		return TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return TokenPair{}, fmt.Errorf("find login user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	pair, err := s.tokens.Issue(user)
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.repository.StoreRefreshToken(
		ctx,
		user.ID,
		HashRefreshToken(pair.RefreshToken),
		pair.RefreshTokenExpiresAt,
	); err != nil {
		return TokenPair{}, err
	}
	return pair, nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return TokenPair{}, ErrRefreshTokenInvalid
	}

	newRefreshToken, refreshExpiresAt, err := s.tokens.NewRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	user, err := s.repository.RotateRefreshToken(
		ctx,
		HashRefreshToken(refreshToken),
		HashRefreshToken(newRefreshToken),
		refreshExpiresAt,
	)
	if err != nil {
		return TokenPair{}, err
	}

	accessToken, accessExpiresAt, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          newRefreshToken,
		TokenType:             "Bearer",
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
		User:                  user.Public(),
	}, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return ErrRefreshTokenInvalid
	}
	return s.repository.RevokeRefreshToken(ctx, HashRefreshToken(refreshToken))
}

func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", fmt.Errorf("password must contain at least 12 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}
