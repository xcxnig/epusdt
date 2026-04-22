package data

import (
	"testing"

	"github.com/GMWalletApp/epusdt/internal/testutil"
	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/mdb"
)

func TestSelectRpcNodeUsesHealthyRow(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkSolana,
		Url:     "https://unknown.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Status:  mdb.RpcNodeStatusUnknown,
	}).Error; err != nil {
		t.Fatalf("seed unknown rpc_node: %v", err)
	}
	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkSolana,
		Url:     "https://ok.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Status:  mdb.RpcNodeStatusOk,
	}).Error; err != nil {
		t.Fatalf("seed ok rpc_node: %v", err)
	}

	got, err := SelectRpcNode(mdb.NetworkSolana, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("SelectRpcNode(): %v", err)
	}
	if got == nil || got.Url != "https://ok.example.com" {
		t.Fatalf("SelectRpcNode() = %#v, want ok row", got)
	}
}

func TestSelectRpcNodeFallsBackToUnknownOnly(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkSolana,
		Url:     "https://unknown.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Status:  mdb.RpcNodeStatusUnknown,
	}).Error; err != nil {
		t.Fatalf("seed unknown rpc_node: %v", err)
	}
	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkSolana,
		Url:     "https://down.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Status:  mdb.RpcNodeStatusDown,
	}).Error; err != nil {
		t.Fatalf("seed down rpc_node: %v", err)
	}

	got, err := SelectRpcNode(mdb.NetworkSolana, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("SelectRpcNode(): %v", err)
	}
	if got == nil || got.Url != "https://unknown.example.com" {
		t.Fatalf("SelectRpcNode() = %#v, want unknown row", got)
	}
}

func TestSelectRpcNodeIgnoresDownRows(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkSolana,
		Url:     "https://down.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Status:  mdb.RpcNodeStatusDown,
	}).Error; err != nil {
		t.Fatalf("seed down rpc_node: %v", err)
	}

	got, err := SelectRpcNode(mdb.NetworkSolana, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("SelectRpcNode(): %v", err)
	}
	if got != nil {
		t.Fatalf("SelectRpcNode() = %#v, want nil", got)
	}
}
