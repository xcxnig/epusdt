package comm

import (
	"github.com/GMWalletApp/epusdt/model/response"
	"github.com/GMWalletApp/epusdt/model/service"
	"github.com/labstack/echo/v4"
)

// CheckoutCounter 收银台
// @Summary      Checkout counter page
// @Description  Return checkout initialization data when the order exists. This endpoint only confirms order existence and returns base order data; call /pay/check-status/{trade_id} for the current order status (1=waiting payment, 2=paid, 3=expired).
// @Tags         Payment
// @Produce      json
// @Param        trade_id path string true "Trade ID"
// @Success      200 {object} response.ApiResponse{data=response.CheckoutCounterResponse}
// @Failure      400 {object} response.ApiResponse "Order not found (status_code=10008) or other request error"
// @Router       /pay/checkout-counter-resp/{trade_id} [get]
func (c *BaseCommController) CheckoutCounter(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	resp, err := service.GetCheckoutCounterByTradeId(tradeId)
	if err != nil {
		return c.FailJson(ctx, err)
	}

	return c.SucJson(ctx, resp)
}

// CheckStatus 支付状态检测
// @Summary      Check payment status
// @Description  Return the current order status by trade ID. Status: 1=waiting payment, 2=paid, 3=expired.
// @Tags         Payment
// @Produce      json
// @Param        trade_id path string true "Trade ID"
// @Success      200 {object} response.ApiResponse{data=response.CheckStatusResponse}
// @Failure      400 {object} response.ApiResponse "Order not found (status_code=10008) or other request error"
// @Router       /pay/check-status/{trade_id} [get]
func (c *BaseCommController) CheckStatus(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	order, err := service.GetOrderInfoByTradeId(tradeId)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	resp := response.CheckStatusResponse{
		TradeId: order.TradeId,
		Status:  order.Status,
	}
	return c.SucJson(ctx, resp)
}
