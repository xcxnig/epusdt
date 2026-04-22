package admin

import (
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/labstack/echo/v4"
)

// UpdateChainRequest is the payload for updating chain settings.
type UpdateChainRequest struct {
	Enabled          *bool   `json:"enabled" example:"true"`
	MinConfirmations *int    `json:"min_confirmations" example:"20"`
	ScanIntervalSec  *int    `json:"scan_interval_sec" example:"5"`
	DisplayName      *string `json:"display_name" example:"Tron"`
}

// ListChains returns the chains table (tron/ethereum/solana seeded; new
// networks like bsc/polygon inserted manually once supported).
// @Summary      List chains
// @Description  Returns all supported blockchain networks
// @Tags         Admin Chains
// @Security     AdminJWT
// @Produce      json
// @Success      200 {object} response.ApiResponse{data=[]mdb.Chain}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/chains [get]
func (c *BaseAdminController) ListChains(ctx echo.Context) error {
	rows, err := data.ListChains()
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, rows)
}

// UpdateChain toggles enabled / tweaks scan params. Scanners re-read the
// chains table on every poll tick, so changes take effect within one
// cycle without a restart.
// @Summary      Update chain
// @Description  Toggle enabled / tweak scan params for a chain
// @Tags         Admin Chains
// @Security     AdminJWT
// @Accept       json
// @Produce      json
// @Param        network path string true "Network name (e.g. tron, ethereum, solana)"
// @Param        request body admin.UpdateChainRequest true "Chain settings"
// @Success      200 {object} response.ApiResponse
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/chains/{network} [patch]
func (c *BaseAdminController) UpdateChain(ctx echo.Context) error {
	network := ctx.Param("network")
	req := new(UpdateChainRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, err)
	}
	fields := map[string]interface{}{}
	if req.Enabled != nil {
		fields["enabled"] = *req.Enabled
	}
	if req.MinConfirmations != nil {
		fields["min_confirmations"] = *req.MinConfirmations
	}
	if req.ScanIntervalSec != nil {
		fields["scan_interval_sec"] = *req.ScanIntervalSec
	}
	if req.DisplayName != nil {
		fields["display_name"] = *req.DisplayName
	}
	if err := data.UpdateChainFields(network, fields); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}
