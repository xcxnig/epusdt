package comm

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/GMWalletApp/epusdt/config"
	"github.com/GMWalletApp/epusdt/model/response"
	"github.com/GMWalletApp/epusdt/model/service"
	"github.com/labstack/echo/v4"
)

// CheckoutCounter 收银台
// @Summary      Checkout counter page
// @Description  Render the payment checkout counter HTML page for a given trade
// @Tags         Payment
// @Produce      html
// @Param        trade_id path string true "Trade ID"
// @Success      200 {string} string "HTML page"
// @Router       /pay/checkout-counter/{trade_id} [get]
func (c *BaseCommController) CheckoutCounter(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	resp, err := service.GetCheckoutCounterByTradeId(tradeId)
	if err != nil {
		if err == service.ErrOrder {
			// Unknown trade id: render the page with empty payload
			// (client side shows a friendly "order not found" screen).
			tmpl, tmplErr := template.ParseFiles(filepath.Join(config.StaticFilePath, "index.html"))
			if tmplErr != nil {
				return ctx.String(http.StatusInternalServerError, tmplErr.Error())
			}
			ctx.Response().Status = http.StatusNotFound
			emptyResp := response.CheckoutCounterResponse{}
			return tmpl.Execute(ctx.Response(), emptyResp)
		}
		return ctx.String(http.StatusInternalServerError, err.Error())
	}
	tmpl, err := template.ParseFiles(filepath.Join(config.StaticFilePath, "index.html"))
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return tmpl.Execute(ctx.Response(), resp)
}

// CheckStatus 支付状态检测
// @Summary      Check payment status
// @Description  Check the payment status of an order by trade ID
// @Tags         Payment
// @Produce      json
// @Param        trade_id path string true "Trade ID"
// @Success      200 {object} response.ApiResponse{data=response.CheckStatusResponse}
// @Failure      400 {object} response.ApiResponse
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
