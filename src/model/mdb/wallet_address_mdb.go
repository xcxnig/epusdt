package mdb

const (
	TokenStatusEnable  = 1
	TokenStatusDisable = 2
)

const (
	NetworkTron     = "tron"
	NetworkSolana   = "solana"
	NetworkEthereum = "eth"
)

type WalletAddress struct {
	Network string `gorm:"column:network;uniqueIndex:wallet_address_network_address_uindex" json:"network"`
	Address string `gorm:"column:address;uniqueIndex:wallet_address_network_address_uindex" json:"address"`
	Status  int64  `gorm:"column:status;default:1" json:"status"`
	BaseModel
}

func (w *WalletAddress) TableName() string {
	return "wallet_address"
}
