package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
)

// ConnectTonLiteAPI builds a tonutils-go API client from a liteserver global
// config URL. Scanner and manual verification use this same connection setup.
func ConnectTonLiteAPI(ctx context.Context, configURL string, requestTimeout time.Duration, reconnectLimit int) (ton.APIClientWrapped, func(), error) {
	configURL = strings.TrimSpace(configURL)
	if configURL == "" {
		return nil, func() {}, fmt.Errorf("ton lite config url is empty")
	}
	if reconnectLimit <= 0 {
		reconnectLimit = 3
	}
	cfg, err := liteclient.GetConfigFromUrl(ctx, configURL)
	if err != nil {
		return nil, func() {}, err
	}
	pool := liteclient.NewConnectionPool()
	pool.SetOnDisconnect(pool.DefaultReconnect(3*time.Second, reconnectLimit))
	if err = pool.AddConnectionsFromConfig(ctx, cfg); err != nil {
		pool.Stop()
		return nil, func() {}, err
	}
	base := ton.NewAPIClient(pool, ton.ProofCheckPolicyFast)
	base.SetTrustedBlockFromConfig(cfg)
	api := base.WithRetry(2).WithTimeout(requestTimeout).WithLSInfoInErrors()
	return api, pool.Stop, nil
}
