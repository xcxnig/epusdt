package task

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GMWalletApp/epusdt/internal/testutil"
	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
)

func TestTronScannerSwitchesNodeAfterFailureThreshold(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()
	data.ResetRpcFailoverForTest()
	t.Cleanup(data.ResetRpcFailoverForTest)

	primary := &mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "https://primary-tron.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  100,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeGeneral,
		Status:  mdb.RpcNodeStatusOk,
	}
	backup := &mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "https://backup-tron.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeGeneral,
		Status:  mdb.RpcNodeStatusOk,
	}
	if err := dao.Mdb.Create(primary).Error; err != nil {
		t.Fatalf("seed primary rpc_node: %v", err)
	}
	if err := dao.Mdb.Create(backup).Error; err != nil {
		t.Fatalf("seed backup rpc_node: %v", err)
	}

	scanner := NewScanner()
	scanner.useRpcNode(primary)
	for i := 0; i < data.RpcFailoverThreshold-1; i++ {
		scanner.recordRpcFailure("test")
		if scanner.nodeID != primary.ID {
			t.Fatalf("node switched before threshold to id=%d", scanner.nodeID)
		}
	}

	scanner.recordRpcFailure("test")
	if scanner.nodeID != backup.ID {
		t.Fatalf("scanner node id = %d, want backup id=%d", scanner.nodeID, backup.ID)
	}
	if scanner.baseURL != backup.Url {
		t.Fatalf("scanner baseURL = %q, want %q", scanner.baseURL, backup.Url)
	}
}

func TestTronScannerStopsOnHistoricalBlockFetchError(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()
	data.ResetRpcFailoverForTest()
	t.Cleanup(data.ResetRpcFailoverForTest)

	node := &mdb.RpcNode{
		Network: mdb.NetworkTron,
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeGeneral,
		Status:  mdb.RpcNodeStatusOk,
	}
	if err := dao.Mdb.Create(node).Error; err != nil {
		t.Fatalf("seed rpc_node: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/wallet/getnowblock":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"block_header": map[string]interface{}{
					"raw_data": map[string]interface{}{
						"number":    3,
						"timestamp": int64(1000),
					},
				},
				"transactions": []interface{}{},
			})
		case "/wallet/getblockbynum":
			http.Error(w, "temporary", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	node.Url = server.URL
	scanner := NewScanner()
	scanner.useRpcNode(node)
	scanner.lastBlock = 1

	scanner.poll()
	if scanner.lastBlock != 1 {
		t.Fatalf("lastBlock = %d, want 1 so failed block is retried", scanner.lastBlock)
	}
}
