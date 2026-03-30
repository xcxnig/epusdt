package crypto

import (
	"crypto/sha256"

	"github.com/shengdoushi/base58"
)

const AddressLength = 20
const PrefixMainnet = 0x41

// Encode returns the Base58 encoding of input using the Bitcoin alphabet.
func Encode(input []byte) string {
	return base58.Encode(input, base58.BitcoinAlphabet)
}

// EncodeCheck returns the Base58Check encoding of input (Base58 with a 4-byte SHA-256d checksum).
func EncodeCheck(input []byte) string {
	h256h0 := sha256.New()
	h256h0.Write(input)
	h0 := h256h0.Sum(nil)

	h256h1 := sha256.New()
	h256h1.Write(h0)
	h1 := h256h1.Sum(nil)

	inputCheck := append(append([]byte(nil), input...), h1[:4]...)

	return Encode(inputCheck)
}
