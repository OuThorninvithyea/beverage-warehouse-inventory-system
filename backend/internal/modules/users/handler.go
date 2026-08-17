package users

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func usersActor(c fiber.Ctx) Actor {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok || claims == nil {
		return Actor{}
	}
	return Actor{ID: claims.Subject, Role: claims.Role}
}

func invalidUsersRequest(message string, err error) error {
	return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", message, err)
}

func usersHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidCursor):
		return invalidUsersRequest("a resource identifier, filter, or cursor is invalid", err)
	case errors.Is(err, ErrForbidden):
		return httpx.NewError(fiber.StatusForbidden, "FORBIDDEN", "your role does not allow this action", err)
	case errors.Is(err, ErrUserNotFound):
		return httpx.NewError(fiber.StatusNotFound, "USER_NOT_FOUND", "user was not found", err)
	case errors.Is(err, ErrWarehouseNotFound):
		return httpx.NewError(fiber.StatusNotFound, "WAREHOUSE_NOT_FOUND", "warehouse was not found or is not active", err)
	case errors.Is(err, ErrEmailConflict):
		return httpx.NewError(fiber.StatusConflict, "EMAIL_CONFLICT", "email is already in use", err)
	case errors.Is(err, ErrLastAdminProtected):
		return httpx.NewError(fiber.StatusConflict, "LAST_ADMIN_PROTECTED", "this action would leave zero active administrators", err)
	case errors.Is(err, ErrSelfDeactivationForbidden):
		return httpx.NewError(fiber.StatusConflict, "SELF_DEACTIVATION_FORBIDDEN", "an administrator cannot deactivate or demote themselves", err)
	case errors.Is(err, ErrInvalidRole):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "INVALID_ROLE", "role is not a known role code", err)
	case errors.Is(err, ErrValidation):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "user data is invalid", err)
	default:
		return httpx.NewError(fiber.StatusInternalServerError, "USER_OPERATION_FAILED", "user operation could not be completed", err)
	}
}

type createUserRequest struct {
	Email       string         `json:"email"`
	FullName    string         `json:"full_name"`
	Role        string         `json:"role"`
	WarehouseID OptionalString `json:"warehouse_id"`
	Password    string         `json:"password"`
}

type updateUserRequest struct {
	FullName    string         `json:"full_name"`
	Role        string         `json:"role"`
	WarehouseID OptionalString `json:"warehouse_id"`
	IsActive    *bool          `json:"is_active"`
}

type passwordResetRequest struct {
	Password string `json:"password"`
}

func bindCreateUserRequest(c fiber.Ctx) (UserCreateInput, error) {
	var body createUserRequest
	if err := c.Bind().Body(&body); err != nil {
		return UserCreateInput{}, invalidUsersRequest("request body could not be parsed", err)
	}
	return UserCreateInput{
		Email: body.Email, FullName: body.FullName, Role: body.Role,
		WarehouseID: body.WarehouseID, Password: body.Password,
	}, nil
}

func bindUpdateUserRequest(c fiber.Ctx) (UserUpdateInput, error) {
	var body updateUserRequest
	if err := c.Bind().Body(&body); err != nil {
		return UserUpdateInput{}, invalidUsersRequest("request body could not be parsed", err)
	}
	return UserUpdateInput{
		FullName: body.FullName, Role: body.Role, WarehouseID: body.WarehouseID, IsActive: body.IsActive,
	}, nil
}

func bindPasswordResetRequest(c fiber.Ctx) (PasswordResetInput, error) {
	var body passwordResetRequest
	if err := c.Bind().Body(&body); err != nil {
		return PasswordResetInput{}, invalidUsersRequest("request body could not be parsed", err)
	}
	return PasswordResetInput{Password: body.Password}, nil
}

func usersOptionalBool(raw string) (*bool, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func parseUsersListFilter(c fiber.Ctx) (ListFilter, error) {
	filter := ListFilter{Search: c.Query("search"), Role: c.Query("role")}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return ListFilter{}, invalidUsersRequest("limit must be an integer from 1 to 100", nil)
		}
		filter.Limit = limit
	}
	if raw := c.Query("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return ListFilter{}, invalidUsersRequest("after cursor is invalid", err)
		}
		filter.After = &cursor
	}
	if raw := c.Query("warehouse_id"); raw != "" {
		filter.WarehouseID = &raw
	}
	active, err := usersOptionalBool(c.Query("is_active"))
	if err != nil {
		return ListFilter{}, invalidUsersRequest("is_active must be true or false", err)
	}
	filter.IsActive = active
	return filter, nil
}

func (h *Handler) ListUsers(c fiber.Ctx) error {
	filter, err := parseUsersListFilter(c)
	if err != nil {
		return err
	}
	page, err := h.service.ListUsers(c.Context(), usersActor(c), filter)
	if err != nil {
		return usersHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) CreateUser(c fiber.Ctx) error {
	input, err := bindCreateUserRequest(c)
	if err != nil {
		return err
	}
	user, err := h.service.CreateUser(c.Context(), usersActor(c), input)
	if err != nil {
		return usersHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(user))
}

func (h *Handler) GetUser(c fiber.Ctx) error {
	user, err := h.service.GetUser(c.Context(), usersActor(c), c.Params("user_id"))
	if err != nil {
		return usersHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(user))
}

func (h *Handler) UpdateUser(c fiber.Ctx) error {
	input, err := bindUpdateUserRequest(c)
	if err != nil {
		return err
	}
	user, err := h.service.UpdateUser(c.Context(), usersActor(c), c.Params("user_id"), input)
	if err != nil {
		return usersHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(user))
}

func (h *Handler) DeactivateUser(c fiber.Ctx) error {
	if err := h.service.DeactivateUser(c.Context(), usersActor(c), c.Params("user_id")); err != nil {
		return usersHTTPError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ResetPassword(c fiber.Ctx) error {
	input, err := bindPasswordResetRequest(c)
	if err != nil {
		return err
	}
	if err := h.service.ResetPassword(c.Context(), usersActor(c), c.Params("user_id"), input); err != nil {
		return usersHTTPError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
