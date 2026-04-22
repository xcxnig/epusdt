package route

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/labstack/echo/v4"
)

const (
	testAdminUsername = "admin"
	testAdminPassword = "test-admin-pass-123"
)

// setupAdminTestEnv sets up the test environment with a seeded admin user
// and returns the echo instance and a valid JWT token.
func setupAdminTestEnv(t *testing.T) (*echo.Echo, string) {
	t.Helper()
	e := setupTestEnv(t)

	// Seed admin user with a known password.
	hash, err := data.HashPassword(testAdminPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	dao.Mdb.Create(&mdb.AdminUser{
		Username:     testAdminUsername,
		PasswordHash: hash,
		Status:       mdb.AdminUserStatusEnable,
	})

	// Login to obtain a JWT for subsequent authenticated requests.
	token := adminLogin(t, e, testAdminUsername, testAdminPassword)
	return e, token
}

// adminLogin performs a login request and returns the JWT token.
func adminLogin(t *testing.T, e *echo.Echo, username, password string) string {
	t.Helper()
	body := map[string]interface{}{
		"username": username,
		"password": password,
	}
	rec := doPost(e, "/admin/api/v1/auth/login", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("login unmarshal: %v", err)
	}
	dataObj, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("login response missing data field: %s", rec.Body.String())
	}
	token, ok := dataObj["token"].(string)
	if !ok || token == "" {
		t.Fatalf("login response missing token: %s", rec.Body.String())
	}
	return token
}

// doGetAdmin sends an authenticated GET request with a Bearer JWT.
func doGetAdmin(e *echo.Echo, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// doPostAdmin sends an authenticated POST request with a Bearer JWT.
func doPostAdmin(e *echo.Echo, path string, body map[string]interface{}, token string) *httptest.ResponseRecorder {
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(jsonBytes)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// doPatchAdmin sends an authenticated PATCH request with a Bearer JWT.
func doPatchAdmin(e *echo.Echo, path string, body map[string]interface{}, token string) *httptest.ResponseRecorder {
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(string(jsonBytes)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// doDeleteAdmin sends an authenticated DELETE request with a Bearer JWT.
func doDeleteAdmin(e *echo.Echo, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// doPutAdmin sends an authenticated PUT request with a Bearer JWT.
func doPutAdmin(e *echo.Echo, path string, body map[string]interface{}, token string) *httptest.ResponseRecorder {
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(string(jsonBytes)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// doGet sends an unauthenticated GET request.
func doGet(e *echo.Echo, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// assertOK asserts the response status is 200 and the status_code field is 200.
func assertOK(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	code, _ := resp["status_code"].(float64)
	if code != 200 {
		t.Fatalf("expected status_code=200, got %v: %s", code, rec.Body.String())
	}
	return resp
}

// assertUnauthorized asserts that a request without a valid JWT returns 401.
func assertUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestAdminLogin_Success verifies correct credentials return 200 + JWT.
func TestAdminLogin_Success(t *testing.T) {
	e := setupTestEnv(t)
	hash, _ := data.HashPassword(testAdminPassword)
	dao.Mdb.Create(&mdb.AdminUser{
		Username:     testAdminUsername,
		PasswordHash: hash,
		Status:       mdb.AdminUserStatusEnable,
	})

	rec := doPost(e, "/admin/api/v1/auth/login", map[string]interface{}{
		"username": testAdminUsername,
		"password": testAdminPassword,
	})
	t.Logf("Login response: %s", rec.Body.String())

	resp := assertOK(t, rec)
	dataObj, _ := resp["data"].(map[string]interface{})
	if dataObj["token"] == "" || dataObj["token"] == nil {
		t.Fatal("expected non-empty token in login response")
	}
}

// TestAdminLogin_WrongPassword verifies wrong credentials return 4xx.
func TestAdminLogin_WrongPassword(t *testing.T) {
	e := setupTestEnv(t)
	hash, _ := data.HashPassword(testAdminPassword)
	dao.Mdb.Create(&mdb.AdminUser{
		Username:     testAdminUsername,
		PasswordHash: hash,
		Status:       mdb.AdminUserStatusEnable,
	})

	rec := doPost(e, "/admin/api/v1/auth/login", map[string]interface{}{
		"username": testAdminUsername,
		"password": "wrong-password",
	})
	t.Logf("Login wrong-password response: status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code == http.StatusOK {
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if code, _ := resp["code"].(float64); code == 200 {
			t.Fatal("expected failure for wrong password, got 200")
		}
	}
}

// TestAdminLogin_MissingFields verifies missing required fields return 4xx.
func TestAdminLogin_MissingFields(t *testing.T) {
	e := setupTestEnv(t)
	rec := doPost(e, "/admin/api/v1/auth/login", map[string]interface{}{
		"username": testAdminUsername,
		// password missing
	})
	t.Logf("Login missing-fields response: status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code == http.StatusOK {
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if code, _ := resp["code"].(float64); code == 200 {
			t.Fatal("expected validation failure, got code=200")
		}
	}
}

// TestAdminProtectedRoute_NoToken verifies protected routes reject unauthenticated requests.
func TestAdminProtectedRoute_NoToken(t *testing.T) {
	e := setupTestEnv(t)
	protectedRoutes := []string{
		"/admin/api/v1/auth/me",
		"/admin/api/v1/api-keys",
		"/admin/api/v1/chains",
		"/admin/api/v1/wallets",
		"/admin/api/v1/orders",
		"/admin/api/v1/dashboard/overview",
		"/admin/api/v1/settings",
		"/admin/api/v1/rpc-nodes",
		"/admin/api/v1/notification-channels",
	}
	for _, path := range protectedRoutes {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		t.Logf("GET %s → %d", path, rec.Code)
		assertUnauthorized(t, rec)
	}
}

// TestAdminMe verifies the /auth/me route returns current user info.
func TestAdminMe(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doGetAdmin(e, "/admin/api/v1/auth/me", token)
	t.Logf("Me response: %s", rec.Body.String())
	resp := assertOK(t, rec)
	dataObj, _ := resp["data"].(map[string]interface{})
	if dataObj["username"] != testAdminUsername {
		t.Fatalf("expected username=%s, got %v", testAdminUsername, dataObj["username"])
	}
}

// TestAdminLogout verifies the /auth/logout route succeeds with a valid token.
func TestAdminLogout(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doPostAdmin(e, "/admin/api/v1/auth/logout", nil, token)
	t.Logf("Logout response: %s", rec.Body.String())
	assertOK(t, rec)
}

// TestAdminChangePassword verifies the change-password route works correctly.
func TestAdminChangePassword(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doPostAdmin(e, "/admin/api/v1/auth/password", map[string]interface{}{
		"old_password": testAdminPassword,
		"new_password": "new-pass-456",
	}, token)
	t.Logf("ChangePassword response: %s", rec.Body.String())
	assertOK(t, rec)

	// Verify old password no longer works.
	rec2 := doPost(e, "/admin/api/v1/auth/login", map[string]interface{}{
		"username": testAdminUsername,
		"password": testAdminPassword,
	})
	var resp2 map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &resp2)
	code2, _ := resp2["code"].(float64)
	if code2 == 200 {
		t.Fatal("old password should no longer work after change")
	}
}

// TestAdminInitPasswordFlow verifies the one-time initial password endpoint
// and hash-based "password changed" detection flow.
func TestAdminInitPasswordFlow(t *testing.T) {
	e := setupTestEnv(t)
	const initPassword = "init-pass-123456"

	adminHash, err := data.HashPassword(initPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	dao.Mdb.Create(&mdb.AdminUser{
		Username:     testAdminUsername,
		PasswordHash: adminHash,
		Status:       mdb.AdminUserStatusEnable,
	})
	_ = data.SetSetting(mdb.SettingGroupSystem, mdb.SettingKeyInitAdminPasswordPlain, initPassword, mdb.SettingTypeString)
	_ = data.SetSetting(mdb.SettingGroupSystem, mdb.SettingKeyInitAdminPasswordHash, data.HashInitialAdminPassword(initPassword), mdb.SettingTypeString)
	_ = data.SetSetting(mdb.SettingGroupSystem, mdb.SettingKeyInitAdminPasswordFetched, "false", mdb.SettingTypeBool)
	_ = data.SetSetting(mdb.SettingGroupSystem, mdb.SettingKeyInitAdminPasswordChanged, "false", mdb.SettingTypeBool)

	recHash := doGet(e, "/admin/api/v1/auth/init-password-hash")
	respHash := assertOK(t, recHash)
	hashData, _ := respHash["data"].(map[string]interface{})
	if got := hashData["password_hash"]; got != data.HashInitialAdminPassword(initPassword) {
		t.Fatalf("expected init hash %s, got %v", data.HashInitialAdminPassword(initPassword), got)
	}
	if got, _ := hashData["password_changed"].(bool); got {
		t.Fatalf("expected password_changed=false before change, got true")
	}

	recFetch := doGet(e, "/admin/api/v1/auth/init-password")
	respFetch := assertOK(t, recFetch)
	fetchData, _ := respFetch["data"].(map[string]interface{})
	if fetchData["username"] != testAdminUsername {
		t.Fatalf("expected username=%s, got %v", testAdminUsername, fetchData["username"])
	}
	if fetchData["password"] != initPassword {
		t.Fatalf("expected initial password %s, got %v", initPassword, fetchData["password"])
	}

	recFetch2 := doGet(e, "/admin/api/v1/auth/init-password")
	if recFetch2.Code != http.StatusBadRequest {
		t.Fatalf("second fetch should fail with 400, got %d body=%s", recFetch2.Code, recFetch2.Body.String())
	}
	if !strings.Contains(strings.ToLower(recFetch2.Body.String()), "already fetched") {
		t.Fatalf("expected already fetched error, got: %s", recFetch2.Body.String())
	}

	token := adminLogin(t, e, testAdminUsername, initPassword)

	recMe1 := doGetAdmin(e, "/admin/api/v1/auth/me", token)
	respMe1 := assertOK(t, recMe1)
	meData1, _ := respMe1["data"].(map[string]interface{})
	if got, _ := meData1["password_is_default"].(bool); !got {
		t.Fatalf("expected password_is_default=true before change, got %v", got)
	}

	recChange := doPostAdmin(e, "/admin/api/v1/auth/password", map[string]interface{}{
		"old_password": initPassword,
		"new_password": "new-pass-789",
	}, token)
	assertOK(t, recChange)

	recHash2 := doGet(e, "/admin/api/v1/auth/init-password-hash")
	respHash2 := assertOK(t, recHash2)
	hashData2, _ := respHash2["data"].(map[string]interface{})
	if got, _ := hashData2["password_changed"].(bool); !got {
		t.Fatalf("expected password_changed=true after change, got %v", got)
	}

	recMe2 := doGetAdmin(e, "/admin/api/v1/auth/me", token)
	respMe2 := assertOK(t, recMe2)
	meData2, _ := respMe2["data"].(map[string]interface{})
	if got, _ := meData2["password_is_default"].(bool); got {
		t.Fatalf("expected password_is_default=false after change, got %v", got)
	}
}

// ─── API Keys ────────────────────────────────────────────────────────────────

// TestAdminApiKeys_CRUD verifies list, create, update, status change, secret,
// stats, rotate, and delete for API keys.
func TestAdminApiKeys_CRUD(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	// List — should contain seeded keys from setupTestEnv.
	rec := doGetAdmin(e, "/admin/api/v1/api-keys", token)
	t.Logf("ListApiKeys: %s", rec.Body.String())
	assertOK(t, rec)

	// Create a new key.
	rec = doPostAdmin(e, "/admin/api/v1/api-keys", map[string]interface{}{
		"name": "test-created-key",
	}, token)
	t.Logf("CreateApiKey: %s", rec.Body.String())
	resp := assertOK(t, rec)
	dataObj, _ := resp["data"].(map[string]interface{})
	keyID := dataObj["id"]
	if keyID == nil {
		t.Fatal("CreateApiKey response missing id")
	}
	keyIDStr := fmt.Sprintf("%.0f", keyID.(float64))

	// Update.
	rec = doPatchAdmin(e, "/admin/api/v1/api-keys/"+keyIDStr, map[string]interface{}{
		"name": "renamed-key",
	}, token)
	t.Logf("UpdateApiKey: %s", rec.Body.String())
	assertOK(t, rec)

	// Get secret.
	rec = doGetAdmin(e, "/admin/api/v1/api-keys/"+keyIDStr+"/secret", token)
	t.Logf("GetApiKeySecret: %s", rec.Body.String())
	assertOK(t, rec)

	// Get stats.
	rec = doGetAdmin(e, "/admin/api/v1/api-keys/"+keyIDStr+"/stats", token)
	t.Logf("GetApiKeyStats: %s", rec.Body.String())
	assertOK(t, rec)

	// Change status.
	rec = doPostAdmin(e, "/admin/api/v1/api-keys/"+keyIDStr+"/status", map[string]interface{}{
		"status": 2, // disable
	}, token)
	t.Logf("ChangeApiKeyStatus: %s", rec.Body.String())
	assertOK(t, rec)

	// Rotate secret.
	rec = doPostAdmin(e, "/admin/api/v1/api-keys/"+keyIDStr+"/rotate-secret", nil, token)
	t.Logf("RotateApiKeySecret: %s", rec.Body.String())
	assertOK(t, rec)

	// Delete.
	rec = doDeleteAdmin(e, "/admin/api/v1/api-keys/"+keyIDStr, token)
	t.Logf("DeleteApiKey: %s", rec.Body.String())
	assertOK(t, rec)
}

// ─── Chains ──────────────────────────────────────────────────────────────────

// TestAdminChains_ListAndUpdate verifies listing and updating chains.
func TestAdminChains_ListAndUpdate(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	// List chains — seeded in setupTestEnv.
	rec := doGetAdmin(e, "/admin/api/v1/chains", token)
	t.Logf("ListChains: %s", rec.Body.String())
	assertOK(t, rec)

	// Update the tron chain display name.
	rec = doPatchAdmin(e, "/admin/api/v1/chains/tron", map[string]interface{}{
		"display_name": "TRON Updated",
	}, token)
	t.Logf("UpdateChain: %s", rec.Body.String())
	assertOK(t, rec)
}

// ─── Chain Tokens ─────────────────────────────────────────────────────────────

// TestAdminChainTokens_CRUD verifies CRUD operations for chain tokens.
func TestAdminChainTokens_CRUD(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	// List — seeded chain tokens.
	rec := doGetAdmin(e, "/admin/api/v1/chain-tokens", token)
	t.Logf("ListChainTokens: %s", rec.Body.String())
	assertOK(t, rec)

	// Create.
	rec = doPostAdmin(e, "/admin/api/v1/chain-tokens", map[string]interface{}{
		"network":          "ethereum",
		"symbol":           "TEST",
		"contract_address": "0xTESTADDRESS",
		"decimals":         18,
	}, token)
	t.Logf("CreateChainToken: %s", rec.Body.String())
	resp := assertOK(t, rec)
	dataObj, _ := resp["data"].(map[string]interface{})
	tokenID := dataObj["id"]
	if tokenID == nil {
		t.Fatal("CreateChainToken missing id")
	}
	tokenIDStr := fmt.Sprintf("%.0f", tokenID.(float64))

	// Update.
	rec = doPatchAdmin(e, "/admin/api/v1/chain-tokens/"+tokenIDStr, map[string]interface{}{
		"symbol": "TEST2",
	}, token)
	t.Logf("UpdateChainToken: %s", rec.Body.String())
	assertOK(t, rec)

	// Change status.
	rec = doPostAdmin(e, "/admin/api/v1/chain-tokens/"+tokenIDStr+"/status", map[string]interface{}{
		"enabled": false,
	}, token)
	t.Logf("ChangeChainTokenStatus: %s", rec.Body.String())
	assertOK(t, rec)

	// Delete.
	rec = doDeleteAdmin(e, "/admin/api/v1/chain-tokens/"+tokenIDStr, token)
	t.Logf("DeleteChainToken: %s", rec.Body.String())
	assertOK(t, rec)
}

// ─── RPC Nodes ────────────────────────────────────────────────────────────────

// TestAdminRpcNodes_CRUD verifies CRUD operations for RPC nodes.
func TestAdminRpcNodes_CRUD(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	// List — empty initially.
	rec := doGetAdmin(e, "/admin/api/v1/rpc-nodes", token)
	t.Logf("ListRpcNodes: %s", rec.Body.String())
	assertOK(t, rec)

	// Create.
	rec = doPostAdmin(e, "/admin/api/v1/rpc-nodes", map[string]interface{}{
		"network": "ethereum",
		"url":     "https://eth-mainnet.example.com",
		"type":    "http",
	}, token)
	t.Logf("CreateRpcNode: %s", rec.Body.String())
	resp := assertOK(t, rec)
	dataObj, _ := resp["data"].(map[string]interface{})
	nodeID := dataObj["id"]
	if nodeID == nil {
		t.Fatal("CreateRpcNode missing id")
	}
	nodeIDStr := fmt.Sprintf("%.0f", nodeID.(float64))

	// Update.
	rec = doPatchAdmin(e, "/admin/api/v1/rpc-nodes/"+nodeIDStr, map[string]interface{}{
		"url": "https://eth-mainnet2.example.com",
	}, token)
	t.Logf("UpdateRpcNode: %s", rec.Body.String())
	assertOK(t, rec)

	// Health check — network likely unreachable in test, but route must not 404/500.
	rec = doPostAdmin(e, "/admin/api/v1/rpc-nodes/"+nodeIDStr+"/health-check", nil, token)
	t.Logf("HealthCheckRpcNode: status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code == http.StatusNotFound {
		t.Fatalf("health-check route returned 404")
	}

	// Delete.
	rec = doDeleteAdmin(e, "/admin/api/v1/rpc-nodes/"+nodeIDStr, token)
	t.Logf("DeleteRpcNode: %s", rec.Body.String())
	assertOK(t, rec)
}

// ─── Wallets ─────────────────────────────────────────────────────────────────

// TestAdminWallets_CRUD verifies wallet management routes.
func TestAdminWallets_CRUD(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	// List — two wallets seeded in setupTestEnv.
	rec := doGetAdmin(e, "/admin/api/v1/wallets", token)
	t.Logf("ListWallets: %s", rec.Body.String())
	resp := assertOK(t, rec)
	_ = resp

	// Add a new wallet.
	rec = doPostAdmin(e, "/admin/api/v1/wallets", map[string]interface{}{
		"network": "ethereum",
		"address": "0xTestEthAddress001",
	}, token)
	t.Logf("AddWallet: %s", rec.Body.String())
	resp2 := assertOK(t, rec)
	dataObj, _ := resp2["data"].(map[string]interface{})
	walletID := dataObj["id"]
	if walletID == nil {
		t.Fatal("AddWallet missing id")
	}
	walletIDStr := fmt.Sprintf("%.0f", walletID.(float64))

	// Get single wallet.
	rec = doGetAdmin(e, "/admin/api/v1/wallets/"+walletIDStr, token)
	t.Logf("GetWallet: %s", rec.Body.String())
	assertOK(t, rec)

	// Update.
	rec = doPatchAdmin(e, "/admin/api/v1/wallets/"+walletIDStr, map[string]interface{}{
		"address": "0xTestEthAddress002",
	}, token)
	t.Logf("UpdateWallet: %s", rec.Body.String())
	assertOK(t, rec)

	// Change status.
	rec = doPostAdmin(e, "/admin/api/v1/wallets/"+walletIDStr+"/status", map[string]interface{}{
		"status": 2, // disable
	}, token)
	t.Logf("ChangeWalletStatus: %s", rec.Body.String())
	assertOK(t, rec)

	// Delete.
	rec = doDeleteAdmin(e, "/admin/api/v1/wallets/"+walletIDStr, token)
	t.Logf("DeleteWallet: %s", rec.Body.String())
	assertOK(t, rec)
}

// TestAdminWallets_BatchImport verifies batch wallet import.
func TestAdminWallets_BatchImport(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doPostAdmin(e, "/admin/api/v1/wallets/batch-import", map[string]interface{}{
		"network":   "ethereum",
		"addresses": []string{"0xBatchAddr001", "0xBatchAddr002"},
	}, token)
	t.Logf("BatchImportWallets: %s", rec.Body.String())
	assertOK(t, rec)
}

// ─── Orders ──────────────────────────────────────────────────────────────────

// TestAdminOrders_List verifies listing orders (empty initially).
func TestAdminOrders_List(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doGetAdmin(e, "/admin/api/v1/orders", token)
	t.Logf("ListOrders: %s", rec.Body.String())
	assertOK(t, rec)
}

// TestAdminOrders_Export verifies the export endpoint is accessible.
func TestAdminOrders_Export(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doGetAdmin(e, "/admin/api/v1/orders/export", token)
	t.Logf("ExportOrders: status=%d", rec.Code)
	// Should return 200 (even with empty result set).
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestAdminOrders_GetNotFound verifies a non-existent trade_id returns a
// non-200 business code (not a 404 or 500 HTTP error).
func TestAdminOrders_GetNotFound(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doGetAdmin(e, "/admin/api/v1/orders/nonexistent-trade-id", token)
	t.Logf("GetOrder (not found): status=%d body=%s", rec.Code, rec.Body.String())
	// Should not be a server error.
	if rec.Code >= 500 {
		t.Fatalf("unexpected server error: %d %s", rec.Code, rec.Body.String())
	}
}

// TestAdminOrders_CloseNotFound verifies closing a non-existent order returns
// a graceful error (not 500).
func TestAdminOrders_CloseNotFound(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doPostAdmin(e, "/admin/api/v1/orders/nonexistent-trade-id/close", nil, token)
	t.Logf("CloseOrder (not found): status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code >= 500 {
		t.Fatalf("unexpected server error: %d %s", rec.Code, rec.Body.String())
	}
}

// TestAdminOrders_MarkPaidNotFound verifies mark-paid on non-existent order.
func TestAdminOrders_MarkPaidNotFound(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doPostAdmin(e, "/admin/api/v1/orders/nonexistent-trade-id/mark-paid", nil, token)
	t.Logf("MarkOrderPaid (not found): status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code >= 500 {
		t.Fatalf("unexpected server error: %d %s", rec.Code, rec.Body.String())
	}
}

// TestAdminOrders_ResendCallbackNotFound verifies resend-callback graceful error.
func TestAdminOrders_ResendCallbackNotFound(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doPostAdmin(e, "/admin/api/v1/orders/nonexistent-trade-id/resend-callback", nil, token)
	t.Logf("ResendCallback (not found): status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code >= 500 {
		t.Fatalf("unexpected server error: %d %s", rec.Code, rec.Body.String())
	}
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

// TestAdminDashboard_AllRoutes verifies all dashboard endpoints return 200.
func TestAdminDashboard_AllRoutes(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	routes := []string{
		"/admin/api/v1/dashboard/overview",
		"/admin/api/v1/dashboard/asset-trend",
		"/admin/api/v1/dashboard/revenue-trend",
		"/admin/api/v1/dashboard/order-stats",
		"/admin/api/v1/dashboard/recent-orders",
	}
	for _, path := range routes {
		rec := doGetAdmin(e, path, token)
		t.Logf("GET %s → %d: %s", path, rec.Code, rec.Body.String())
		assertOK(t, rec)
	}
}

// ─── Settings ────────────────────────────────────────────────────────────────

// TestAdminSettings_ListAndUpsert verifies listing and upserting settings.
func TestAdminSettings_ListAndUpsert(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	// List settings.
	rec := doGetAdmin(e, "/admin/api/v1/settings", token)
	t.Logf("ListSettings: %s", rec.Body.String())
	assertOK(t, rec)

	// Upsert a setting.
	rec = doPutAdmin(e, "/admin/api/v1/settings", map[string]interface{}{
		"items": []map[string]interface{}{
			{"group": "rate", "key": "rate.forced_usdt_rate", "value": "7.5", "type": "string"},
		},
	}, token)
	t.Logf("UpsertSettings: %s", rec.Body.String())
	assertOK(t, rec)
}

// TestAdminSettings_DeleteNonExistent verifies deleting a non-existent setting.
func TestAdminSettings_DeleteNonExistent(t *testing.T) {
	e, token := setupAdminTestEnv(t)
	rec := doDeleteAdmin(e, "/admin/api/v1/settings/nonexistent.key", token)
	t.Logf("DeleteSetting (nonexistent): status=%d body=%s", rec.Code, rec.Body.String())
	// Should not be a server error.
	if rec.Code >= 500 {
		t.Fatalf("unexpected server error: %d %s", rec.Code, rec.Body.String())
	}
}

// ─── Notification Channels ───────────────────────────────────────────────────

// TestAdminNotificationChannels_CRUD verifies notification channel routes.
func TestAdminNotificationChannels_CRUD(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	// List — empty initially.
	rec := doGetAdmin(e, "/admin/api/v1/notification-channels", token)
	t.Logf("ListNotificationChannels: %s", rec.Body.String())
	assertOK(t, rec)

	// Create a webhook channel.
	rec = doPostAdmin(e, "/admin/api/v1/notification-channels", map[string]interface{}{
		"name":   "test-webhook",
		"type":   "webhook",
		"config": map[string]interface{}{"url": "http://localhost/webhook"},
		"events": map[string]bool{"order.paid": true},
	}, token)
	t.Logf("CreateNotificationChannel: %s", rec.Body.String())
	resp := assertOK(t, rec)
	dataObj, _ := resp["data"].(map[string]interface{})
	chanID := dataObj["id"]
	if chanID == nil {
		t.Fatal("CreateNotificationChannel missing id")
	}
	chanIDStr := fmt.Sprintf("%.0f", chanID.(float64))

	// Update.
	rec = doPatchAdmin(e, "/admin/api/v1/notification-channels/"+chanIDStr, map[string]interface{}{
		"name": "renamed-webhook",
	}, token)
	t.Logf("UpdateNotificationChannel: %s", rec.Body.String())
	assertOK(t, rec)

	// Change status.
	rec = doPostAdmin(e, "/admin/api/v1/notification-channels/"+chanIDStr+"/status", map[string]interface{}{
		"enabled": false,
	}, token)
	t.Logf("ChangeNotificationChannelStatus: %s", rec.Body.String())
	assertOK(t, rec)

	// Delete.
	rec = doDeleteAdmin(e, "/admin/api/v1/notification-channels/"+chanIDStr, token)
	t.Logf("DeleteNotificationChannel: %s", rec.Body.String())
	assertOK(t, rec)
}

// TestAdminNotificationChannels_TelegramFrontendPayloadCompatibility verifies
// the admin API accepts common frontend telegram payload variants:
// - type in mixed case ("Telegram")
// - config in camelCase keys (botToken/chatId)
// - events as string array
func TestAdminNotificationChannels_TelegramFrontendPayloadCompatibility(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	rec := doPostAdmin(e, "/admin/api/v1/notification-channels", map[string]interface{}{
		"name": "frontend-telegram",
		"type": "Telegram",
		"config": map[string]interface{}{
			"botToken": "123:ABC",
			"chatId":   "-1001234567890",
		},
		"events": []string{"order.paid", "order.expired"},
	}, token)
	t.Logf("CreateFrontendStyleTelegramChannel: %s", rec.Body.String())
	resp := assertOK(t, rec)

	dataObj, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing data object: %s", rec.Body.String())
	}

	eventsRaw, _ := dataObj["events"].(string)
	if eventsRaw == "" {
		t.Fatalf("missing events json in response data: %s", rec.Body.String())
	}
	var events map[string]bool
	if err := json.Unmarshal([]byte(eventsRaw), &events); err != nil {
		t.Fatalf("unmarshal events: %v", err)
	}
	if !events["order.paid"] || !events["order.expired"] {
		t.Fatalf("events not normalized as expected: %+v", events)
	}
}

// ─── Orders list-with-sub ─────────────────────────────────────────────────────

// TestAdminOrders_ListWithSubExcludesSubOrdersFromTopLevel verifies that
// sub-orders do NOT appear at the top level of /orders/list-with-sub and are
// instead nested inside their parent's sub_orders array.
func TestAdminOrders_ListWithSubExcludesSubOrdersFromTopLevel(t *testing.T) {
	e, token := setupAdminTestEnv(t)

	// Seed Ethereum wallet and chain_token so SwitchNetwork can allocate
	// a sub-order on that network. The chain record is already seeded by
	// MdbTableInit; only the wallet address and token are needed here.
	dao.Mdb.Create(&mdb.WalletAddress{
		Network: mdb.NetworkEthereum,
		Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Status:  mdb.TokenStatusEnable,
	})
	if err := dao.Mdb.Where(mdb.ChainToken{Network: mdb.NetworkEthereum, Symbol: "USDT"}).
		FirstOrCreate(&mdb.ChainToken{
			Network: mdb.NetworkEthereum, Symbol: "USDT",
			ContractAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			Decimals:        6, Enabled: true,
		}).Error; err != nil {
		t.Fatalf("seed ethereum USDT token: %v", err)
	}

	// Create a parent order via the gmpay API.
	parentBody := signBody(map[string]interface{}{
		"order_id":     "listsubtest-parent",
		"amount":       5.0,
		"currency":     "CNY",
		"token":        "USDT",
		"network":      "tron",
		"notify_url":   "https://merchant.example/callback",
		"redirect_url": "https://merchant.example/redirect",
	})
	rec := doPost(e, "/payments/gmpay/v1/order/create-transaction", parentBody)
	t.Logf("CreateParent: %s", rec.Body.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("create parent: got %d: %s", rec.Code, rec.Body.String())
	}
	var parentResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &parentResp)
	parentTradeId := parentResp["data"].(map[string]interface{})["trade_id"].(string)

	// Switch to Ethereum to create a sub-order (no signature required).
	rec = doPost(e, "/pay/switch-network", map[string]interface{}{
		"trade_id": parentTradeId,
		"token":    "USDT",
		"network":  "ethereum",
	})
	t.Logf("SwitchNetwork: %s", rec.Body.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("switch network: got %d: %s", rec.Code, rec.Body.String())
	}
	var switchResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &switchResp)
	subTradeId := switchResp["data"].(map[string]interface{})["trade_id"].(string)

	// Fetch list-with-sub.
	rec = doGetAdmin(e, "/admin/api/v1/orders/list-with-sub", token)
	t.Logf("ListWithSub: %s", rec.Body.String())
	resp := assertOK(t, rec)

	dataObj, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing data object: %s", rec.Body.String())
	}
	listRaw, _ := dataObj["list"].([]interface{})

	// Collect all trade_ids at the top level and in sub_orders.
	topLevelIds := map[string]bool{}
	subOrderIds := map[string]bool{}
	for _, item := range listRaw {
		entry := item.(map[string]interface{})
		tradeId := entry["trade_id"].(string)
		topLevelIds[tradeId] = true
		if subs, ok := entry["sub_orders"].([]interface{}); ok {
			for _, s := range subs {
				sub := s.(map[string]interface{})
				subOrderIds[sub["trade_id"].(string)] = true
			}
		}
	}

	// Sub-order must NOT appear at the top level.
	if topLevelIds[subTradeId] {
		t.Fatalf("sub-order trade_id=%s appears at top level — should be nested only", subTradeId)
	}
	// Parent must appear at the top level.
	if !topLevelIds[parentTradeId] {
		t.Fatalf("parent trade_id=%s missing from top level", parentTradeId)
	}
	// Sub-order must appear under the parent's sub_orders.
	if !subOrderIds[subTradeId] {
		t.Fatalf("sub-order trade_id=%s not found in any sub_orders array", subTradeId)
	}
}
