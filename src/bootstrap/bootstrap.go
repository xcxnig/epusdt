package bootstrap

import (
	"sync"

	"github.com/GMWalletApp/epusdt/config"
	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/mq"
	"github.com/GMWalletApp/epusdt/task"
	"github.com/GMWalletApp/epusdt/telegram"
	appjwt "github.com/GMWalletApp/epusdt/util/jwt"
	"github.com/GMWalletApp/epusdt/util/log"
	"github.com/gookit/color"
)

var initOnce sync.Once

func InitApp() {
	initOnce.Do(func() {
		config.Init()
		log.Init()
		dao.Init()
		// Wire settings-table lookups into the config package so
		// GetRateApiUrl / GetUsdtRate prefer DB-backed overrides.
		config.SettingsGetString = func(key string) string {
			return data.GetSettingString(key, "")
		}
		// Seed rate.api_url from .env into the settings table on first run
		// so the admin UI can display and change it without a code restart.
		// Only written if the key is not already present in the DB.
		if data.GetSettingString("rate.api_url", "") == "" {
			if envURL := config.GetRateApiUrlFromEnv(); envURL != "" {
				if err := data.SetSetting("rate", "rate.api_url", envURL, "string"); err != nil {
					color.Red.Printf("[bootstrap] seed rate.api_url err=%s\n", err)
				}
			}
		}
		// config.Init() computes RateApiUrl before SettingsGetString is
		// installed, so refresh the cache once DB-backed settings are available.
		config.RateApiUrl = config.GetRateApiUrl()
		// Seed admin account and JWT secret so the management console is
		// immediately usable on a fresh install. Both are idempotent.
		_, isNew, err := data.EnsureDefaultAdmin()
		if err != nil {
			color.Red.Printf("[bootstrap] ensure default admin err=%s\n", err)
		}
		if isNew {
			color.Yellow.Println("╔════════════════════════════════════════════════════════════════════════╗")
			color.Yellow.Println("║  Default admin account created. Fetch one-time password via API first!║")
			color.Yellow.Printf("║  Username: admin                                                       ║\n")
			color.Yellow.Println("║  GET /admin/api/v1/auth/init-password (one-time)                      ║")
			color.Yellow.Println("╚════════════════════════════════════════════════════════════════════════╝")
		}
		if _, err := appjwt.EnsureSecret(); err != nil {
			color.Red.Printf("[bootstrap] ensure jwt secret err=%s\n", err)
		}
		// Seed one universal default API key on fresh installs. The seeded
		// key (PID=1000) works for all three gateway flows.
		_, err = data.EnsureDefaultApiKey()
		if err != nil {
			color.Red.Printf("[bootstrap] ensure default api key err=%s\n", err)
		}
		mq.Start()
		go telegram.BotStart()
		go task.Start()
	})
}
