package comm

import (
	"sort"
	"strconv"
	"strings"

	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/response"
	"github.com/GMWalletApp/epusdt/util/constant"
	"github.com/labstack/echo/v4"
)

// GetSupportedAssets 对外公开可用链与 token 列表。数据源是管理后台
// 的 chains + chain_tokens + wallet_address 三张表的实时查询，admin
// 启用/禁用链或代币后，这个接口在下次调用时就会反映新状态（无缓存）。
// 仅返回：链 enabled=true、代币 enabled=true、且该链上至少有一个可用
// 钱包地址的组合。
// @Summary      List supported (network, tokens) groupings
// @Description  Public endpoint, Auth required in /admin/api/v1/supported-assets
// @Tags         Supported Assets
// @Produce      json
// @Success      200 {object} response.ApiResponse{data=response.SupportedAssetsResponse}
// @Failure      400 {object} response.ApiResponse
// @Router       /payments/gmpay/v1/supported-assets [get]
// @Router       /admin/api/v1/supported-assets [get]
func (c *BaseCommController) GetSupportedAssets(ctx echo.Context) error {
	chains, err := data.ListEnabledChains()
	if err != nil {
		return c.FailJson(ctx, err)
	}
	wallets, err := data.GetAvailableWalletAddress()
	if err != nil {
		return c.FailJson(ctx, err)
	}

	// A chain only qualifies if it has at least one wallet available —
	// without a recipient address, payments on that chain can't be
	// processed even if the chain + token config says "enabled".
	networkHasWallet := make(map[string]struct{})
	for _, w := range wallets {
		networkHasWallet[strings.ToLower(w.Network)] = struct{}{}
	}

	supports := make([]response.NetworkTokenSupport, 0, len(chains))
	for _, ch := range chains {
		network := strings.ToLower(ch.Network)
		if _, ok := networkHasWallet[network]; !ok {
			continue
		}
		tokens, err := data.ListEnabledChainTokensByNetwork(network)
		if err != nil {
			return c.FailJson(ctx, err)
		}
		if len(tokens) == 0 {
			continue
		}
		symbols := make([]string, 0, len(tokens))
		for _, t := range tokens {
			sym := strings.ToUpper(strings.TrimSpace(t.Symbol))
			if sym == "" {
				continue
			}
			symbols = append(symbols, sym)
		}
		if len(symbols) == 0 {
			continue
		}
		sort.Strings(symbols)
		supports = append(supports, response.NetworkTokenSupport{
			Network: network,
			Tokens:  symbols,
		})
	}

	return c.SucJson(ctx, response.SupportedAssetsResponse{Supports: supports})
}

// ListSupportedAssetRecords 查询支持项明细（无需鉴权）。
// @Summary      List supported asset records
// @Description  Public endpoint. Returns enabled chain_tokens, optionally filtered by network.
// @Tags         Supported Assets
// @Produce      json
// @Param        network query string false "Network filter (e.g. tron, ethereum)"
// @Success      200 {object} response.ApiResponse{data=[]mdb.ChainToken}
// @Failure      400 {object} response.ApiResponse
// @Router       /payments/gmpay/v1/supported-assets/records [get]
func (c *BaseCommController) ListSupportedAssetRecords(ctx echo.Context) error {
	network := ctx.QueryParam("network")
	list, err := data.ListEnabledChainTokens(network)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, list)
}

// GetSupportedAsset 查询单条支持项（无需鉴权）。
// @Summary      Get a supported asset
// @Description  Public endpoint. Returns one chain_token row by ID. Returns 404 if the token is disabled.
// @Tags         Supported Assets
// @Produce      json
// @Param        id path int true "ChainToken ID"
// @Success      200 {object} response.ApiResponse{data=mdb.ChainToken}
// @Failure      400 {object} response.ApiResponse
// @Router       /payments/gmpay/v1/supported-assets/{id} [get]
func (c *BaseCommController) GetSupportedAsset(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	token, err := data.GetChainTokenByID(id)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	if token.ID <= 0 || !token.Enabled {
		return c.FailJson(ctx, constant.SupportedAssetNotFound)
	}
	return c.SucJson(ctx, token)
}
