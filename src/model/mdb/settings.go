package mdb

// Setting stores non-credential runtime configuration as key/value pairs.
// Groups: brand, rate, system, epay. Merchant credentials (pid + secret_key)
// live in the api_keys table; notification configs live in NotificationChannel.
// Default rows for the epay group are seeded on first startup (see
// model/dao/mdb_table_init.go seedDefaultSettings). All other groups start
// empty and fall back to hardcoded defaults until an admin sets them.
// Only exception: system.jwt_secret is auto-generated on first startup.
const (
	SettingGroupBrand  = "brand"
	SettingGroupRate   = "rate"
	SettingGroupSystem = "system"
	SettingGroupEpay   = "epay"
)

const (
	SettingTypeString = "string"
	SettingTypeInt    = "int"
	SettingTypeBool   = "bool"
	SettingTypeJSON   = "json"
)

const (
	SettingKeyJwtSecret                = "system.jwt_secret"
	SettingKeyInitAdminPasswordPlain   = "system.init_admin_password_plain"
	SettingKeyInitAdminPasswordHash    = "system.init_admin_password_hash"
	SettingKeyInitAdminPasswordFetched = "system.init_admin_password_fetched"
	SettingKeyInitAdminPasswordChanged = "system.init_admin_password_changed"
	SettingKeyOrderExpiration          = "system.order_expiration_time"
	SettingKeyBrandSiteName            = "brand.site_name"
	SettingKeyBrandLogoUrl             = "brand.logo_url"
	SettingKeyBrandPageTitle           = "brand.page_title"
	SettingKeyBrandPaySuccess          = "brand.pay_success_text"
	SettingKeyBrandSupportUrl          = "brand.support_url"
	SettingKeyRateForcedUsdt           = "rate.forced_usdt_rate"
	SettingKeyRateAdjustPercent        = "rate.adjust_percent"
	SettingKeyRateOkxC2cEnabled        = "rate.okx_c2c_enabled"
	SettingKeyRateApiUrl               = "rate.api_url"

	// EPAY route defaults — can be overridden via admin settings.
	SettingKeyEpayDefaultToken    = "epay.default_token"
	SettingKeyEpayDefaultCurrency = "epay.default_currency"
	SettingKeyEpayDefaultNetwork  = "epay.default_network"
)

type Setting struct {
	Group       string `gorm:"column:group;size:32;index:settings_group_index" json:"group" enums:"brand,rate,system" example:"rate"`
	Key         string `gorm:"column:key;uniqueIndex:settings_key_uindex;size:128" json:"key" example:"rate.forced_usdt_rate"`
	Value       string `gorm:"column:value;type:text" json:"value" example:"7.2"`
	Type        string `gorm:"column:type;size:16;default:string" json:"type" enums:"string,int,bool,json" example:"string"`
	Description string `gorm:"column:description;size:255" json:"description" example:"强制USDT汇率"`
	BaseModel
}

func (s *Setting) TableName() string {
	return "settings"
}
