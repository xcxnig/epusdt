package admin

import (
	"strings"

	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/GMWalletApp/epusdt/telegram"
	"github.com/labstack/echo/v4"
)

// SettingUpsertItem is a single setting entry for batch upsert.
// Supported groups and keys:
//
//   - group=rate:
//     rate.forced_usdt_rate  (float)  — override USDT exchange rate (0 = use api)
//     rate.api_url           (string) — external rate API URL
//     rate.adjust_percent    (float)  — rate adjustment percentage
//     rate.okx_c2c_enabled   (bool)   — use OKX C2C rate feed
//
//   - group=epay:
//     epay.default_token     (string) — token for EPAY orders, e.g. "usdt" (default)
//     epay.default_currency  (string) — fiat currency for EPAY orders, e.g. "cny" (default)
//     epay.default_network   (string) — blockchain network for EPAY orders, e.g. "tron" (default)
//
//   - group=brand:
//     brand.site_name        (string) — site display name
//     brand.logo_url         (string) — logo image URL
//     brand.page_title       (string) — payment page title
//     brand.pay_success_text (string) — text shown on payment success
//     brand.support_url      (string) — support / help URL
//
//   - group=system:
//     system.order_expiration_time (int) — order expiry in minutes
type SettingUpsertItem struct {
	Group string `json:"group" enums:"brand,rate,system,epay" example:"epay"`
	Key   string `json:"key" example:"epay.default_network"`
	Value string `json:"value" example:"tron"`
	Type  string `json:"type" enums:"string,int,bool,json" example:"string"`
}

// SettingsUpsertRequest is the payload for batch upserting settings.
type SettingsUpsertRequest struct {
	Items []SettingUpsertItem `json:"items" validate:"required"`
}

// ListSettings returns all rows, optionally filtered by group.
// @Summary      List settings
// @Description  Returns all settings, optionally filtered by group.
// @Description  Available groups: brand, rate, system, epay.
// @Description  See SettingUpsertItem for the full list of supported keys per group.
// @Tags         Admin Settings
// @Security     AdminJWT
// @Produce      json
// @Param        group query string false "Group filter (brand|rate|system|epay)"
// @Success      200 {object} response.ApiResponse{data=[]mdb.Setting}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/settings [get]
func (c *BaseAdminController) ListSettings(ctx echo.Context) error {
	group := strings.ToLower(strings.TrimSpace(ctx.QueryParam("group")))
	rows, err := data.ListSettingsByGroup(group)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, rows)
}

// UpsertSettings batch-inserts / updates rows. Each item is treated
// independently so a malformed row in the middle doesn't drop earlier
// ones. Errors are returned per-key so the UI can surface them.
// @Summary      Upsert settings
// @Description  Batch insert/update settings. Returns per-key status.
// @Description  Supported groups: brand, rate, system, epay.
// @Description  epay group keys: epay.default_token (e.g. "usdt"), epay.default_currency (e.g. "cny"), epay.default_network (e.g. "tron").
// @Description  rate group keys: rate.forced_usdt_rate, rate.api_url, rate.adjust_percent, rate.okx_c2c_enabled.
// @Description  brand group keys: brand.site_name, brand.logo_url, brand.page_title, brand.pay_success_text, brand.support_url.
// @Description  system group keys: system.order_expiration_time.
// @Tags         Admin Settings
// @Security     AdminJWT
// @Accept       json
// @Produce      json
// @Param        request body admin.SettingsUpsertRequest true "Settings payload"
// @Success      200 {object} response.ApiResponse{data=[]admin.SettingsUpsertResult}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/settings [put]
func (c *BaseAdminController) UpsertSettings(ctx echo.Context) error {
	req := new(SettingsUpsertRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, err)
	}
	if err := c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	type result struct {
		Key   string `json:"key"`
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	out := make([]result, 0, len(req.Items))
	for _, item := range req.Items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			out = append(out, result{Key: item.Key, OK: false, Error: "key required"})
			continue
		}
		if err := data.SetSetting(item.Group, key, item.Value, item.Type); err != nil {
			out = append(out, result{Key: key, OK: false, Error: err.Error()})
			continue
		}
		out = append(out, result{Key: key, OK: true})
	}

	// When telegram credentials are updated via settings, reload the
	// command bot so operators don't need to restart the process, and
	// sync the notification_channels row so the notify dispatcher picks
	// up the new values immediately.
	telegramKeys := map[string]bool{
		"system.telegram_bot_token":               true,
		"system.telegram_chat_id":                 true,
		"system.telegram_payment_notice_enabled":  true,
		"system.telegram_abnormal_notice_enabled": true,
	}
	for _, item := range req.Items {
		if telegramKeys[strings.TrimSpace(item.Key)] {
			telegram.ReloadBotAsync("settings upsert")
			go dao.SyncTelegramChannelFromSettings()
			break
		}
	}

	return c.SucJson(ctx, out)
}

// DeleteSetting removes one row. The next read of that key will fall
// back to the hardcoded default (see settings_data.GetSetting*).
// @Summary      Delete setting
// @Description  Remove a setting by key (falls back to default)
// @Tags         Admin Settings
// @Security     AdminJWT
// @Produce      json
// @Param        key path string true "Setting key"
// @Success      200 {object} response.ApiResponse
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/settings/{key} [delete]
func (c *BaseAdminController) DeleteSetting(ctx echo.Context) error {
	key := strings.TrimSpace(ctx.Param("key"))
	if key == "" {
		return c.SucJson(ctx, nil)
	}
	if err := data.DeleteSetting(key); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}

// Public helper for the rate/usdt overrides — used by config package to
// read settings-backed values without importing the controller package.
var _ = mdb.SettingKeyRateForcedUsdt // ensure key constants remain referenced
