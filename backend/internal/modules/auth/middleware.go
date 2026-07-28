package auth

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

const claimsContextKey = "bwims.auth.claims"

func Authenticate(tokens *TokenManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
		scheme, rawToken, found := strings.Cut(header, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(rawToken) == "" {
			return httpx.NewError(
				fiber.StatusUnauthorized,
				"UNAUTHENTICATED",
				"a Bearer access token is required",
				nil,
			)
		}

		claims, err := tokens.ParseAccessToken(strings.TrimSpace(rawToken))
		if err != nil {
			return httpx.NewError(
				fiber.StatusUnauthorized,
				"INVALID_ACCESS_TOKEN",
				"access token is invalid or expired",
				err,
			)
		}

		c.Locals(claimsContextKey, claims)
		return c.Next()
	}
}

func RequireRoles(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c fiber.Ctx) error {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			return httpx.NewError(
				fiber.StatusUnauthorized,
				"UNAUTHENTICATED",
				"authentication required",
				nil,
			)
		}
		if _, ok := allowed[claims.Role]; !ok {
			return httpx.NewError(
				fiber.StatusForbidden,
				"FORBIDDEN",
				"your role does not allow this action",
				nil,
			)
		}
		return c.Next()
	}
}

func ClaimsFromContext(c fiber.Ctx) (*Claims, bool) {
	claims, ok := c.Locals(claimsContextKey).(*Claims)
	return claims, ok && claims != nil
}
