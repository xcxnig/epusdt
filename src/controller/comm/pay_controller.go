package comm

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model/response"
	"github.com/assimon/luuu/model/service"
	"github.com/labstack/echo/v4"
)

// LegecyCheckoutCounter 收银台
func (c *BaseCommController) LegecyCheckoutCounter(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	resp, err := service.GetCheckoutCounterByTradeId(tradeId)
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}
	tmpl, err := template.ParseFiles(filepath.Join(config.StaticFilePath, "index.html"))
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}
	resp.Token = resp.ReceiveAddress // only for legacy checkout counter, token is the receive address
	return tmpl.Execute(ctx.Response(), resp)
}

// LegecyCheckStatus 支付状态检测
func (c *BaseCommController) LegecyCheckStatus(ctx echo.Context) (err error) {
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
