package telegram

import (
	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/util/log"
	tb "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	"time"
)

var bots *tb.Bot

// BotStart 机器人启动
func BotStart() {
	var err error
	botSetting := tb.Settings{
		Token:       config.TgBotToken,
		Poller:      &tb.LongPoller{Timeout: 50 * time.Second},
		Synchronous: true,
	}
	if config.TgProxy != "" {
		botSetting.URL = config.TgProxy
	}
	bots, err = tb.NewBot(botSetting)
	if err != nil {
		log.Sugar.Error(err.Error())
		return
	}
	err = bots.SetCommands(Cmds)
	if err != nil {
		log.Sugar.Error(err.Error())
		return
	}
	RegisterHandle()
	bots.Start()
}

// RegisterHandle 注册处理器
func RegisterHandle() {
	adminOnly := bots.Group()
	adminOnly.Use(middleware.Whitelist(config.TgManage))
	adminOnly.Handle(START_CMD, WalletList)
	adminOnly.Handle(tb.OnText, OnTextMessageHandle)
}

// SendToBot 主动发送消息机器人消息
func SendToBot(msg string) {
	go func() {
		if bots == nil {
			log.Sugar.Error("[Telegram] bots实例为nil，无法发送消息")
			return
		}
		user := tb.User{
			ID: config.TgManage,
		}
		log.Sugar.Infof("[Telegram] 正在发送消息给用户ID=%d", config.TgManage)
		_, err := bots.Send(&user, msg, &tb.SendOptions{
			ParseMode: tb.ModeHTML,
		})
		if err != nil {
			log.Sugar.Errorf("[Telegram] 发送消息失败: userID=%d, err=%v", config.TgManage, err)
		} else {
			log.Sugar.Infof("[Telegram] 发送消息成功: userID=%d", config.TgManage)
		}
	}()
}
