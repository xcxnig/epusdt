package addressutil

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/crc16"
)

// TonParsedAddress carries the canonical display and raw forms used by epusdt.
type TonParsedAddress struct {
	Address *address.Address
	Display string
	Raw     string
}

// ParseTonMainnetAddress accepts raw and user-friendly TON addresses and rejects
// user-friendly testnet-only addresses. Raw addresses do not contain a testnet
// flag, so they are treated as mainnet addresses by definition.
func ParseTonMainnetAddress(input string) (*TonParsedAddress, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, fmt.Errorf("ton address is empty")
	}

	var (
		addr       *address.Address
		err        error
		checkFlags bool
	)
	if strings.Contains(raw, ":") {
		addr, err = address.ParseRawAddr(raw)
	} else {
		addr, err = parseUserFriendly(raw)
		checkFlags = true
	}
	if err != nil {
		return nil, err
	}
	if addr == nil || addr.Type() != address.StdAddress || addr.IsAddrNone() {
		return nil, fmt.Errorf("unsupported ton address type")
	}
	if checkFlags && addr.IsTestnetOnly() {
		return nil, fmt.Errorf("testnet-only ton address is not supported")
	}

	mainnet := addr.Bounce(false).Testnet(false)
	return &TonParsedAddress{
		Address: mainnet,
		Display: mainnet.String(),
		Raw:     strings.ToLower(mainnet.StringRaw()),
	}, nil
}

// NormalizeTonAddress returns the canonical user-friendly address stored in DB.
func NormalizeTonAddress(input string) (string, error) {
	parsed, err := ParseTonMainnetAddress(input)
	if err != nil {
		return "", err
	}
	return parsed.Display, nil
}

// TonRawAddressKey returns the workchain:hash key used for comparisons and locks.
func TonRawAddressKey(input string) (string, error) {
	parsed, err := ParseTonMainnetAddress(input)
	if err != nil {
		return "", err
	}
	return parsed.Raw, nil
}

// NormalizeTonAddressObject returns the canonical user-friendly form for a parsed
// address object.
func NormalizeTonAddressObject(addr *address.Address) string {
	if addr == nil {
		return ""
	}
	return addr.Bounce(false).Testnet(false).String()
}

// TonRawAddressObjectKey returns the raw workchain:hash key for a parsed address.
func TonRawAddressObjectKey(addr *address.Address) string {
	if addr == nil {
		return ""
	}
	return strings.ToLower(addr.Bounce(false).Testnet(false).StringRaw())
}

// TonAddressFromShortAccount builds a standard TON address from a shard
// transaction account id returned by liteserver block transaction lists.
func TonAddressFromShortAccount(workchain int32, account []byte) *address.Address {
	if len(account) != 32 {
		return nil
	}
	return address.NewAddress(0, byte(workchain), append([]byte(nil), account...)).Bounce(false).Testnet(false)
}

func parseUserFriendly(input string) (*address.Address, error) {
	if addr, err := address.ParseAddr(input); err == nil {
		return addr, nil
	}
	for _, enc := range []*base64.Encoding{
		base64.RawStdEncoding,
		base64.StdEncoding,
		base64.URLEncoding,
	} {
		data, err := enc.DecodeString(input)
		if err != nil {
			continue
		}
		addr, err := addressFromFriendlyBytes(data)
		if err == nil {
			return addr, nil
		}
	}
	return nil, fmt.Errorf("invalid ton address")
}

func addressFromFriendlyBytes(data []byte) (*address.Address, error) {
	if len(data) != 36 {
		return nil, fmt.Errorf("incorrect ton address data length")
	}
	want := binary.BigEndian.Uint16(data[34:])
	if crc16.ChecksumXMODEM(data[:34]) != want {
		return nil, fmt.Errorf("invalid ton address checksum")
	}
	if len(data[2:34]) != 32 {
		return nil, fmt.Errorf("incorrect ton address hash length")
	}
	return address.NewAddress(data[0], data[1], append([]byte(nil), data[2:34]...)), nil
}
