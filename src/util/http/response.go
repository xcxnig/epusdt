package http

import (
	"github.com/GMWalletApp/epusdt/util/constant"
	"github.com/GMWalletApp/epusdt/util/page"
	"github.com/labstack/echo/v4"
	"net/http"
)

type Resp struct{}

type Response struct {
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	RequestID  string      `json:"request_id"`
}

func (r *Resp) View(e echo.Context, code int, html string) error {
	return e.HTML(code, html)
}

// SucView 成功返回
func (r *Resp) SucView(e echo.Context, html string) error {
	return r.View(e, http.StatusOK, html)
}

func (r *Resp) Json(e echo.Context, code int, data interface{}) error {
	return e.JSON(code, data)
}

// SucJson 成功返回json
func (r *Resp) SucJson(e echo.Context, data interface{}, message ...string) error {
	rp := new(Response)
	rp.StatusCode = http.StatusOK
	if len(message) == 0 {
		rp.Message = "success"
	} else {
		for _, m := range message {
			rp.Message += "," + m
		}
	}
	rp.Data = data
	rp.RequestID = e.Request().Header.Get(echo.HeaderXRequestID)
	return r.Json(e, http.StatusOK, rp)
}

// SucJsonPage 分页封装返回json
func (r *Resp) SucJsonPage(e echo.Context, data interface{}, pagination page.Pagination, message ...string) error {
	type PageData struct {
		List       interface{}     `json:"list"`
		Pagination page.Pagination `json:"pagination"`
	}
	pageDate := PageData{
		List:       data,
		Pagination: pagination,
	}
	return r.SucJson(e, pageDate, message...)
}

// FailJson 失败json —  Propagates semantic HTTP status codes to the
// wire. RspError.Code in the 4xx/5xx range is used directly; business
// codes (>=1000) map to HTTP 400 with the specific code still visible
// in the body's status_code field.
func (r *Resp) FailJson(e echo.Context, err error) error {
	rr := new(Response)
	httpStatus := http.StatusBadRequest
	switch t := err.(type) {
	case *constant.RspError:
		rr.StatusCode, rr.Message = t.Render()
		httpStatus = t.HttpStatus()
	default:
		rr.StatusCode = http.StatusBadRequest
		rr.Message = err.Error()
	}
	rr.RequestID = e.Request().Header.Get(echo.HeaderXRequestID)
	return r.Json(e, httpStatus, &rr)
}
