package telegram

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/btcsuite/btcutil/base58"
	"github.com/gagliardetto/solana-go"
)

// isValidEthereumAddress 校验 0x + 20 字节十六进制（主网收款）。
func isValidEthereumAddress(addr string) bool {
	addr = strings.TrimSpace(addr)
	if len(addr) != 42 || !strings.HasPrefix(addr, "0x") {
		return false
	}
	_, err := hex.DecodeString(addr[2:])
	return err == nil
}

// isValidTronAddress 校验 Tron Base58Check 地址是否合法
func isValidTronAddress(addr string) bool {
	// 基本过滤
	if len(addr) < 26 || len(addr) > 35 || addr[0] != 'T' {
		return false
	}

	decoded := base58.Decode(addr)
	if len(decoded) != 25 {
		return false
	}

	// TRON 主网地址必须以 0x41 开头
	if decoded[0] != 0x41 {
		return false
	}

	// Base58Check 校验
	payload := decoded[:21]  // 前 21 字节
	checksum := decoded[21:] // 后 4 字节

	hash := sha256.Sum256(payload)
	hash2 := sha256.Sum256(hash[:])

	return string(checksum) == string(hash2[:4])
}

func isValidAddressByNetwork(network, addr string) bool {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case mdb.NetworkTron:
		return isValidTronAddress(addr)
	case mdb.NetworkSolana:
		return isValidSolanaAddress(addr)
	default:
		// 其余 EVM 链统一使用 0x 地址校验
		return isValidEthereumAddress(addr)
	}
}

func normalizeWalletAddressByNetwork(network, addr string) string {
	addr = strings.TrimSpace(addr)
	switch strings.ToLower(strings.TrimSpace(network)) {
	case mdb.NetworkTron, mdb.NetworkSolana:
		return addr
	default:
		return strings.ToLower(addr)
	}
}

// isValidSolanaAddress 校验 Solana Base58 地址是否合法（32 字节公钥）。
func isValidSolanaAddress(addr string) bool {
	addr = strings.TrimSpace(addr)
	if len(addr) < 32 || len(addr) > 44 {
		return false
	}
	_, err := solana.PublicKeyFromBase58(addr)
	return err == nil
}
