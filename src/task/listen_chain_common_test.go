package task

import (
	"testing"

	"github.com/GMWalletApp/epusdt/internal/testutil"
	"github.com/GMWalletApp/epusdt/model/dao"
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
