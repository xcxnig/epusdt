package route

import (
	"net/http"

	"github.com/assimon/luuu/controller/comm"
	"github.com/assimon/luuu/middleware"
	"github.com/labstack/echo/v4"
)

// RegisterRoute 路由注册
func RegisterRoute(e *echo.Echo) {
	e.Any("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello epusdt, https://github.com/GMwalletApp/epusdt")
	})
	// ==== 支付相关=====
	payRoute := e.Group("/legacy/pay")
	// 收银台
	payRoute.GET("/checkout-counter/:trade_id", comm.Ctrl.LegecyCheckoutCounter)
	// 状态检测
	payRoute.GET("/check-status/:trade_id", comm.Ctrl.LegecyCheckStatus)

	apiV1Route := e.Group("/legacy/api/v1")
	// ====订单相关====
	orderRoute := apiV1Route.Group("/order", middleware.LegecyCheckApiSign())
	// 创建订单
	orderRoute.POST("/create-transaction", comm.Ctrl.LegecyCreateTransaction)
}
