package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/GMWalletApp/epusdt/task"
	"github.com/labstack/echo/v4"
)

// CreateRpcNodeRequest is the payload for creating an RPC node.
type CreateRpcNodeRequest struct {
	Network string `json:"network" validate:"required" example:"tron"`
	Url string `json:"url" validate:"required" example:"https://api.trongrid.io"`
	// 连接类型 http=HTTP请求 ws=WebSocket长连接
	Type string `json:"type" validate:"required|in:http,ws" enums:"http,ws" example:"http"`
	Weight  int    `json:"weight" example:"1"`
	ApiKey  string `json:"api_key" example:""`
	Enabled *bool  `json:"enabled" example:"true"`
}

// UpdateRpcNodeRequest is the payload for updating an RPC node.
type UpdateRpcNodeRequest struct {
	Url     *string `json:"url" example:"https://api.trongrid.io"`
	Weight  *int    `json:"weight" example:"1"`
	ApiKey  *string `json:"api_key" example:"your-api-key"`
	Enabled *bool   `json:"enabled" example:"true"`
}

// ListRpcNodes returns rows optionally filtered by network.
// @Summary      List RPC nodes
// @Description  Returns RPC nodes, optionally filtered by network
// @Tags         Admin RPC Nodes
// @Security     AdminJWT
// @Produce      json
// @Param        network query string false "Network filter"
// @Success      200 {object} response.ApiResponse{data=[]mdb.RpcNode}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/rpc-nodes [get]
func (c *BaseAdminController) ListRpcNodes(ctx echo.Context) error {
	network := strings.ToLower(strings.TrimSpace(ctx.QueryParam("network")))
	rows, err := data.ListRpcNodes(network)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, rows)
}

// CreateRpcNode inserts a row. Status starts as "unknown" until the
// health-check task runs.
// @Summary      Create RPC node
// @Description  Create a new RPC node
// @Tags         Admin RPC Nodes
// @Security     AdminJWT
// @Accept       json
// @Produce      json
// @Param        request body admin.CreateRpcNodeRequest true "RPC node payload"
// @Success      200 {object} response.ApiResponse{data=mdb.RpcNode}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/rpc-nodes [post]
func (c *BaseAdminController) CreateRpcNode(ctx echo.Context) error {
	req := new(CreateRpcNodeRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, err)
	}
	if err := c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	weight := req.Weight
	if weight < 1 {
		weight = 1
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &mdb.RpcNode{
		Network:       strings.ToLower(strings.TrimSpace(req.Network)),
		Url:           strings.TrimSpace(req.Url),
		Type:          strings.ToLower(strings.TrimSpace(req.Type)),
		Weight:        weight,
		ApiKey:        req.ApiKey,
		Enabled:       enabled,
		Status:        mdb.RpcNodeStatusUnknown,
		LastLatencyMs: -1,
	}
	if err := data.CreateRpcNode(row); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, row)
}

// UpdateRpcNode patches url/weight/api_key/enabled.
// @Summary      Update RPC node
// @Description  Patch RPC node fields
// @Tags         Admin RPC Nodes
// @Security     AdminJWT
// @Accept       json
// @Produce      json
// @Param        id path int true "RPC node ID"
// @Param        request body admin.UpdateRpcNodeRequest true "Fields to update"
// @Success      200 {object} response.ApiResponse
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/rpc-nodes/{id} [patch]
func (c *BaseAdminController) UpdateRpcNode(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	req := new(UpdateRpcNodeRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, err)
	}
	fields := map[string]interface{}{}
	if req.Url != nil {
		fields["url"] = *req.Url
	}
	if req.Weight != nil {
		fields["weight"] = *req.Weight
	}
	if req.ApiKey != nil {
		fields["api_key"] = *req.ApiKey
	}
	if req.Enabled != nil {
		fields["enabled"] = *req.Enabled
	}
	if err := data.UpdateRpcNodeFields(id, fields); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}

// DeleteRpcNode soft-deletes the row.
// @Summary      Delete RPC node
// @Description  Soft-delete an RPC node
// @Tags         Admin RPC Nodes
// @Security     AdminJWT
// @Produce      json
// @Param        id path int true "RPC node ID"
// @Success      200 {object} response.ApiResponse
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/rpc-nodes/{id} [delete]
func (c *BaseAdminController) DeleteRpcNode(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	if err := data.DeleteRpcNodeByID(id); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}

// HealthCheckRpcNode performs an on-demand probe and writes the result.
// For HTTP endpoints this is a GET to the URL with a short timeout; for
// WS we just attempt a TCP-level check via the same HTTP client (the
// handshake URL resolves the host identically).
// @Summary      Health check RPC node
// @Description  Perform an on-demand health probe on an RPC node
// @Tags         Admin RPC Nodes
// @Security     AdminJWT
// @Produce      json
// @Param        id path int true "RPC node ID"
// @Success      200 {object} response.ApiResponse{data=admin.RpcHealthCheckResponse}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/rpc-nodes/{id}/health-check [post]
func (c *BaseAdminController) HealthCheckRpcNode(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	row, err := data.GetRpcNodeByID(id)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	if row.ID == 0 {
		return c.FailJson(ctx, errors.New("node not found"))
	}
	status, latency := task.ProbeNode(row.Url)
	if err := data.UpdateRpcNodeHealth(id, status, latency); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, map[string]interface{}{
		"status":          status,
		"last_latency_ms": latency,
	})
}
