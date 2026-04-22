// Package install provides a first-run setup REST API.
//
// When the .env config file is absent (or has install=true) the HTTP start
// command calls RunInstallServer, which listens on the same address the main
// server will eventually use (default :8000) and mounts two JSON endpoints
// under /install consumed by the frontend install UI:
//
//	GET  /install/defaults  — default field values for the form
//	POST /install           — validate + write .env, then shut down
//
// The HTTP listen address is submitted as two separate fields (http_bind_addr
// and http_bind_port) and combined internally as "ADDR:PORT" before writing
// the http_listen key in .env.  This makes the form easier for users who only
// want to change the port without touching the bind address.
//
// Once the .env is written the install server stops and normal bootstrap
// proceeds on the same port without a restart.
package install

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	luluHttp "github.com/GMWalletApp/epusdt/util/http"
	"github.com/gookit/color"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// DefaultInstallAddr is the listen address used by the install API.
// Matches the default http_listen so no extra port is needed.
const DefaultInstallAddr = ":8000"

// InstallRequest is the payload submitted by the install form.
// All fields are optional except AppURI; omitted fields fall back to InstallDefaults().
type InstallRequest struct {
	// Application display name (default: epusdt)
	AppName string `json:"app_name" form:"app_name" example:"epusdt"`
	// Public base URL of the service, e.g. https://pay.example.com (required)
	AppURI string `json:"app_uri" form:"app_uri" example:"https://pay.example.com"`
	// Bind address for the HTTP server (default: 127.0.0.1)
	HttpBindAddr string `json:"http_bind_addr" form:"http_bind_addr" example:"127.0.0.1"`
	// Bind port for the HTTP server (default: 8000)
	HttpBindPort int `json:"http_bind_port" form:"http_bind_port" example:"8000"`
	// Runtime directory for SQLite DB and temp files (default: ./runtime)
	RuntimeRootPath string `json:"runtime_root_path" form:"runtime_root_path" example:"./runtime"`
	// Directory for application log files (default: ./logs)
	LogSavePath string `json:"log_save_path" form:"log_save_path" example:"./logs"`
	// Minutes before an unpaid order expires (default: 10)
	OrderExpirationTime int `json:"order_expiration_time" form:"order_expiration_time" example:"10"`
	// Maximum webhook retry attempts (default: 1)
	OrderNoticeMaxRetry int `json:"order_notice_max_retry" form:"order_notice_max_retry" example:"1"`
}

// InstallDefaults returns sensible default values for the install form.
func InstallDefaults() InstallRequest {
	return InstallRequest{
		AppName:             "epusdt",
		AppURI:              "",
		HttpBindAddr:        "127.0.0.1",
		HttpBindPort:        8000,
		RuntimeRootPath:     "./runtime",
		LogSavePath:         "./logs",
		OrderExpirationTime: 10,
		OrderNoticeMaxRetry: 1,
	}
}

// installHandler holds the per-invocation state shared between handlers.
type installHandler struct {
	envFilePath string
	done        chan struct{}
}

// GetDefaults returns default values for the install form.
//
// @Summary      Install — get default values
// @Description  Returns sensible default field values for the first-run install form.
//
//	Available only before the .env config file has been written.
//	After installation completes this route is no longer served.
//
// @Tags         Install
// @Produce      json
// @Success      200 {object} InstallRequest "Default install values"
// @Router       /api/install/defaults [get]
func (h *installHandler) GetDefaults(c echo.Context) error {
	return c.JSON(http.StatusOK, InstallDefaults())
}

// Submit validates the install payload, writes the .env file, and signals
// the install server to shut down so the main bootstrap can proceed.
// http_bind_addr and http_bind_port are combined as "ADDR:PORT" to produce
// the http_listen config key (e.g. 0.0.0.0:8000).
//
// @Summary      Install — submit configuration
// @Description  Validates the submitted configuration and writes the .env file.
//
//	http_bind_addr + http_bind_port are joined internally as "ADDR:PORT" for
//	the http_listen config key (defaults: 127.0.0.1 and 8000 respectively).
//	http_bind_port must be in the range 1–65535 if provided.
//	app_uri is required. All other fields are optional and fall back to
//	the defaults returned by GET /api/install/defaults.
//	Sets install=false in the written .env, then shuts down the install
//	server so that normal application bootstrap starts on the same port.
//	After installation completes this route is no longer served.
//
// @Tags         Install
// @Accept       json
// @Produce      json
// @Param        body body     InstallRequest true "Install configuration"
// @Success      200  {object} map[string]string "message"
// @Failure      400  {object} map[string]string "error"
// @Failure      500  {object} map[string]string "error"
// @Router       /api/install [post]
func (h *installHandler) Submit(c echo.Context) error {
	req := new(InstallRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	req.AppURI = strings.TrimSpace(req.AppURI)
	if req.AppURI == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "app_uri is required"})
	}
	if req.HttpBindPort != 0 && (req.HttpBindPort < 1 || req.HttpBindPort > 65535) {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "http_bind_port must be between 1 and 65535"})
	}
	d := InstallDefaults()
	if strings.TrimSpace(req.AppName) == "" {
		req.AppName = d.AppName
	}
	if strings.TrimSpace(req.HttpBindAddr) == "" {
		req.HttpBindAddr = d.HttpBindAddr
	}
	if req.HttpBindPort <= 0 {
		req.HttpBindPort = d.HttpBindPort
	}
	if strings.TrimSpace(req.RuntimeRootPath) == "" {
		req.RuntimeRootPath = d.RuntimeRootPath
	}
	if strings.TrimSpace(req.LogSavePath) == "" {
		req.LogSavePath = d.LogSavePath
	}
	if req.OrderExpirationTime <= 0 {
		req.OrderExpirationTime = d.OrderExpirationTime
	}
	if req.OrderNoticeMaxRetry < 0 {
		req.OrderNoticeMaxRetry = d.OrderNoticeMaxRetry
	}

	if err := writeEnvFile(h.envFilePath, req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	go func() { close(h.done) }()
	return c.JSON(http.StatusOK, map[string]interface{}{"message": "install complete, starting server…"})
}

// RunInstallServer starts the install REST API on listenAddr (default :8000)
// under the /install path and blocks until the .env file has been written.
// The caller should then proceed with normal app initialisation (bootstrap.InitApp).
func RunInstallServer(listenAddr, envFilePath string) {
	if listenAddr == "" {
		listenAddr = DefaultInstallAddr
	}

	h := &installHandler{
		envFilePath: envFilePath,
		done:        make(chan struct{}),
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// api routes for the install frontend
	api := e.Group("/api")

	api.GET("/install/defaults", h.GetDefaults)
	api.POST("/install", h.Submit)

	// Resolve www/ relative to the executable so SPA routes work regardless
	// of the working directory. main.go extracts www/ next to the binary.
	wwwRoot := "./www"
	if exePath, err := os.Executable(); err == nil {
		if exePath, err = filepath.EvalSymlinks(exePath); err == nil {
			wwwRoot = filepath.Join(filepath.Dir(exePath), "www")
		}
	}
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper: func(c echo.Context) bool {
			return luluHttp.ShouldSkipSPAFallback(c.Request().URL.Path)
		},
		HTML5: true,
		Index: "index.html",
		Root:  wwwRoot,
	}))

	// Build a human-readable URL for the console hint.
	installHost := listenAddr
	if strings.HasPrefix(installHost, ":") {
		installHost = "localhost" + installHost
	}
	color.Green.Printf("[install] no config found — install API available at http://%s/install\n", installHost)

	go func() {
		<-h.done
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = e.Shutdown(ctx)
	}()

	if err := e.Start(listenAddr); err != nil && err != http.ErrServerClosed {
		color.Red.Printf("[install] server error: %s\n", err)
		os.Exit(1)
	}

	color.Green.Printf("[install] configuration saved to %s, starting…\n", envFilePath)
}

// formControlledKeys are keys whose values always come from the install form
// (or are set unconditionally by the template). Existing config values for
// these keys must NOT be preserved — the form submission takes precedence.
var formControlledKeys = map[string]bool{
	"app_name":               true,
	"app_uri":                true,
	"http_listen":            true,
	"runtime_root_path":      true,
	"log_save_path":          true,
	"order_expiration_time":  true,
	"order_notice_max_retry": true,
	"install":                true,
}

// writeEnvFile renders and writes a minimal .env file.
// If the file already exists, values for keys that are NOT controlled by the
// install form are preserved from the existing file so that operator-specific
// settings (tg_bot_token, db_type, etc.) survive a re-install.
// Keys that the form controls (app_uri, http_listen, …) always use the
// submitted values.
func writeEnvFile(path string, r *InstallRequest) error {
	// Collect existing non-empty key→value pairs for non-form keys.
	existingValues := map[string]string{}
	if data, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.IndexByte(line, '='); idx >= 0 {
				k := strings.TrimSpace(line[:idx])
				v := strings.TrimSpace(line[idx+1:])
				if v != "" && !formControlledKeys[k] {
					existingValues[k] = v
				}
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Render the template into a buffer first.
	var buf bytes.Buffer
	if err := envTemplate.Execute(&buf, r); err != nil {
		return fmt.Errorf("render env template: %w", err)
	}

	// For non-form keys that already had a value, substitute the existing value
	// so the template default does not clobber operator configuration.
	lines := strings.Split(buf.String(), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if idx := strings.IndexByte(trimmed, '='); idx >= 0 {
			k := strings.TrimSpace(trimmed[:idx])
			if existing, ok := existingValues[k]; ok {
				lines[i] = k + "=" + existing
			}
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprint(f, strings.Join(lines, "\n"))
	return err
}

var envTemplate = template.Must(template.New("env").Parse(`app_name={{.AppName}}
app_uri={{.AppURI}}
log_level=info
http_access_log=false
sql_debug=false
http_listen={{.HttpBindAddr}}:{{.HttpBindPort}}

static_path=/static
runtime_root_path={{.RuntimeRootPath}}

log_save_path={{.LogSavePath}}
log_max_size=32
log_max_age=7
max_backups=3

# supported values: postgres,mysql,sqlite
db_type=sqlite

# sqlite primary database config
sqlite_database_filename=
sqlite_table_prefix=

# sqlite runtime store config
runtime_sqlite_filename=epusdt-runtime.db

# background scheduler config
queue_concurrency=10
queue_poll_interval_ms=1000
callback_retry_base_seconds=5

order_expiration_time={{.OrderExpirationTime}}
order_notice_max_retry={{.OrderNoticeMaxRetry}}

api_rate_url=

# Set to true to re-run the install wizard on next startup.
install=false
`))
