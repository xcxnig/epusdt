package service

import (
	"bytes"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/GMWalletApp/epusdt/model/mdb"
	addressutil "github.com/GMWalletApp/epusdt/util/address"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func tonTestInboundTx(t *testing.T, src, dst *address.Address, amount tlb.Coins, body *cell.Cell) *tlb.Transaction {
	t.Helper()
	return &tlb.Transaction{
		LT:   123,
		Now:  uint32(time.Now().Unix()),
		Hash: bytes.Repeat([]byte{0x11}, 32),
		IO: struct {
			In  *tlb.Message      `tlb:"maybe ^"`
			Out *tlb.MessagesList `tlb:"maybe ^"`
		}{
			In: &tlb.Message{
				MsgType: tlb.MsgTypeInternal,
				Msg: &tlb.InternalMessage{
					SrcAddr: src,
					DstAddr: dst,
					Amount:  amount,
					Body:    body,
				},
			},
		},
	}
}

func TestParseTonInboundTransferNativeTON(t *testing.T) {
	receive := address.MustParseAddr("EQC6KV4zs8TJtSZapOrRFmqSkxzpq-oSCoxekQRKElf4nC1I")
	sender := address.NewAddress(0, 0, bytes.Repeat([]byte{0x22}, 32)).Bounce(false).Testnet(false)
	native := &mdb.ChainToken{BaseModel: mdb.BaseModel{ID: 1}, Network: mdb.NetworkTon, Symbol: "TON", Decimals: 9, Enabled: true}

	tx := tonTestInboundTx(t, sender, receive, tlb.MustFromTON("1.23"), cell.BeginCell().EndCell())
	transfer, err := ParseTonInboundTransfer(tx, receive, native, nil)
	if err != nil {
		t.Fatalf("ParseTonInboundTransfer(): %v", err)
	}
	if transfer == nil {
		t.Fatal("expected native TON transfer")
	}
	if transfer.Token.Symbol != "TON" {
		t.Fatalf("token = %q, want TON", transfer.Token.Symbol)
	}
	if transfer.Amount != 1.23 {
		t.Fatalf("amount = %v, want 1.23", transfer.Amount)
	}
	if transfer.ReceiveAddress != addressutil.NormalizeTonAddressObject(receive) {
		t.Fatalf("receive address = %q", transfer.ReceiveAddress)
	}
	if !strings.HasPrefix(transfer.BlockID, "ton:"+addressutil.TonRawAddressObjectKey(receive)+":123:") {
		t.Fatalf("block id = %q, want canonical TON prefix", transfer.BlockID)
	}
}

func TestParseTonInboundTransferJettonNotification(t *testing.T) {
	receive := address.MustParseAddr("EQC6KV4zs8TJtSZapOrRFmqSkxzpq-oSCoxekQRKElf4nC1I")
	sender := address.NewAddress(0, 0, bytes.Repeat([]byte{0x33}, 32)).Bounce(false).Testnet(false)
	jettonWallet := address.NewAddress(0, 0, bytes.Repeat([]byte{0x44}, 32)).Bounce(false).Testnet(false)
	native := &mdb.ChainToken{BaseModel: mdb.BaseModel{ID: 1}, Network: mdb.NetworkTon, Symbol: "TON", Decimals: 9, Enabled: true}
	usdt := mdb.ChainToken{BaseModel: mdb.BaseModel{ID: 2}, Network: mdb.NetworkTon, Symbol: "USDT", Decimals: 6, Enabled: true}

	body, err := tlb.ToCell(jetton.TransferNotification{
		QueryID:        7,
		Amount:         tlb.MustFromNano(big.NewInt(1230000), 6),
		Sender:         sender,
		ForwardPayload: cell.BeginCell().EndCell(),
	})
	if err != nil {
		t.Fatalf("build jetton notification body: %v", err)
	}

	tx := tonTestInboundTx(t, jettonWallet, receive, tlb.FromNanoTONU(1), body)
	transfer, err := ParseTonInboundTransfer(tx, receive, native, map[string]mdb.ChainToken{
		addressutil.TonRawAddressObjectKey(jettonWallet): usdt,
	})
	if err != nil {
		t.Fatalf("ParseTonInboundTransfer(): %v", err)
	}
	if transfer == nil {
		t.Fatal("expected USDT jetton transfer")
	}
	if transfer.Token.Symbol != "USDT" {
		t.Fatalf("token = %q, want USDT", transfer.Token.Symbol)
	}
	if transfer.Amount != 1.23 {
		t.Fatalf("amount = %v, want 1.23", transfer.Amount)
	}
	if transfer.SenderAddress != addressutil.NormalizeTonAddressObject(sender) {
		t.Fatalf("sender address = %q", transfer.SenderAddress)
	}
	if transfer.JettonWallet != addressutil.NormalizeTonAddressObject(jettonWallet) {
		t.Fatalf("jetton wallet = %q", transfer.JettonWallet)
	}
}

func TestParseTonInboundTransferJettonNotificationWithNoGasComputeSkip(t *testing.T) {
	receive := address.MustParseAddr("EQC6KV4zs8TJtSZapOrRFmqSkxzpq-oSCoxekQRKElf4nC1I")
	sender := address.NewAddress(0, 0, bytes.Repeat([]byte{0x33}, 32)).Bounce(false).Testnet(false)
	jettonWallet := address.NewAddress(0, 0, bytes.Repeat([]byte{0x44}, 32)).Bounce(false).Testnet(false)
	usdt := mdb.ChainToken{BaseModel: mdb.BaseModel{ID: 2}, Network: mdb.NetworkTon, Symbol: "USDT", Decimals: 6, Enabled: true}

	body, err := tlb.ToCell(jetton.TransferNotification{
		QueryID:        7,
		Amount:         tlb.MustFromNano(big.NewInt(150000), 6),
		Sender:         sender,
		ForwardPayload: cell.BeginCell().EndCell(),
	})
	if err != nil {
		t.Fatalf("build jetton notification body: %v", err)
	}

	tx := tonTestInboundTx(t, jettonWallet, receive, tlb.FromNanoTONU(1), body)
	tx.Description = tlb.TransactionDescriptionOrdinary{
		ComputePhase: tlb.ComputePhase{Phase: tlb.ComputePhaseSkipped{
			Reason: tlb.ComputeSkipReason{Type: tlb.ComputeSkipReasonNoGas},
		}},
		Aborted: true,
	}
	transfer, err := ParseTonInboundTransfer(tx, receive, nil, map[string]mdb.ChainToken{
		addressutil.TonRawAddressObjectKey(jettonWallet): usdt,
	})
	if err != nil {
		t.Fatalf("ParseTonInboundTransfer(): %v", err)
	}
	if transfer == nil {
		t.Fatal("expected no-gas jetton notification to be accepted")
	}
	if transfer.Token.Symbol != "USDT" {
		t.Fatalf("token = %q, want USDT", transfer.Token.Symbol)
	}
	if transfer.Amount != 0.15 {
		t.Fatalf("amount = %v, want 0.15", transfer.Amount)
	}
}

func TestParseTonInboundTransferDoesNotCountUntrustedJettonNotificationAsTON(t *testing.T) {
	receive := address.MustParseAddr("EQC6KV4zs8TJtSZapOrRFmqSkxzpq-oSCoxekQRKElf4nC1I")
	sender := address.NewAddress(0, 0, bytes.Repeat([]byte{0x33}, 32)).Bounce(false).Testnet(false)
	untrustedJettonWallet := address.NewAddress(0, 0, bytes.Repeat([]byte{0x55}, 32)).Bounce(false).Testnet(false)
	native := &mdb.ChainToken{BaseModel: mdb.BaseModel{ID: 1}, Network: mdb.NetworkTon, Symbol: "TON", Decimals: 9, Enabled: true}

	body, err := tlb.ToCell(jetton.TransferNotification{
		QueryID:        7,
		Amount:         tlb.MustFromNano(big.NewInt(1230000), 6),
		Sender:         sender,
		ForwardPayload: cell.BeginCell().EndCell(),
	})
	if err != nil {
		t.Fatalf("build jetton notification body: %v", err)
	}

	tx := tonTestInboundTx(t, untrustedJettonWallet, receive, tlb.FromNanoTONU(1), body)
	transfer, err := ParseTonInboundTransfer(tx, receive, native, nil)
	if err != nil {
		t.Fatalf("ParseTonInboundTransfer(): %v", err)
	}
	if transfer != nil {
		t.Fatalf("untrusted jetton notification produced transfer: %#v", transfer)
	}
}
