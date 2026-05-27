package task

import (
	"testing"

	"github.com/GMWalletApp/epusdt/internal/testutil"
	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
)

func TestResolveChainWsURLRequiresEnabledRpcNode(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if got, ok := resolveChainWsURL(mdb.NetworkEthereum, "[TEST]"); ok {
		t.Fatalf("resolveChainWsURL() = (%q, true), want false", got)
	}
}

func TestResolveChainWsURLWithRow(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	node := &mdb.RpcNode{
		Network: mdb.NetworkEthereum,
		Url:     " wss://ethereum.example.com ",
		Type:    mdb.RpcNodeTypeWs,
		Weight:  1,
		Enabled: true,
		Status:  mdb.RpcNodeStatusOk,
	}
	if err := dao.Mdb.Create(node).Error; err != nil {
		t.Fatalf("seed rpc_node: %v", err)
	}

	got, ok := resolveChainWsURL(mdb.NetworkEthereum, "[TEST]")
	if !ok {
		t.Fatalf("resolveChainWsURL() ok=false, want true")
	}
	if got != "wss://ethereum.example.com" {
		t.Fatalf("resolveChainWsURL() = %q, want wss://ethereum.example.com", got)
	}
}

func TestResolveChainWsURLIgnoresManualVerifyOnly(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkEthereum,
		Url:     "wss://paid-ethereum.example.com",
		Type:    mdb.RpcNodeTypeWs,
		Weight:  100,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeManualVerify,
		Status:  mdb.RpcNodeStatusOk,
	}).Error; err != nil {
		t.Fatalf("seed manual rpc_node: %v", err)
	}

	if got, ok := resolveChainWsURL(mdb.NetworkEthereum, "[TEST]"); ok {
		t.Fatalf("resolveChainWsURL() = (%q, true), want false", got)
	}
}

func TestResolveChainWsURLUsesGeneralWhenManualVerifyExists(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	rows := []mdb.RpcNode{
		{Network: mdb.NetworkEthereum, Url: "wss://paid-ethereum.example.com", Type: mdb.RpcNodeTypeWs, Weight: 100, Enabled: true, Purpose: mdb.RpcNodePurposeManualVerify, Status: mdb.RpcNodeStatusOk},
		{Network: mdb.NetworkEthereum, Url: " wss://general-ethereum.example.com ", Type: mdb.RpcNodeTypeWs, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
	}
	if err := dao.Mdb.Create(&rows).Error; err != nil {
		t.Fatalf("seed rpc_nodes: %v", err)
	}

	got, ok := resolveChainWsURL(mdb.NetworkEthereum, "[TEST]")
	if !ok {
		t.Fatalf("resolveChainWsURL() ok=false, want true")
	}
	if got != "wss://general-ethereum.example.com" {
		t.Fatalf("resolveChainWsURL() = %q, want general node", got)
	}
}

func TestResolveChainWsNodeSkipsCoolingNode(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()
	data.ResetRpcFailoverForTest()
	t.Cleanup(data.ResetRpcFailoverForTest)

	rows := []mdb.RpcNode{
		{Network: mdb.NetworkEthereum, Url: "wss://primary-ethereum.example.com", Type: mdb.RpcNodeTypeWs, Weight: 100, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
		{Network: mdb.NetworkEthereum, Url: "wss://backup-ethereum.example.com", Type: mdb.RpcNodeTypeWs, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
	}
	if err := dao.Mdb.Create(&rows).Error; err != nil {
		t.Fatalf("seed rpc_nodes: %v", err)
	}

	primary, ok := resolveChainWsNode(mdb.NetworkEthereum, "[TEST]")
	if !ok || primary.Url != "wss://primary-ethereum.example.com" {
		t.Fatalf("primary node = %#v ok=%v, want primary", primary, ok)
	}
	for i := 0; i < data.RpcFailoverThreshold; i++ {
		data.RecordRpcNodeFailure(primary.ID)
	}

	got, ok := resolveChainWsNode(mdb.NetworkEthereum, "[TEST]")
	if !ok {
		t.Fatalf("resolveChainWsNode() ok=false, want true")
	}
	if got.Url != "wss://backup-ethereum.example.com" {
		t.Fatalf("resolveChainWsNode() = %#v, want backup", got)
	}
}

func TestResolveChainWsURLDisabledRow(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	node := &mdb.RpcNode{
		Network: mdb.NetworkEthereum,
		Url:     "wss://disabled.example.com",
		Type:    mdb.RpcNodeTypeWs,
		Weight:  1,
		Enabled: true,
		Status:  mdb.RpcNodeStatusOk,
	}
	if err := dao.Mdb.Create(node).Error; err != nil {
		t.Fatalf("seed rpc_node: %v", err)
	}
	if err := dao.Mdb.Model(node).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable rpc_node: %v", err)
	}

	if got, ok := resolveChainWsURL(mdb.NetworkEthereum, "[TEST]"); ok {
		t.Fatalf("resolveChainWsURL() = (%q, true), want false", got)
	}
}
