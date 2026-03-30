package mdb

const (
	TokenStatusEnable  = 1
	TokenStatusDisable = 2
)

// WalletAddress  钱包表
type WalletAddress struct {
	Address string `gorm:"index:wallet_address_index;column:address" json:"address"` //  钱包地址
	Status  int64  `gorm:"column:status;default:1" json:"status"`                    //  1:启用 2:禁用
	BaseModel
}

// TableName sets the insert table name for this struct type
func (w *WalletAddress) TableName() string {
	return "wallet_address"
}
