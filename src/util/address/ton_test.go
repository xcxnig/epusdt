package addressutil

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/xssnick/tonutils-go/address"
)

const testTonBounceable = "EQC6KV4zs8TJtSZapOrRFmqSkxzpq-oSCoxekQRKElf4nC1I"

func TestParseTonMainnetAddressNormalizesSupportedFormats(t *testing.T) {
	base := address.MustParseAddr(testTonBounceable)
	raw := base.StringRaw()
	nonBounce := base.Bounce(false).String()
	rawBytes, err := base64.RawURLEncoding.DecodeString(testTonBounceable)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	stdBase64 := base64.RawStdEncoding.EncodeToString(rawBytes)

	for _, input := range []string{testTonBounceable, nonBounce, raw, stdBase64} {
		parsed, err := ParseTonMainnetAddress(input)
		if err != nil {
			t.Fatalf("ParseTonMainnetAddress(%q): %v", input, err)
		}
		if parsed.Display != nonBounce {
			t.Fatalf("display for %q = %q, want %q", input, parsed.Display, nonBounce)
		}
		if parsed.Raw != strings.ToLower(raw) {
			t.Fatalf("raw for %q = %q, want %q", input, parsed.Raw, strings.ToLower(raw))
		}
	}
}

func TestParseTonMainnetAddressRejectsTestnetOnly(t *testing.T) {
	testnet := address.MustParseAddr(testTonBounceable).Testnet(true).String()
	if _, err := ParseTonMainnetAddress(testnet); err == nil {
		t.Fatal("expected testnet-only address to be rejected")
	}
}

func TestTonAddressFromShortAccount(t *testing.T) {
	base := address.MustParseAddr(testTonBounceable)
	got := TonAddressFromShortAccount(base.Workchain(), base.Data())
	if got == nil {
		t.Fatal("TonAddressFromShortAccount() returned nil")
	}
	if TonRawAddressObjectKey(got) != TonRawAddressObjectKey(base) {
		t.Fatalf("raw address = %q, want %q", TonRawAddressObjectKey(got), TonRawAddressObjectKey(base))
	}
	if got = TonAddressFromShortAccount(base.Workchain(), base.Data()[:31]); got != nil {
		t.Fatalf("TonAddressFromShortAccount() with short account = %#v, want nil", got)
	}
}
