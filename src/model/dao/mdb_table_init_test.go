package dao

import (
	"path/filepath"
	"testing"

	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/libtnb/sqlite"
	"github.com/spf13/viper"
	"gorm.io/gorm"
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

func TestDefaultRpcNodesIncludesTonLiteGeneralNode(t *testing.T) {
	var got *mdb.RpcNode
	nodes := defaultRpcNodes()
	for i := range nodes {
		node := nodes[i]
		if node.Network == mdb.NetworkTon && node.Type == mdb.RpcNodeTypeLite {
			got = &node
			break
		}
	}

	if got == nil {
		t.Fatal("missing TON lite seed rpc node")
	}
	if got.Url != "https://ton-blockchain.github.io/global.config.json" {
		t.Fatalf("TON lite seed url = %q", got.Url)
	}
	if got.Purpose != mdb.RpcNodePurposeGeneral {
		t.Fatalf("TON lite seed purpose = %q, want %q", got.Purpose, mdb.RpcNodePurposeGeneral)
	}
	if !got.Enabled {
		t.Fatal("TON lite seed enabled = false, want true")
	}
	if got.Status != mdb.RpcNodeStatusUnknown {
		t.Fatalf("TON lite seed status = %q, want %q", got.Status, mdb.RpcNodeStatusUnknown)
	}
}

func TestSeedDefaultSettingsIncludesSystemLogLevel(t *testing.T) {
	db := setupSeedSettingsTestDB(t)
	Mdb = db

	seedDefaultSettings()

	var row mdb.Setting
	if err := Mdb.Where("`key` = ?", mdb.SettingKeySystemLogLevel).Take(&row).Error; err != nil {
		t.Fatalf("load system.log_level seed: %v", err)
	}
	if row.Group != mdb.SettingGroupSystem {
		t.Fatalf("system.log_level group = %q, want %q", row.Group, mdb.SettingGroupSystem)
	}
	if row.Value != mdb.SettingDefaultSystemLogLevel {
		t.Fatalf("system.log_level value = %q, want %q", row.Value, mdb.SettingDefaultSystemLogLevel)
	}
	if row.Type != mdb.SettingTypeString {
		t.Fatalf("system.log_level type = %q, want %q", row.Type, mdb.SettingTypeString)
	}
}

func TestSeedDefaultSettingsDoesNotOverwriteSystemLogLevel(t *testing.T) {
	db := setupSeedSettingsTestDB(t)
	Mdb = db
	if err := Mdb.Create(&mdb.Setting{
		Group: mdb.SettingGroupSystem,
		Key:   mdb.SettingKeySystemLogLevel,
		Value: "debug",
		Type:  mdb.SettingTypeString,
	}).Error; err != nil {
		t.Fatalf("precreate system.log_level: %v", err)
	}

	seedDefaultSettings()

	var row mdb.Setting
	if err := Mdb.Where("`key` = ?", mdb.SettingKeySystemLogLevel).Take(&row).Error; err != nil {
		t.Fatalf("load system.log_level seed: %v", err)
	}
	if row.Value != "debug" {
		t.Fatalf("system.log_level value = %q, want existing debug", row.Value)
	}
}

func setupSeedSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := Mdb
	viper.Reset()
	viper.Set("app_uri", "https://example.com")
	t.Cleanup(func() {
		Mdb = oldDB
		viper.Reset()
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "seed-settings.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&mdb.Setting{}); err != nil {
		t.Fatalf("migrate settings: %v", err)
	}
	return db
}
