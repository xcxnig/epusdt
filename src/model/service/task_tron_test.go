package service

import (
	"testing"

	"github.com/GMWalletApp/epusdt/internal/testutil"
	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/mdb"
)

// TestResolveTronNode_NoRow verifies that resolveTronNode requires an enabled
// TRON row from rpc_nodes.
func TestResolveTronNode_NoRow(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if gotURL, gotKey, err := resolveTronNode(); err == nil {
		t.Fatalf("resolveTronNode() = (%q, %q, nil), want error", gotURL, gotKey)
	}
}

// TestResolveTronNode_WithRow verifies that resolveTronNode uses the DB row
// when an enabled TRON HTTP node is present.
func TestResolveTronNode_WithRow(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	// Insert an enabled TRON http node.
	node := &mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "https://custom-tron.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		ApiKey:  "db-api-key",
		Weight:  1,
		Enabled: true,
		Status:  mdb.RpcNodeStatusOk,
	}
	if err := dao.Mdb.Create(node).Error; err != nil {
		t.Fatalf("seed rpc_node: %v", err)
	}

	gotURL, gotKey, err := resolveTronNode()
	if err != nil {
		t.Fatalf("resolveTronNode(): %v", err)
	}
	if gotURL != "https://custom-tron.example.com" {
		t.Errorf("url = %q, want https://custom-tron.example.com", gotURL)
	}
	if gotKey != "db-api-key" {
		t.Errorf("apiKey = %q, want \"db-api-key\"", gotKey)
	}
}

// TestResolveTronNode_DisabledRow verifies that a disabled row is ignored.
func TestResolveTronNode_DisabledRow(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	node := &mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "https://disabled-tron.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		ApiKey:  "disabled-key",
		Weight:  1,
		Enabled: true, // start enabled, then disable below
		Status:  mdb.RpcNodeStatusOk,
	}
	if err := dao.Mdb.Create(node).Error; err != nil {
		t.Fatalf("seed rpc_node: %v", err)
	}
	// Explicitly disable — GORM does not save bool zero-value on Create.
	if err := dao.Mdb.Model(node).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable rpc_node: %v", err)
	}

	if gotURL, gotKey, err := resolveTronNode(); err == nil {
		t.Fatalf("resolveTronNode() = (%q, %q, nil), want error", gotURL, gotKey)
	}
}
