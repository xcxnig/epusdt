package admin

import (
	"strconv"
	"strings"

	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/labstack/echo/v4"
)

// CreateChainTokenRequest is the payload for creating a chain token.
type CreateChainTokenRequest struct {
	Network         string  `json:"network" validate:"required" example:"tron"`
	Symbol          string  `json:"symbol" validate:"required|maxLen:32" example:"USDT"`
	ContractAddress string  `json:"contract_address" example:"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"`
	Decimals        int     `json:"decimals" example:"6"`
	Enabled         *bool   `json:"enabled" example:"true"`
	MinAmount       float64 `json:"min_amount" example:"1.0"`
}

// UpdateChainTokenRequest is the payload for updating a chain token.
type UpdateChainTokenRequest struct {
	ContractAddress *string  `json:"contract_address" example:"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"`
	Decimals        *int     `json:"decimals" example:"6"`
	Enabled         *bool    `json:"enabled" example:"true"`
	MinAmount       *float64 `json:"min_amount" example:"1.0"`
}

// ChangeChainTokenStatusRequest is the payload for toggling chain token status.
type ChangeChainTokenStatusRequest struct {
	Enabled bool `json:"enabled" example:"true"`
}

// ListChainTokens returns rows, optionally filtered by network.
// @Summary      List chain tokens
// @Description  Returns chain tokens, optionally filtered by network
// @Tags         Admin Chain Tokens
// @Security     AdminJWT
// @Produce      json
// @Param        network query string false "Network filter"
// @Success      200 {object} response.ApiResponse{data=[]mdb.ChainToken}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/chain-tokens [get]
func (c *BaseAdminController) ListChainTokens(ctx echo.Context) error {
	network := strings.ToLower(strings.TrimSpace(ctx.QueryParam("network")))
	rows, err := data.ListChainTokens(network)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, rows)
}

// CreateChainToken inserts a new row. Default decimals=6 when 0 passed.
// @Summary      Create chain token
// @Description  Create a new chain token entry
// @Tags         Admin Chain Tokens
// @Security     AdminJWT
// @Accept       json
// @Produce      json
// @Param        request body admin.CreateChainTokenRequest true "Chain token payload"
// @Success      200 {object} response.ApiResponse{data=mdb.ChainToken}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/chain-tokens [post]
func (c *BaseAdminController) CreateChainToken(ctx echo.Context) error {
	req := new(CreateChainTokenRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, err)
	}
	if err := c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	decimals := req.Decimals
	if decimals == 0 {
		decimals = 6
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &mdb.ChainToken{
		Network:         strings.ToLower(strings.TrimSpace(req.Network)),
		Symbol:          strings.ToUpper(strings.TrimSpace(req.Symbol)),
		ContractAddress: strings.TrimSpace(req.ContractAddress),
		Decimals:        decimals,
		Enabled:         enabled,
		MinAmount:       req.MinAmount,
	}
	if err := data.CreateChainToken(row); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, row)
}

// UpdateChainToken patches mutable columns.
// @Summary      Update chain token
// @Description  Patch chain token fields
// @Tags         Admin Chain Tokens
// @Security     AdminJWT
// @Accept       json
// @Produce      json
// @Param        id path int true "Chain token ID"
// @Param        request body admin.UpdateChainTokenRequest true "Fields to update"
// @Success      200 {object} response.ApiResponse
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/chain-tokens/{id} [patch]
func (c *BaseAdminController) UpdateChainToken(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	req := new(UpdateChainTokenRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, err)
	}
	fields := map[string]interface{}{}
	if req.ContractAddress != nil {
		fields["contract_address"] = *req.ContractAddress
	}
	if req.Decimals != nil {
		fields["decimals"] = *req.Decimals
	}
	if req.Enabled != nil {
		fields["enabled"] = *req.Enabled
	}
	if req.MinAmount != nil {
		fields["min_amount"] = *req.MinAmount
	}
	if err := data.UpdateChainTokenFields(id, fields); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}

// ChangeChainTokenStatus toggles enable/disable.
// @Summary      Change chain token status
// @Description  Toggle enable/disable for a chain token
// @Tags         Admin Chain Tokens
// @Security     AdminJWT
// @Accept       json
// @Produce      json
// @Param        id path int true "Chain token ID"
// @Param        request body admin.ChangeChainTokenStatusRequest true "Status payload"
// @Success      200 {object} response.ApiResponse
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/chain-tokens/{id}/status [post]
func (c *BaseAdminController) ChangeChainTokenStatus(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	req := new(ChangeChainTokenStatusRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, err)
	}
	if err := data.UpdateChainTokenFields(id, map[string]interface{}{"enabled": req.Enabled}); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}

// DeleteChainToken soft-deletes the row.
// @Summary      Delete chain token
// @Description  Soft-delete a chain token
// @Tags         Admin Chain Tokens
// @Security     AdminJWT
// @Produce      json
// @Param        id path int true "Chain token ID"
// @Success      200 {object} response.ApiResponse
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/chain-tokens/{id} [delete]
func (c *BaseAdminController) DeleteChainToken(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	if err := data.DeleteChainTokenByID(id); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}
