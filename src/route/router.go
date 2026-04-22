package route

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/GMWalletApp/epusdt/controller/admin"
	"github.com/GMWalletApp/epusdt/controller/comm"
	"github.com/GMWalletApp/epusdt/middleware"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/GMWalletApp/epusdt/util/constant"
	"github.com/GMWalletApp/epusdt/util/sign"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

// RegisterRoute 路由注册
func RegisterRoute(e *echo.Echo) {
	e.Any("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello epusdt, https://github.com/GMwalletApp/epusdt")
	})

	payRoute := e.Group("/pay")
	payRoute.GET("/checkout-counter/:trade_id", comm.Ctrl.CheckoutCounter)
	payRoute.GET("/check-status/:trade_id", comm.Ctrl.CheckStatus)
	payRoute.POST("/switch-network", comm.Ctrl.SwitchNetwork)

	// payment routes
	paymentRoute := e.Group("/payments")

	// gmpay v1 routes
	gmpayV1 := paymentRoute.Group("/gmpay/v1")
	gmpayV1.POST("/order/create-transaction", comm.Ctrl.CreateTransaction, middleware.CheckApiSign())
	gmpayV1.GET("/supported-assets", comm.Ctrl.GetSupportedAssets)
	// gmpayV1.GET("/supported-assets/records", comm.Ctrl.ListSupportedAssetRecords)
	// gmpayV1.GET("/supported-assets/:id", comm.Ctrl.GetSupportedAsset)

	// epay v1 routes
	//
	// Signature uses the pid from the request as the api_keys lookup
	// key; the matching row's secret_key plays the role of the legacy
	// EPAY "key" value. Env-based (EPAY_PID/EPAY_KEY) fallback was
	// removed — the default seeded api_key is always available, and
	// having two sources of truth led to inbound/outbound sig mismatch.
	epayV1 := paymentRoute.Group("/epay/v1")
	epayV1.Match([]string{http.MethodPost, http.MethodGet}, "/order/create-transaction/submit.php", func(ctx echo.Context) error {
		params := make(map[string]interface{})
		copyParams := func(values map[string][]string) {
			for k, v := range values {
				if len(v) == 0 {
					continue
				}
				params[k] = v[0]
			}
		}

		copyParams(ctx.QueryParams())

		formParams, err := ctx.FormParams()
		if err != nil && ctx.Request().Method == http.MethodPost {
			return comm.Ctrl.FailJson(ctx, fmt.Errorf("invalid epay form params: %w", err))
		}
		if err == nil {
			copyParams(formParams)
		}

		getString := func(m map[string]interface{}, key string) string {
			v, ok := m[key]
			if !ok {
				return ""
			}
			switch t := v.(type) {
			case string:
				return t
			}
			return ""
		}

		signstr := getString(params, "sign")
		if signstr == "" {
			return constant.SignatureErr
		}

		delete(params, "sign")
		delete(params, "sign_type")

		pidStr := getString(params, "pid")
		if pidStr == "" {
			return constant.SignatureErr
		}
		apiKeyRow, err := data.GetEnabledApiKey(pidStr)
		if err != nil || apiKeyRow == nil || apiKeyRow.ID == 0 {
			return constant.SignatureErr
		}

		checkSignature, err := sign.Get(params, apiKeyRow.SecretKey)
		if err != nil {
			return constant.SignatureErr
		}
		if subtle.ConstantTimeCompare([]byte(checkSignature), []byte(signstr)) != 1 {
			return constant.SignatureErr
		}

		if !middleware.IsIPWhitelisted(apiKeyRow.IpWhitelist, ctx.RealIP()) {
			return constant.SignatureErr
		}

		_ = data.TouchApiKeyUsage(apiKeyRow.ID)

		money := getString(params, "money")
		name := getString(params, "name")
		notifyURL := getString(params, "notify_url")
		outTradeNo := getString(params, "out_trade_no")
		returnURL := getString(params, "return_url")

		amountFloat, err := strconv.ParseFloat(money, 64)
		if err != nil {
			return comm.Ctrl.FailJson(ctx, fmt.Errorf("invalid money value: %s", money))
		}

		body := map[string]interface{}{
			"token":        data.GetSettingString(mdb.SettingKeyEpayDefaultToken, "usdt"),
			"currency":     data.GetSettingString(mdb.SettingKeyEpayDefaultCurrency, "cny"),
			"network":      data.GetSettingString(mdb.SettingKeyEpayDefaultNetwork, "tron"),
			"amount":       amountFloat,
			"notify_url":   notifyURL,
			"order_id":     outTradeNo,
			"redirect_url": returnURL,
			"signature":    signstr,
			"name":         name,
			"payment_type": mdb.PaymentTypeEpay,
		}

		ctx.Set("request_body", body)
		ctx.Set(middleware.ApiKeyIDKey, apiKeyRow.ID)
		ctx.Set(middleware.ApiKeyRowKey, apiKeyRow)

		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return comm.Ctrl.FailJson(ctx, err)
		}

		ctx.Request().Body = io.NopCloser(bytes.NewBuffer(jsonBytes))
		ctx.Request().ContentLength = int64(len(jsonBytes))
		ctx.Request().Method = http.MethodPost
		ctx.Request().Header.Set("Content-Type", "application/json")

		return comm.Ctrl.CreateTransactionAndRedirect(ctx)
	})

	registerAdminRoutes(e)
}

// registerAdminRoutes wires the management console API surface under
// /admin/api/v1. Everything except /auth/login requires a valid JWT.
func registerAdminRoutes(e *echo.Echo) {
	// CORS for the management console. The admin SPA is commonly served
	// from a different origin (local dev, CDN, etc.), so allow any origin
	// but require explicit echoing — browsers refuse wildcard + credentials.
	adminCORS := echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) { return true, nil },
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPatch,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAuthorization,
			echo.HeaderAccept,
			echo.HeaderXRequestedWith,
		},
		AllowCredentials: true,
		MaxAge:           86400,
	})

	adminV1 := e.Group("/admin/api/v1", adminCORS)

	// Public (no JWT)
	adminV1.POST("/auth/login", admin.Ctrl.Login)
	adminV1.GET("/auth/init-password", admin.Ctrl.GetInitialPassword)
	adminV1.GET("/auth/init-password-hash", admin.Ctrl.GetInitialPasswordHash)

	// Authenticated
	authed := adminV1.Group("", middleware.CheckAdminJWT())
	authed.POST("/auth/logout", admin.Ctrl.Logout)
	authed.GET("/auth/me", admin.Ctrl.Me)
	authed.POST("/auth/password", admin.Ctrl.ChangePassword)

	// API key management
	authed.GET("/api-keys", admin.Ctrl.ListApiKeys)
	authed.POST("/api-keys", admin.Ctrl.CreateApiKey)
	authed.PATCH("/api-keys/:id", admin.Ctrl.UpdateApiKey)
	authed.POST("/api-keys/:id/status", admin.Ctrl.ChangeApiKeyStatus)
	authed.POST("/api-keys/:id/rotate-secret", admin.Ctrl.RotateApiKeySecret)
	authed.DELETE("/api-keys/:id", admin.Ctrl.DeleteApiKey)
	authed.GET("/api-keys/:id/stats", admin.Ctrl.GetApiKeyStats)
	authed.GET("/api-keys/:id/secret", admin.Ctrl.GetApiKeySecret)

	// Notification channels
	authed.GET("/notification-channels", admin.Ctrl.ListNotificationChannels)
	authed.POST("/notification-channels", admin.Ctrl.CreateNotificationChannel)
	authed.PATCH("/notification-channels/:id", admin.Ctrl.UpdateNotificationChannel)
	authed.POST("/notification-channels/:id/status", admin.Ctrl.ChangeNotificationChannelStatus)
	authed.DELETE("/notification-channels/:id", admin.Ctrl.DeleteNotificationChannel)

	//	gmpayV1.GET("/supported-assets", comm.Ctrl.GetSupportedAssets)
	authed.GET("/supported-assets", comm.Ctrl.GetSupportedAssets) // wrap for admin console, same handler as public endpoint

	// Chains
	authed.GET("/chains", admin.Ctrl.ListChains)
	authed.PATCH("/chains/:network", admin.Ctrl.UpdateChain)

	// Chain tokens (per-chain token catalog)
	authed.GET("/chain-tokens", admin.Ctrl.ListChainTokens)
	authed.POST("/chain-tokens", admin.Ctrl.CreateChainToken)
	authed.PATCH("/chain-tokens/:id", admin.Ctrl.UpdateChainToken)
	authed.POST("/chain-tokens/:id/status", admin.Ctrl.ChangeChainTokenStatus)
	authed.DELETE("/chain-tokens/:id", admin.Ctrl.DeleteChainToken)

	// RPC nodes
	authed.GET("/rpc-nodes", admin.Ctrl.ListRpcNodes)
	authed.POST("/rpc-nodes", admin.Ctrl.CreateRpcNode)
	authed.PATCH("/rpc-nodes/:id", admin.Ctrl.UpdateRpcNode)
	authed.DELETE("/rpc-nodes/:id", admin.Ctrl.DeleteRpcNode)
	authed.POST("/rpc-nodes/:id/health-check", admin.Ctrl.HealthCheckRpcNode)

	// Wallet (address) management
	authed.GET("/wallets", admin.Ctrl.AdminListWallets)
	authed.POST("/wallets", admin.Ctrl.AdminAddWallet)
	authed.GET("/wallets/:id", admin.Ctrl.AdminGetWallet)
	authed.PATCH("/wallets/:id", admin.Ctrl.AdminUpdateWallet)
	authed.POST("/wallets/:id/status", admin.Ctrl.AdminChangeWalletStatus)
	authed.DELETE("/wallets/:id", admin.Ctrl.AdminDeleteWallet)
	authed.POST("/wallets/batch-import", admin.Ctrl.AdminBatchImportWallets)

	// Orders
	authed.GET("/orders", admin.Ctrl.ListOrders)
	authed.GET("/orders/export", admin.Ctrl.ExportOrders)
	authed.GET("/orders/list-with-sub", admin.Ctrl.ListOrdersWithSub)
	authed.GET("/orders/:trade_id", admin.Ctrl.GetOrder)
	authed.GET("/orders/:trade_id/with-sub", admin.Ctrl.GetOrderWithSub)
	authed.POST("/orders/:trade_id/close", admin.Ctrl.CloseOrder)
	authed.POST("/orders/:trade_id/mark-paid", admin.Ctrl.MarkOrderPaid)
	authed.POST("/orders/:trade_id/resend-callback", admin.Ctrl.ResendCallback)

	// Dashboard
	authed.GET("/dashboard/overview", admin.Ctrl.Overview)
	authed.GET("/dashboard/asset-trend", admin.Ctrl.AssetTrend)
	authed.GET("/dashboard/revenue-trend", admin.Ctrl.RevenueTrend)
	authed.GET("/dashboard/order-stats", admin.Ctrl.OrderStats)
	authed.GET("/dashboard/recent-orders", admin.Ctrl.RecentOrders)

	// Settings
	authed.GET("/settings", admin.Ctrl.ListSettings)
	authed.PUT("/settings", admin.Ctrl.UpsertSettings)
	authed.DELETE("/settings/:key", admin.Ctrl.DeleteSetting)
}
