package data

import (
	"strings"
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

func TestSelectRpcNodeExcludesManualVerifyRows(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "https://manual.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  100,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeManualVerify,
		Status:  mdb.RpcNodeStatusOk,
	}).Error; err != nil {
		t.Fatalf("seed manual rpc_node: %v", err)
	}
	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "https://general.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeGeneral,
		Status:  mdb.RpcNodeStatusUnknown,
	}).Error; err != nil {
		t.Fatalf("seed general rpc_node: %v", err)
	}

	got, err := SelectRpcNode(mdb.NetworkTron, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("SelectRpcNode(): %v", err)
	}
	if got == nil || got.Url != "https://general.example.com" {
		t.Fatalf("SelectRpcNode() = %#v, want general row", got)
	}
}

func TestSelectRpcNodePurposeFilterAppliesToAnyNodeType(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	const nodeType = "grpc"
	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "grpc://manual.example.com",
		Type:    nodeType,
		Weight:  100,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeManualVerify,
		Status:  mdb.RpcNodeStatusOk,
	}).Error; err != nil {
		t.Fatalf("seed manual rpc_node: %v", err)
	}
	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "grpc://general.example.com",
		Type:    nodeType,
		Weight:  1,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeGeneral,
		Status:  mdb.RpcNodeStatusOk,
	}).Error; err != nil {
		t.Fatalf("seed general rpc_node: %v", err)
	}

	got, err := SelectRpcNode(mdb.NetworkTron, nodeType)
	if err != nil {
		t.Fatalf("SelectRpcNode(): %v", err)
	}
	if got == nil || got.Url != "grpc://general.example.com" {
		t.Fatalf("SelectRpcNode() = %#v, want general grpc row", got)
	}
}

func TestSelectRpcNodeManualVerifyOnlyReturnsNil(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if err := dao.Mdb.Create(&mdb.RpcNode{
		Network: mdb.NetworkTron,
		Url:     "https://manual-only.example.com",
		Type:    mdb.RpcNodeTypeHttp,
		Weight:  1,
		Enabled: true,
		Purpose: mdb.RpcNodePurposeManualVerify,
		Status:  mdb.RpcNodeStatusOk,
	}).Error; err != nil {
		t.Fatalf("seed manual rpc_node: %v", err)
	}

	got, err := SelectRpcNode(mdb.NetworkTron, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("SelectRpcNode(): %v", err)
	}
	if got != nil {
		t.Fatalf("SelectRpcNode() = %#v, want nil", got)
	}
}

func TestListManualPaymentRpcCandidatesGeneralBeforeManualVerify(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	rows := []mdb.RpcNode{
		{Network: mdb.NetworkTron, Url: "https://manual-ok.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeManualVerify, Status: mdb.RpcNodeStatusOk},
		{Network: mdb.NetworkTron, Url: "https://general-unknown.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusUnknown},
		{Network: mdb.NetworkTron, Url: "https://general-ok.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
		{Network: mdb.NetworkTron, Url: "https://manual-unknown.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeManualVerify, Status: mdb.RpcNodeStatusUnknown},
		{Network: mdb.NetworkTron, Url: "https://down.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusDown},
	}
	if err := dao.Mdb.Create(&rows).Error; err != nil {
		t.Fatalf("seed rpc_nodes: %v", err)
	}

	got, err := ListManualPaymentRpcCandidates(mdb.NetworkTron, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("ListManualPaymentRpcCandidates(): %v", err)
	}
	var urls []string
	for _, row := range got {
		urls = append(urls, row.Url)
	}
	want := []string{
		"https://general-ok.example.com",
		"https://general-unknown.example.com",
		"https://manual-ok.example.com",
		"https://manual-unknown.example.com",
	}
	if len(urls) != len(want) {
		t.Fatalf("candidate urls = %#v, want %#v", urls, want)
	}
	for i := range want {
		if urls[i] != want[i] {
			t.Fatalf("candidate urls = %#v, want %#v", urls, want)
		}
	}
}

func TestListManualPaymentRpcCandidatesAllowsManualVerifyFallbackOnly(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	rows := []mdb.RpcNode{
		{Network: mdb.NetworkTron, Url: "https://manual-ok.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeManualVerify, Status: mdb.RpcNodeStatusOk},
		{Network: mdb.NetworkTron, Url: "https://manual-down.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeManualVerify, Status: mdb.RpcNodeStatusDown},
	}
	if err := dao.Mdb.Create(&rows).Error; err != nil {
		t.Fatalf("seed rpc_nodes: %v", err)
	}

	got, err := ListManualPaymentRpcCandidates(mdb.NetworkTron, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("ListManualPaymentRpcCandidates(): %v", err)
	}
	if len(got) != 1 || got[0].Url != "https://manual-ok.example.com" {
		t.Fatalf("candidate rows = %#v, want only manual ok fallback", got)
	}
}

func TestSelectGeneralRpcNodeSkipsCoolingNode(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()
	ResetRpcFailoverForTest()
	t.Cleanup(ResetRpcFailoverForTest)

	rows := []mdb.RpcNode{
		{Network: mdb.NetworkTron, Url: "https://primary.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 100, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
		{Network: mdb.NetworkTron, Url: "https://backup.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
	}
	if err := dao.Mdb.Create(&rows).Error; err != nil {
		t.Fatalf("seed rpc_nodes: %v", err)
	}

	first, err := SelectGeneralRpcNode(mdb.NetworkTron, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("SelectGeneralRpcNode(): %v", err)
	}
	if first == nil || first.Url != "https://primary.example.com" {
		t.Fatalf("first node = %#v, want primary", first)
	}

	for i := 0; i < RpcFailoverThreshold; i++ {
		RecordRpcNodeFailure(first.ID)
	}

	got, err := SelectGeneralRpcNode(mdb.NetworkTron, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("SelectGeneralRpcNode(): %v", err)
	}
	if got == nil || got.Url != "https://backup.example.com" {
		t.Fatalf("node after cooldown = %#v, want backup", got)
	}
}

func TestSelectGeneralRpcNodeFallsBackWhenAllNodesCooling(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()
	ResetRpcFailoverForTest()
	t.Cleanup(ResetRpcFailoverForTest)

	rows := []mdb.RpcNode{
		{Network: mdb.NetworkTron, Url: "https://primary.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 100, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
		{Network: mdb.NetworkTron, Url: "https://backup.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
	}
	if err := dao.Mdb.Create(&rows).Error; err != nil {
		t.Fatalf("seed rpc_nodes: %v", err)
	}
	for i := range rows {
		for n := 0; n < RpcFailoverThreshold; n++ {
			RecordRpcNodeFailure(rows[i].ID)
		}
	}

	got, err := SelectGeneralRpcNode(mdb.NetworkTron, mdb.RpcNodeTypeHttp)
	if err != nil {
		t.Fatalf("SelectGeneralRpcNode(): %v", err)
	}
	if got == nil || got.Url != "https://primary.example.com" {
		t.Fatalf("SelectGeneralRpcNode() = %#v, want primary when every node is cooling", got)
	}
}

func TestSelectGeneralRpcNodeReturnsNilWhenExcludedAlternatesCooling(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()
	ResetRpcFailoverForTest()
	t.Cleanup(ResetRpcFailoverForTest)

	rows := []mdb.RpcNode{
		{Network: mdb.NetworkTron, Url: "https://primary.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 100, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
		{Network: mdb.NetworkTron, Url: "https://backup.example.com", Type: mdb.RpcNodeTypeHttp, Weight: 1, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusOk},
	}
	if err := dao.Mdb.Create(&rows).Error; err != nil {
		t.Fatalf("seed rpc_nodes: %v", err)
	}
	for i := 0; i < RpcFailoverThreshold; i++ {
		RecordRpcNodeFailure(rows[1].ID)
	}

	got, err := SelectGeneralRpcNode(mdb.NetworkTron, mdb.RpcNodeTypeHttp, rows[0].ID)
	if err != nil {
		t.Fatalf("SelectGeneralRpcNode(): %v", err)
	}
	if got != nil {
		t.Fatalf("SelectGeneralRpcNode() = %#v, want nil when only alternate is cooling", got)
	}
}

func TestRecordRpcNodeSuccessClearsCooldown(t *testing.T) {
	ResetRpcFailoverForTest()
	t.Cleanup(ResetRpcFailoverForTest)

	const nodeID uint64 = 42
	for i := 0; i < RpcFailoverThreshold; i++ {
		RecordRpcNodeFailure(nodeID)
	}
	if !IsRpcNodeCoolingDown(nodeID) {
		t.Fatal("node should be cooling after threshold")
	}
	RecordRpcNodeSuccess(nodeID)
	if IsRpcNodeCoolingDown(nodeID) {
		t.Fatal("node should not be cooling after success")
	}
	failures, cooling := RecordRpcNodeFailure(nodeID)
	if cooling || failures != 1 {
		t.Fatalf("failure state after success = failures=%d cooling=%v, want failures=1 cooling=false", failures, cooling)
	}
}

func TestRpcNodeLogLabelHidesPathAndQuery(t *testing.T) {
	label := RpcNodeLogLabel(mdb.RpcNode{
		BaseModel: mdb.BaseModel{ID: 7},
		Network:   mdb.NetworkEthereum,
		Type:      mdb.RpcNodeTypeWs,
		Url:       "wss://rpc.example.com/v3/secret-key?token=secret",
	})
	if strings.Contains(label, "secret") || strings.Contains(label, "/v3/") {
		t.Fatalf("RpcNodeLogLabel() leaked sensitive URL parts: %s", label)
	}
	if !strings.Contains(label, "wss://rpc.example.com") {
		t.Fatalf("RpcNodeLogLabel() = %s, want host", label)
	}
}

func TestListRpcNodesForHealthExcludesManualVerify(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	rows := []mdb.RpcNode{
		{Network: mdb.NetworkTron, Url: "https://general.example.com", Type: mdb.RpcNodeTypeHttp, Enabled: true, Purpose: mdb.RpcNodePurposeGeneral, Status: mdb.RpcNodeStatusUnknown},
		{Network: mdb.NetworkTron, Url: "https://manual.example.com", Type: mdb.RpcNodeTypeHttp, Enabled: true, Purpose: mdb.RpcNodePurposeManualVerify, Status: mdb.RpcNodeStatusUnknown},
	}
	if err := dao.Mdb.Create(&rows).Error; err != nil {
		t.Fatalf("seed rpc_nodes: %v", err)
	}
	got, err := ListRpcNodesForHealth()
	if err != nil {
		t.Fatalf("ListRpcNodesForHealth(): %v", err)
	}
	if len(got) != 1 || got[0].Url != "https://general.example.com" {
		t.Fatalf("ListRpcNodesForHealth() = %#v, want only general row", got)
	}
}
