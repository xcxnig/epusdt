package comm

import (
	"github.com/assimon/luuu/model/request"
	"github.com/assimon/luuu/model/service"
	"github.com/assimon/luuu/util/constant"
	"github.com/labstack/echo/v4"
)

// LegecyCreateTransaction 创建交易
func (c *BaseCommController) LegecyCreateTransaction(ctx echo.Context) (err error) {
	req := new(request.CreateTransactionRequest)
	if err = ctx.Bind(req); err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	// 兼容旧版本插件：如果未传递 token 和 currency，使用默认值
	if req.Token == "" {
		req.Token = "usdt"
	}
	if req.Currency == "" {
		req.Currency = "cny"
	}
	if err = c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	resp, err := service.CreateTransaction(req)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, resp)
}
