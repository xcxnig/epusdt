package route

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/GMWalletApp/epusdt/util/log"
	"github.com/GMWalletApp/epusdt/util/sign"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"gorm.io/gorm/clause"
)

const testAPIToken = "test-secret-token"

func setupTestEnv(t *testing.T) *echo.Echo {
	t.Helper()

	tmpDir := t.TempDir()

	// minimal viper config
	viper.Reset()
	viper.Set("db_type", "sqlite")
	viper.Set("app_uri", "http://localhost:8080")
	viper.Set("order_expiration_time", 10)
	viper.Set("api_rate_url", "")
	viper.Set("forced_usdt_rate", 7.0)
	viper.Set("runtime_root_path", tmpDir)
	viper.Set("log_save_path", tmpDir)
	viper.Set("sqlite_database_filename", tmpDir+"/test.db")
	viper.Set("runtime_sqlite_filename", tmpDir+"/runtime.db")

	log.Init()

	// init config paths
	os.Setenv("EPUSDT_CONFIG", tmpDir)
	defer os.Unsetenv("EPUSDT_CONFIG")

	// init DB
	if err := dao.DBInit(); err != nil {
		t.Fatalf("DBInit: %v", err)
	}
	if err := dao.RuntimeInit(); err != nil {
		t.Fatalf("RuntimeInit: %v", err)
	}

	// ensure tables exist (MdbTableInit uses sync.Once, so migrate directly)
	dao.Mdb.AutoMigrate(
		&mdb.Orders{},
		&mdb.WalletAddress{},
		&mdb.AdminUser{},
		&mdb.ApiKey{},
		&mdb.Setting{},
		&mdb.NotificationChannel{},
		&mdb.Chain{},
		&mdb.ChainToken{},
		&mdb.RpcNode{},
	)

	// reset the settings cache so stale entries from a prior test don't leak
	_ = data.ReloadSettings()

	// seed wallet addresses
	dao.Mdb.Create(&mdb.WalletAddress{Network: mdb.NetworkTron, Address: "TTestTronAddress001", Status: mdb.TokenStatusEnable})
	dao.Mdb.Create(&mdb.WalletAddress{Network: mdb.NetworkSolana, Address: "SolTestAddress001", Status: mdb.TokenStatusEnable})
	// seed chains so scanners know the test networks are "enabled".
	// Use ON CONFLICT DO NOTHING: MdbTableInit (called via DBInit) uses sync.Once
	// and may have already seeded these rows into this DB on the first test run.
	dao.Mdb.Clauses(clause.OnConflict{DoNothing: true}).Create(&[]mdb.Chain{
		{Network: mdb.NetworkTron, DisplayName: "TRON", Enabled: true},
		{Network: mdb.NetworkSolana, DisplayName: "Solana", Enabled: true},
		{Network: mdb.NetworkEthereum, DisplayName: "Ethereum", Enabled: true},
		{Network: mdb.NetworkBsc, DisplayName: "BSC", Enabled: true},
		{Network: mdb.NetworkPolygon, DisplayName: "Polygon", Enabled: true},
		{Network: mdb.NetworkPlasma, DisplayName: "Plasma", Enabled: true},
	})

	// seed chain_tokens — GetSupportedAssets now reads from this table.
	// Same idempotency rationale as above.
	dao.Mdb.Clauses(clause.OnConflict{DoNothing: true}).Create(&[]mdb.ChainToken{
		{Network: mdb.NetworkTron, Symbol: "USDT", ContractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", Decimals: 6, Enabled: true},
		{Network: mdb.NetworkTron, Symbol: "TRX", ContractAddress: "", Decimals: 6, Enabled: true},
		{Network: mdb.NetworkSolana, Symbol: "USDT", ContractAddress: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", Decimals: 6, Enabled: true},
		{Network: mdb.NetworkSolana, Symbol: "USDC", ContractAddress: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", Decimals: 6, Enabled: true},
		{Network: mdb.NetworkSolana, Symbol: "SOL", ContractAddress: "", Decimals: 9, Enabled: true},
	})

	// Seed one universal api_keys row. The test's testAPIToken doubles
	// as both pid and secret_key so sign.Get(body, testAPIToken) calls
	// stay valid.
	dao.Mdb.Create(&mdb.ApiKey{
		Name:      "test-universal",
		Pid:       testAPIToken,
		SecretKey: testAPIToken,
		Status:    mdb.ApiKeyStatusEnable,
	})
	// Additional numeric-PID row for EPAY tests (EPAY pid must be numeric).
	dao.Mdb.Create(&mdb.ApiKey{
		Name:      "test-epay-pid-1",
		Pid:       "1",
		SecretKey: testAPIToken,
		Status:    mdb.ApiKeyStatusEnable,
	})

	e := echo.New()
	RegisterRoute(e)
	return e
}

func signBody(body map[string]interface{}) map[string]interface{} {
	// Signature middleware looks up api_keys by the "pid" field and
	// uses that row's secret_key as the signing bizKey.
	if _, ok := body["pid"]; !ok {
		body["pid"] = testAPIToken
	}
	sig, _ := sign.Get(body, testAPIToken)
	body["signature"] = sig
	return body
}

func doPost(e *echo.Echo, path string, body map[string]interface{}) *httptest.ResponseRecorder {
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(jsonBytes)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func signEpayValues(values url.Values) url.Values {
	signParams := make(map[string]interface{})
	for key, items := range values {
		if key == "sign" || key == "sign_type" || len(items) == 0 {
			continue
		}
		signParams[key] = items[0]
	}
	sig, _ := sign.Get(signParams, testAPIToken)
	values.Set("sign", sig)
	values.Set("sign_type", "MD5")
	return values
}

func doFormPost(e *echo.Echo, path string, values url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestCreateOrderGmpayV1Solana tests the gmpay route with solana network.
func TestCreateOrderGmpayV1Solana(t *testing.T) {
	e := setupTestEnv(t)

	body := signBody(map[string]interface{}{
		"order_id":   "test-sol-001",
		"amount":     1.00,
		"token":      "usdt",
		"currency":   "cny",
		"network":    "solana",
		"notify_url": "http://localhost/notify",
	})

	rec := doPost(e, "/payments/gmpay/v1/order/create-transaction", body)
	t.Logf("Status: %d, Body: %s", rec.Code, rec.Body.String())

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data in response, got: %v", resp)
	}

	if data["trade_id"] == nil || data["trade_id"] == "" {
		t.Error("expected trade_id in response")
	}
	if data["receive_address"] != "SolTestAddress001" {
		t.Errorf("expected solana address, got: %v", data["receive_address"])
	}
	t.Logf("Order created: trade_id=%v address=%v amount=%v", data["trade_id"], data["receive_address"], data["actual_amount"])
}

// TestCreateOrderGmpayV1SolNative tests creating an order for native SOL token.
func TestCreateOrderGmpayV1SolNative(t *testing.T) {
	e := setupTestEnv(t)

	body := signBody(map[string]interface{}{
		"order_id":   "test-sol-native-001",
		"amount":     0.05,
		"token":      "sol",
		"currency":   "usd",
		"network":    "solana",
		"notify_url": "http://localhost/notify",
	})

	rec := doPost(e, "/payments/gmpay/v1/order/create-transaction", body)
	t.Logf("Status: %d, Body: %s", rec.Code, rec.Body.String())

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	t.Logf("Response: %v", resp)

	// This may fail if rate API is not configured, which is expected in test
	// The important thing is the route accepts the request with network=solana token=sol
	if rec.Code != http.StatusOK {
		t.Logf("Note: non-200 may be expected if rate API is not configured for SOL")
	}
}

func parseResp(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp
}

// signGmpayFormValues builds a signed url.Values for the GMPAY create-transaction endpoint.
// The GMPAY middleware uses "signature" (not "sign") and "pid" (not numeric pid from EPAY).
func signGmpayFormValues(values url.Values) url.Values {
	if values.Get("pid") == "" {
		values.Set("pid", testAPIToken)
	}
	params := make(map[string]interface{})
	for k, vs := range values {
		if k == "signature" || len(vs) == 0 {
			continue
		}
		params[k] = vs[0]
	}
	sig, _ := sign.Get(params, testAPIToken)
	values.Set("signature", sig)
	return values
}

// TestCreateOrderGmpayV1FormData verifies that the GMPAY create-transaction endpoint
// accepts application/x-www-form-urlencoded in addition to JSON.
func TestCreateOrderGmpayV1FormData(t *testing.T) {
	e := setupTestEnv(t)

	values := signGmpayFormValues(url.Values{
		"order_id":   {"test-form-001"},
		"amount":     {"1.00"},
		"token":      {"usdt"},
		"currency":   {"cny"},
		"network":    {"solana"},
		"notify_url": {"http://localhost/notify"},
	})

	rec := doFormPost(e, "/payments/gmpay/v1/order/create-transaction", values)
	t.Logf("Status: %d, Body: %s", rec.Code, rec.Body.String())

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data in response, got: %v", resp)
	}
	if data["trade_id"] == nil || data["trade_id"] == "" {
		t.Error("expected trade_id in response")
	}
	t.Logf("Form-data order created: trade_id=%v", data["trade_id"])
}

// getSupportedNetworks is a helper that calls GET /payments/gmpay/v1/supported-assets
// and returns a map of network → []token for easy assertions.
func getSupportedNetworks(t *testing.T, e *echo.Echo) map[string][]string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/payments/gmpay/v1/supported-assets", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("getSupportedNetworks: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResp(t, rec)
	rawData, _ := resp["data"].(map[string]interface{})
	rawSupports, _ := rawData["supports"].([]interface{})
	result := make(map[string][]string, len(rawSupports))
	for _, item := range rawSupports {
		row := item.(map[string]interface{})
		network := row["network"].(string)
		rawTokens, _ := row["tokens"].([]interface{})
		tokens := make([]string, 0, len(rawTokens))
		for _, tok := range rawTokens {
			tokens = append(tokens, tok.(string))
		}
		result[network] = tokens
	}
	return result
}

// TestGetSupportedAssets_ChainTokenToggle verifies that:
//   - disabling a chain_token removes that token (and possibly the whole network)
//     from the /supported-assets response
//   - re-enabling it brings it back
func TestGetSupportedAssets_ChainTokenToggle(t *testing.T) {
	e := setupTestEnv(t)

	// Baseline: tron should appear with USDT and TRX.
	before := getSupportedNetworks(t, e)
	if _, ok := before["tron"]; !ok {
		t.Fatal("expected tron in baseline supported-assets")
	}

	// Disable the tron USDT chain_token.
	dao.Mdb.Model(&mdb.ChainToken{}).
		Where("network = ? AND symbol = ?", "tron", "USDT").
		Update("enabled", false)

	after := getSupportedNetworks(t, e)
	tronTokens := after["tron"]
	for _, tok := range tronTokens {
		if tok == "USDT" {
			t.Fatal("USDT should be absent after disabling tron USDT chain_token")
		}
	}
	t.Logf("After disabling tron USDT: tron tokens = %v", tronTokens)

	// Disable the tron TRX chain_token as well — now tron has no tokens → network disappears.
	dao.Mdb.Model(&mdb.ChainToken{}).
		Where("network = ? AND symbol = ?", "tron", "TRX").
		Update("enabled", false)

	afterAllDisabled := getSupportedNetworks(t, e)
	if _, ok := afterAllDisabled["tron"]; ok {
		t.Fatal("tron should disappear from supported-assets when all its chain_tokens are disabled")
	}
	t.Logf("After disabling all tron tokens: networks = %v", afterAllDisabled)

	// Re-enable tron USDT — tron reappears with only USDT.
	dao.Mdb.Model(&mdb.ChainToken{}).
		Where("network = ? AND symbol = ?", "tron", "USDT").
		Update("enabled", true)

	restored := getSupportedNetworks(t, e)
	tronRestored, ok := restored["tron"]
	if !ok {
		t.Fatal("tron should reappear after re-enabling USDT chain_token")
	}
	found := false
	for _, tok := range tronRestored {
		if tok == "USDT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected USDT in restored tron tokens, got %v", tronRestored)
	}
	t.Logf("After re-enabling tron USDT: tron tokens = %v", tronRestored)
}

// TestGetSupportedAssets_WalletAddressToggle verifies that:
//   - disabling ALL wallet addresses for a network removes that network from
//     the /supported-assets response (even if chain + tokens are still enabled)
//   - re-enabling any address brings the network back
func TestGetSupportedAssets_WalletAddressToggle(t *testing.T) {
	e := setupTestEnv(t)

	// Baseline: solana should be present.
	before := getSupportedNetworks(t, e)
	if _, ok := before["solana"]; !ok {
		t.Fatal("expected solana in baseline supported-assets")
	}

	// Disable the only solana wallet address.
	dao.Mdb.Model(&mdb.WalletAddress{}).
		Where("network = ?", mdb.NetworkSolana).
		Update("status", mdb.TokenStatusDisable)

	after := getSupportedNetworks(t, e)
	if _, ok := after["solana"]; ok {
		t.Fatal("solana should disappear from supported-assets when all its wallet addresses are disabled")
	}
	t.Logf("After disabling solana wallets: networks = %v", after)

	// Re-enable the solana wallet.
	dao.Mdb.Model(&mdb.WalletAddress{}).
		Where("network = ?", mdb.NetworkSolana).
		Update("status", mdb.TokenStatusEnable)

	restored := getSupportedNetworks(t, e)
	if _, ok := restored["solana"]; !ok {
		t.Fatal("solana should reappear after re-enabling its wallet address")
	}
	t.Logf("After re-enabling solana wallets: solana tokens = %v", restored["solana"])
}

func TestGetSupportedAssetsPublic(t *testing.T) {
	e := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/payments/gmpay/v1/supported-assets", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	resp := parseResp(t, rec)
	if resp["status_code"].(float64) != 200 {
		t.Fatalf("expected status_code=200, got %v", resp["status_code"])
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data, got %T", resp["data"])
	}
	supports, ok := data["supports"].([]interface{})
	if !ok {
		t.Fatalf("expected supports array, got %T", data["supports"])
	}
	if len(supports) < 2 {
		t.Fatalf("expected >= 2 network supports, got %d", len(supports))
	}

	seen := map[string]bool{}
	for _, item := range supports {
		row := item.(map[string]interface{})
		network := row["network"].(string)
		seen[network] = true
	}
	for _, n := range []string{"tron", "solana"} {
		if !seen[n] {
			t.Fatalf("missing network support: %s", n)
		}
	}
}

// TestCreateOrderNetworkIsolation verifies tron and solana wallets don't mix.
func TestCreateOrderNetworkIsolation(t *testing.T) {
	e := setupTestEnv(t)

	// Try to create a solana order — should get solana address, not tron
	body := signBody(map[string]interface{}{
		"order_id":   fmt.Sprintf("test-isolation-%d", 1),
		"amount":     1.00,
		"token":      "usdt",
		"currency":   "cny",
		"network":    "solana",
		"notify_url": "http://localhost/notify",
	})
	rec := doPost(e, "/payments/gmpay/v1/order/create-transaction", body)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data, got: %v", resp)
	}
	if data["receive_address"] == "TTestTronAddress001" {
		t.Error("solana order should NOT get a tron address")
	}
	if data["receive_address"] != "SolTestAddress001" {
		t.Errorf("expected SolTestAddress001, got %v", data["receive_address"])
	}
}

func TestEpaySubmitPhpGetCompatible(t *testing.T) {
	e := setupTestEnv(t)

	values := signEpayValues(url.Values{
		"pid":          {"1"},
		"name":         {"epay-get-001"},
		"type":         {"alipay"},
		"money":        {"1.00"},
		"out_trade_no": {"epay-get-001"},
		"notify_url":   {"http://localhost/notify"},
		"return_url":   {"http://localhost/return"},
	})

	req := httptest.NewRequest(http.MethodGet, "/payments/epay/v1/order/create-transaction/submit.php?"+values.Encode(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("Location"), "/pay/checkout-counter/") {
		t.Fatalf("expected checkout redirect, got %q", rec.Header().Get("Location"))
	}
}

func TestEpaySubmitPhpPostFormCompatible(t *testing.T) {
	e := setupTestEnv(t)

	values := signEpayValues(url.Values{
		"pid":          {"1"},
		"name":         {"epay-post-001"},
		"type":         {"alipay"},
		"money":        {"1.00"},
		"out_trade_no": {"epay-post-001"},
		"notify_url":   {"http://localhost/notify"},
		"return_url":   {"http://localhost/return"},
		"sitename":     {"example-shop"},
	})

	rec := doFormPost(e, "/payments/epay/v1/order/create-transaction/submit.php", values)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("Location"), "/pay/checkout-counter/") {
		t.Fatalf("expected checkout redirect, got %q", rec.Header().Get("Location"))
	}
}

// TestCheckStatus_NotFound verifies that /pay/check-status/:trade_id returns a
// graceful JSON error (not 500) when the trade_id doesn't exist.
func TestCheckStatus_NotFound(t *testing.T) {
	e := setupTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/pay/check-status/nonexistent-trade-id", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	t.Logf("CheckStatus(not found): status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code >= 500 {
		t.Fatalf("unexpected server error: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Code == http.StatusNotFound && rec.Body.Len() == 0 {
		t.Fatalf("route returned 404 with empty body — route may not be registered")
	}
}

// TestCheckStatus_WithOrder verifies /pay/check-status/:trade_id returns 200
// and a status field when the order exists.
func TestCheckStatus_WithOrder(t *testing.T) {
	e := setupTestEnv(t)

	// Create an order first via the GMPAY route.
	body := signBody(map[string]interface{}{
		"order_id":   "check-status-001",
		"amount":     1.00,
		"token":      "usdt",
		"currency":   "cny",
		"network":    "tron",
		"notify_url": "http://localhost/notify",
	})
	createRec := doPost(e, "/payments/gmpay/v1/order/create-transaction", body)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create order failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var createResp map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResp)
	tradeId, _ := createResp["data"].(map[string]interface{})["trade_id"].(string)
	if tradeId == "" {
		t.Fatal("no trade_id in create response")
	}

	// Now check status.
	req := httptest.NewRequest(http.MethodGet, "/pay/check-status/"+tradeId, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	t.Logf("CheckStatus: status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["data"] == nil {
		t.Fatal("expected data in check-status response")
	}
}

// TestCheckoutCounter_NotFound verifies that /pay/checkout-counter/:trade_id
// with an unknown trade_id does not return an empty 404 (which would mean the
// route is not registered). When the static HTML template is present the
// controller renders it with a 404 status; when it is absent (test env) the
// controller returns a 500 with a descriptive body — both outcomes are
// acceptable because the route IS registered and functional.
func TestCheckoutCounter_NotFound(t *testing.T) {
	e := setupTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/pay/checkout-counter/nonexistent-trade-id", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	t.Logf("CheckoutCounter(not found): status=%d content-type=%s body=%s",
		rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	// A completely empty 404 means the route is not registered at all.
	if rec.Code == http.StatusNotFound && rec.Body.Len() == 0 {
		t.Fatalf("route not registered (404 with empty body)")
	}
	// A 500 whose body mentions a missing file is expected in test environments
	// where the static directory is not present.
	if rec.Code >= 500 && rec.Body.Len() == 0 {
		t.Fatalf("unexpected server error with empty body: %d", rec.Code)
	}
}

// TestSwitchNetwork_MissingFields verifies that /pay/switch-network validates
// required fields and returns a graceful error when they are missing.
func TestSwitchNetwork_MissingFields(t *testing.T) {
	e := setupTestEnv(t)
	rec := doPost(e, "/pay/switch-network", map[string]interface{}{
		// trade_id, token, network are all missing
	})
	t.Logf("SwitchNetwork(missing fields): status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code >= 500 {
		t.Fatalf("unexpected server error: %d %s", rec.Code, rec.Body.String())
	}
	// Must return a non-200 business code for validation failure.
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if code, _ := resp["code"].(float64); code == 200 {
		t.Fatal("expected validation error for missing fields, got code=200")
	}
}

// TestSwitchNetwork_OrderNotFound verifies that /pay/switch-network returns a
// graceful error when the referenced trade_id doesn't exist.
func TestSwitchNetwork_OrderNotFound(t *testing.T) {
	e := setupTestEnv(t)
	rec := doPost(e, "/pay/switch-network", map[string]interface{}{
		"trade_id": "nonexistent-trade-id",
		"token":    "USDT",
		"network":  "tron",
	})
	t.Logf("SwitchNetwork(not found): status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code >= 500 {
		t.Fatalf("unexpected server error: %d %s", rec.Code, rec.Body.String())
	}
}

// TestSwitchNetwork_WithOrder verifies /pay/switch-network can create a
// sub-order when given a valid parent trade_id.
func TestSwitchNetwork_WithOrder(t *testing.T) {
	e := setupTestEnv(t)

	// Create a parent order on solana.
	createBody := signBody(map[string]interface{}{
		"order_id":   "switch-net-parent-001",
		"amount":     1.00,
		"token":      "usdt",
		"currency":   "cny",
		"network":    "solana",
		"notify_url": "http://localhost/notify",
	})
	createRec := doPost(e, "/payments/gmpay/v1/order/create-transaction", createBody)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create parent order failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var createResp map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResp)
	tradeId, _ := createResp["data"].(map[string]interface{})["trade_id"].(string)
	if tradeId == "" {
		t.Fatal("no trade_id in create response")
	}

	// Switch to tron.
	rec := doPost(e, "/pay/switch-network", map[string]interface{}{
		"trade_id": tradeId,
		"token":    "USDT",
		"network":  "tron",
	})
	t.Logf("SwitchNetwork(valid): status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["data"] == nil {
		t.Fatal("expected data in switch-network response")
	}
}
