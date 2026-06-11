package comm

import (
	"strings"

	"github.com/GMWalletApp/epusdt/model/request"
	"github.com/GMWalletApp/epusdt/model/response"
	"github.com/GMWalletApp/epusdt/model/service"
	"github.com/GMWalletApp/epusdt/util/constant"
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

// SubmitTxHash 手动提交交易 hash 补单
// @Summary      Submit transaction hash
// @Description  Submit an on-chain transaction hash from the cashier. The hash is verified against the order and, when valid, the same manual payment processing path used by admin mark-paid is executed. OkPay/provider orders are not supported.
// @Description  TON accepts canonical ton:<receive_raw>:<lt>:<hash>, lt:hash, or a unique recent hash-only reference for the order receive address.
// @Tags         Payment
// @Accept       json
// @Produce      json
// @Param        trade_id path string true "Trade ID"
// @Param        request body request.ManualPaymentRequest true "Transaction hash payload"
// @Success      200 {object} response.ApiResponse{data=response.ManualPaymentResponse}
// @Failure      400 {object} response.ApiResponse "Stable errno in status_code: 10008 order not found, 10009 invalid params, 10013 not waiting payment, 10038 verification failed, 10039 unsupported provider, 10007 hash already processed"
// @Router       /pay/submit-tx-hash/{trade_id} [post]
func (c *BaseCommController) SubmitTxHash(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	req := new(request.ManualPaymentRequest)
	if err = ctx.Bind(req); err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	req.BlockTransactionId = strings.TrimSpace(req.BlockTransactionId)
	if err = c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}

	resp, err := service.SubmitCashierManualPayment(tradeId, req.BlockTransactionId)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, resp)
}
