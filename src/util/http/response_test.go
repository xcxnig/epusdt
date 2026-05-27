package http

import (
	"encoding/json"
	"errors"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/GMWalletApp/epusdt/util/constant"
	"github.com/labstack/echo/v4"
)

func TestFailJsonHidesPlainErrorsBehindSystemErrno(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := new(Resp).FailJson(ctx, errors.New("database connection details")); err != nil {
		t.Fatalf("FailJson returned error: %v", err)
	}
	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("http status = %d, want 400", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := int(body["status_code"].(float64)); got != 400 {
		t.Fatalf("status_code = %d, want 400; body=%v", got, body)
	}
	if got, _ := body["message"].(string); got != constant.Errno[400] {
		t.Fatalf("message = %q, want %q", got, constant.Errno[400])
	}
}

func TestFailJsonPreservesWrappedRspErrorErrno(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	err := fmt.Errorf("wrap: %w", constant.OrderNotExists)
	if callErr := new(Resp).FailJson(ctx, err); callErr != nil {
		t.Fatalf("FailJson returned error: %v", callErr)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := int(body["status_code"].(float64)); got != 10008 {
		t.Fatalf("status_code = %d, want 10008; body=%v", got, body)
	}
	if got, _ := body["message"].(string); got != constant.Errno[10008] {
		t.Fatalf("message = %q, want %q", got, constant.Errno[10008])
	}
}
