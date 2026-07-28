package auth

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Login(c fiber.Ctx) error {
	var request loginRequest
	if err := c.Bind().Body(&request); err != nil {
		return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", "invalid JSON request", err)
	}
	if strings.TrimSpace(request.Email) == "" || request.Password == "" {
		return httpx.NewError(
			fiber.StatusUnprocessableEntity,
			"VALIDATION_ERROR",
			"email and password are required",
			nil,
		)
	}

	pair, err := h.service.Login(c.Context(), request.Email, request.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		return httpx.NewError(
			fiber.StatusUnauthorized,
			"INVALID_CREDENTIALS",
			"email or password is incorrect",
			err,
		)
	}
	if err != nil {
		return httpx.NewError(
			fiber.StatusInternalServerError,
			"AUTHENTICATION_FAILED",
			"authentication could not be completed",
			err,
		)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(pair))
}

func (h *Handler) Refresh(c fiber.Ctx) error {
	var request refreshRequest
	if err := c.Bind().Body(&request); err != nil {
		return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", "invalid JSON request", err)
	}

	pair, err := h.service.Refresh(c.Context(), request.RefreshToken)
	if errors.Is(err, ErrRefreshTokenInvalid) {
		return httpx.NewError(
			fiber.StatusUnauthorized,
			"INVALID_REFRESH_TOKEN",
			"refresh token is invalid or expired",
			err,
		)
	}
	if err != nil {
		return httpx.NewError(
			fiber.StatusInternalServerError,
			"TOKEN_REFRESH_FAILED",
			"token refresh could not be completed",
			err,
		)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(pair))
}

func (h *Handler) Logout(c fiber.Ctx) error {
	var request refreshRequest
	if err := c.Bind().Body(&request); err != nil {
		return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", "invalid JSON request", err)
	}

	if err := h.service.Logout(c.Context(), request.RefreshToken); err != nil &&
		!errors.Is(err, ErrRefreshTokenInvalid) {
		return httpx.NewError(
			fiber.StatusInternalServerError,
			"LOGOUT_FAILED",
			"logout could not be completed",
			err,
		)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Me(c fiber.Ctx) error {
	claims, ok := ClaimsFromContext(c)
	if !ok {
		return httpx.NewError(fiber.StatusUnauthorized, "UNAUTHENTICATED", "authentication required", nil)
	}

	return c.Status(fiber.StatusOK).JSON(httpx.Success(PublicUser{
		ID:          claims.Subject,
		Email:       claims.Email,
		FullName:    claims.FullName,
		Role:        claims.Role,
		WarehouseID: claims.WarehouseID,
	}))
}

func (h *Handler) AdminPing(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(httpx.Success(map[string]string{
		"status": "allowed",
		"role":   RoleAdmin,
	}))
}
