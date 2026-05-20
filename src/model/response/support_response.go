package response

type NetworkTokenSupport struct {
	Network     string   `json:"network" example:"tron"`
	DisplayName string   `json:"display_name" example:"TRON"`
	Tokens      []string `json:"tokens" example:"USDT,USDC"`
}

type EpayPublicConfig struct {
	DefaultToken    string `json:"default_token" example:"usdt"`
	DefaultCurrency string `json:"default_currency" example:"cny"`
	DefaultNetwork  string `json:"default_network" example:"tron"`
}

type SitePublicConfig struct {
	CashierName        string `json:"cashier_name" example:"Acme Cashier"`
	LogoURL            string `json:"logo_url" example:"https://cdn.example.com/logo.png"`
	WebsiteTitle       string `json:"website_title" example:"Acme Payments"`
	SupportLink        string `json:"support_link" example:"https://example.com/support"`
	BackgroundColor    string `json:"background_color" example:"#0f172a"`
	BackgroundImageURL string `json:"background_image_url" example:"https://cdn.example.com/background.png"`
}

type OkPayPublicConfig struct {
	Enabled        bool     `json:"enabled" example:"true"`
	AllowTokens    []string `json:"allow_tokens" example:"USDT,TRX"`
	ShopID         string   `json:"shop_id,omitempty" example:"okpay-shop-test"`
	ShopToken      string   `json:"shop_token,omitempty" example:"secret-token"`
	APIURL         string   `json:"api_url,omitempty" example:"https://api.okaypay.me/shop/"`
	CallbackURL    string   `json:"callback_url,omitempty" example:"https://pay.example.com/payments/okpay/v1/notify"`
	ReturnURL      string   `json:"return_url,omitempty" example:"https://pay.example.com/success"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty" example:"10"`
}

type PublicConfigResponse struct {
	SupportedAssets []NetworkTokenSupport `json:"supported_assets"`
	Site            SitePublicConfig      `json:"site"`
	Epay            EpayPublicConfig      `json:"epay"`
	OkPay           OkPayPublicConfig     `json:"okpay"`
}
