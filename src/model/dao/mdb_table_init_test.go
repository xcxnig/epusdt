package dao

import (
	"testing"

	"github.com/GMWalletApp/epusdt/model/mdb"
)

func TestDefaultRpcNodesIncludesManualVerifyEpusdtEvmNodes(t *testing.T) {
	want := map[string]string{
		mdb.NetworkEthereum: "https://rpc.epusdt.com/ethereum",
		mdb.NetworkBsc:      "https://rpc.epusdt.com/binance",
		mdb.NetworkPolygon:  "https://rpc.epusdt.com/polygon",
	}
	got := make(map[string]mdb.RpcNode)
	for _, node := range defaultRpcNodes() {
		if node.Purpose != mdb.RpcNodePurposeManualVerify {
			continue
		}
		if _, ok := want[node.Network]; ok {
			got[node.Network] = node
		}
	}

	for network, url := range want {
		node, ok := got[network]
		if !ok {
			t.Fatalf("missing manual_verify seed rpc node for %s", network)
		}
		if node.Url != url {
			t.Fatalf("%s manual_verify seed url = %q, want %q", network, node.Url, url)
		}
		if node.Type != mdb.RpcNodeTypeHttp {
			t.Fatalf("%s manual_verify seed type = %q, want %q", network, node.Type, mdb.RpcNodeTypeHttp)
		}
		if !node.Enabled {
			t.Fatalf("%s manual_verify seed enabled = false, want true", network)
		}
		if node.Status != mdb.RpcNodeStatusUnknown {
			t.Fatalf("%s manual_verify seed status = %q, want %q", network, node.Status, mdb.RpcNodeStatusUnknown)
		}
	}
}
