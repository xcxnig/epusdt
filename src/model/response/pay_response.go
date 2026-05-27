package response

type CheckoutCounterResponse struct {
	TradeId        string  `json:"trade_id" example:"3nQ9pL2xV7sK1mR8cT4yB_aZ"`   //  epusdt订单号
	Amount         float64 `json:"amount" example:"100.0000"`                     //  订单金额，按 system.amount_precision 保留小数 法币金额
	ActualAmount   float64 `json:"actual_amount" example:"14.2857"`               //  订单实际需要支付的金额，按 system.amount_precision 保留小数 加密货币金额
	Token          string  `json:"token" example:"USDT"`                          //  所属币种 TRX USDT......
	Currency       string  `json:"currency" example:"CNY"`                        //  法币币种 CNY USD ...
	ReceiveAddress string  `json:"receive_address" example:"TTestTronAddress001"` //  收款钱包地址
	Network        string  `json:"network" example:"tron"`                        //  网络 TRON ETH ...
	ExpirationTime int64   `json:"expiration_time" example:"1713264600"`          // 过期时间 时间戳
	RedirectUrl    string  `json:"redirect_url" example:"https://example.com/success"`
	PaymentUrl     string  `json:"payment_url" example:"https://pay.example.com/checkout/3nQ9pL2xV7sK1mR8cT4yB_aZ"` // 支付链接；链上订单通常为本地收银台，OkPay 子订单为第三方 payLink
	CreatedAt      int64   `json:"created_at" example:"1713264000"`                                                 // 订单创建时间 时间戳
	IsSelected     bool    `json:"is_selected" example:"false"`
}

type CheckStatusResponse struct {
	TradeId string `json:"trade_id" example:"3nQ9pL2xV7sK1mR8cT4yB_aZ"` //  epusdt订单号
	// 订单状态 1=等待支付 2=支付成功 3=已过期
	Status int `json:"status" enums:"1,2,3" example:"1"`
}

type ManualPaymentResponse struct {
	TradeId            string `json:"trade_id" example:"3nQ9pL2xV7sK1mR8cT4yB_aZ"`      // epusdt订单号
	Status             int    `json:"status" enums:"1,2,3" example:"2"`                 // 订单状态 1=等待支付 2=支付成功 3=已过期
	BlockTransactionId string `json:"block_transaction_id" example:"0xabc123def456..."` // 已验证并入账的交易哈希
}
