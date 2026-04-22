package middleware

import (
	"net/http"
	"strings"

	appjwt "github.com/GMWalletApp/epusdt/util/jwt"
	"github.com/labstack/echo/v4"
)

const (
	// AdminUserIDKey is the echo.Context key holding the authenticated
	// admin user's ID after successful JWT validation.
	AdminUserIDKey = "admin_user_id"
	// AdminUsernameKey mirrors the username for convenience.
	AdminUsernameKey = "admin_username"
)

// CheckAdminJWT validates an Authorization: Bearer <jwt> header against
// the HS256 secret stored in system.jwt_secret. On success it injects
// the admin user ID and username into the echo context.
//
// Accepts both "Authorization: Bearer <jwt>" and a bare "<jwt>" for
// developer convenience (curl / postman). The token itself is always
// cryptographically validated — the relaxed header parsing doesn't
// widen the trust surface.
func CheckAdminJWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			raw := strings.TrimSpace(ctx.Request().Header.Get("Authorization"))
			if raw == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}
			if !strings.HasPrefix(raw, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "authorization header must use Bearer scheme")
			}
			token := strings.TrimSpace(raw[len("Bearer "):])
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "empty token")
			}
			claims, err := appjwt.Parse(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			ctx.Set(AdminUserIDKey, claims.AdminUserID)
			ctx.Set(AdminUsernameKey, claims.Username)
			return next(ctx)
		}
	}
}
