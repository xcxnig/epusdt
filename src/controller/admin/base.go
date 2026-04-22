package admin

import (
	"github.com/GMWalletApp/epusdt/controller"
	"github.com/GMWalletApp/epusdt/middleware"
	"github.com/labstack/echo/v4"
)

// Ctrl is the singleton admin controller. Keeps parity with comm.Ctrl.
var Ctrl = &BaseAdminController{}

// BaseAdminController aggregates admin-domain handlers. Split across
// *_controller.go files in this package; all receive methods hang off
// this type so a single router registration is enough.
type BaseAdminController struct {
	controller.BaseController
}

// currentAdminUserID returns the admin user id injected by CheckAdminJWT.
// Returns 0 if the middleware is missing (treat as auth bug upstream).
func currentAdminUserID(ctx echo.Context) uint64 {
	if v, ok := ctx.Get(middleware.AdminUserIDKey).(uint64); ok {
		return v
	}
	return 0
}
