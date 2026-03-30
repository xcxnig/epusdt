package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/assimon/luuu/util/http_client"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

var (
	AppDebug          bool
	MysqlDns          string
	RuntimePath       string
	LogSavePath       string
	StaticPath        string
	TgBotToken        string
	TgProxy           string
	TgManage          int64
	UsdtRate          float64
	RateApiUrl        string
	TRON_GRID_API_KEY string
)

func Init() {
	viper.AddConfigPath("./")
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	gwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	AppDebug = viper.GetBool("app_debug")
	StaticPath = viper.GetString("static_path")
	RuntimePath = fmt.Sprintf(
		"%s%s",
		gwd,
		viper.GetString("runtime_root_path"))
	LogSavePath = fmt.Sprintf(
		"%s%s",
		RuntimePath,
		viper.GetString("log_save_path"))
	MysqlDns = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		url.QueryEscape(viper.GetString("mysql_user")),
		url.QueryEscape(viper.GetString("mysql_passwd")),
		fmt.Sprintf(
			"%s:%s",
			viper.GetString("mysql_host"),
			viper.GetString("mysql_port")),
		viper.GetString("mysql_database"))
	TgBotToken = viper.GetString("tg_bot_token")
	TgProxy = viper.GetString("tg_proxy")
	TgManage = viper.GetInt64("tg_manage")

	GetRateApiUrl()
	TRON_GRID_API_KEY = viper.GetString("tron_grid_api_key")
}

func GetAppVersion() string {
	return "0.0.2"
}

func GetAppName() string {
	appName := viper.GetString("app_name")
	if appName == "" {
		return "epusdt"
	}
	return appName
}

func GetAppUri() string {
	return viper.GetString("app_uri")
}

func GetApiAuthToken() string {
	return viper.GetString("api_auth_token")
}

func GetRateApiUrl() string {
	url := viper.GetString("api_rate_url")
	urlFromEnv := os.Getenv("API_RATE_URL")
	if url == "" && urlFromEnv != "" {
		url = urlFromEnv
	}
	if url == "" {
		log.Println("api_rate_url is empty")
	}
	RateApiUrl = url
	return url
}

func GetRateForCoin(coin string, base string) float64 {
	client := http_client.GetHttpClient()
	baseUrl := RateApiUrl
	if baseUrl == "" {
		log.Printf("rate api url is empty")
		return 0.0
	}
	if baseUrl[len(baseUrl)-1] != '/' {
		baseUrl += "/"
	}
	url := baseUrl + fmt.Sprintf("%s.json", base)

	fmt.Println("call rate api url:", url)

	resp, err := client.R().Get(url)
	if err != nil {
		log.Printf("call rate api error: %s", err.Error())
		return 0.0
	}

	targetRate := 0.0

	gjson.GetBytes(resp.Body(), base).ForEach(func(key, value gjson.Result) bool {
		if key.String() == coin {
			targetRate = value.Float()
			return false
		}
		return true
	})
	return targetRate

}

func GetUsdtRate() float64 {
	forcedUsdtRate := viper.GetFloat64("forced_usdt_rate")
	if forcedUsdtRate > 0 {
		return forcedUsdtRate
	}
	if UsdtRate <= 0 {
		return 6.4
	}
	return UsdtRate
}

func GetOrderExpirationTime() int {
	timer := viper.GetInt("order_expiration_time")
	if timer <= 0 {
		return 10
	}
	return timer
}

func GetOrderExpirationTimeDuration() time.Duration {
	timer := GetOrderExpirationTime()
	return time.Minute * time.Duration(timer)
}
