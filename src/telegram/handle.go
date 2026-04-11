package telegram

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
	tb "gopkg.in/telebot.v3"
)

const (
	ReplayAddWallet         = "请发给我 TRON（T 开头）或以太坊主网（0x 开头）收款地址"
	pendingWalletAddressTTL = 5 * time.Minute
)

type pendingWalletAddressState struct {
	RequestedAt time.Time
}

var pendingWalletAddressUsers sync.Map

func OnTextMessageHandle(c tb.Context) error {
	msg := c.Message()
	if msg == nil {
		return nil
	}

	sender := c.Sender()
	senderID := int64(0)
	if sender != nil {
		senderID = sender.ID
	}

	isReplyFlow := msg.ReplyTo != nil && msg.ReplyTo.Text == ReplayAddWallet
	isPendingFlow := isWalletAddressPending(senderID)
	if !isReplyFlow && !isPendingFlow {
		return nil
	}

	if isReplyFlow {
		defer bots.Delete(msg.ReplyTo)
	}

	msgText := strings.TrimSpace(msg.Text)
	var err error
	switch {
	case isValidTronAddress(msgText):
		_, err = data.AddWalletAddress(msgText)
	case isValidEthereumAddress(msgText):
		_, err = data.AddWalletAddressWithNetwork(mdb.NetworkEthereum, strings.ToLower(msgText))
	default:
		_ = c.Send(fmt.Sprintf("钱包 [%s] 添加失败：不是合法的 TRON 或以太坊地址", msgText))
		return nil
	}
	if err != nil {
		return c.Send(err.Error())
	}
	pendingWalletAddressUsers.Delete(senderID)

	_ = c.Send(fmt.Sprintf("钱包 [%s] 添加成功", msgText))
	return WalletList(c)
}

func WalletList(c tb.Context) error {
	wallets, err := data.GetAllWalletAddress()
	if err != nil {
		return err
	}

	var btnList [][]tb.InlineButton
	for _, wallet := range wallets {
		status := "已启用✅"
		if wallet.Status == mdb.TokenStatusDisable {
			status = "已禁用🚫"
		}
		net := wallet.Network
		if net == "" {
			net = mdb.NetworkTron
		}

		btnInfo := tb.InlineButton{
			Unique: wallet.Address,
			Text:   fmt.Sprintf("[%s] %s [%s]", net, wallet.Address, status),
			Data:   strutil.MustString(wallet.ID),
		}
		bots.Handle(&btnInfo, WalletInfo)
		btnList = append(btnList, []tb.InlineButton{btnInfo})
	}

	addBtn := tb.InlineButton{Text: "添加钱包地址", Unique: "AddWallet"}
	bots.Handle(&addBtn, func(c tb.Context) error {
		if sender := c.Sender(); sender != nil {
			pendingWalletAddressUsers.Store(sender.ID, pendingWalletAddressState{RequestedAt: time.Now()})
		}
		return c.Send(ReplayAddWallet, &tb.ReplyMarkup{
			ForceReply: true,
		})
	})
	btnList = append(btnList, []tb.InlineButton{addBtn})

	return c.EditOrSend("请选择钱包继续操作", &tb.ReplyMarkup{
		InlineKeyboard: btnList,
	})
}

func WalletInfo(c tb.Context) error {
	id := mathutil.MustUint(c.Data())
	tokenInfo, err := data.GetWalletAddressById(id)
	if err != nil {
		return c.Send(err.Error())
	}

	enableBtn := tb.InlineButton{
		Text:   "启用",
		Unique: "enableBtn",
		Data:   c.Data(),
	}
	disableBtn := tb.InlineButton{
		Text:   "禁用",
		Unique: "disableBtn",
		Data:   c.Data(),
	}
	delBtn := tb.InlineButton{
		Text:   "删除",
		Unique: "delBtn",
		Data:   c.Data(),
	}
	backBtn := tb.InlineButton{
		Text:   "返回",
		Unique: "WalletList",
	}

	bots.Handle(&enableBtn, EnableWallet)
	bots.Handle(&disableBtn, DisableWallet)
	bots.Handle(&delBtn, DelWallet)
	bots.Handle(&backBtn, WalletList)

	net := tokenInfo.Network
	if net == "" {
		net = mdb.NetworkTron
	}
	detail := fmt.Sprintf("网络：%s\n地址：%s", net, tokenInfo.Address)
	return c.EditOrReply(detail, &tb.ReplyMarkup{InlineKeyboard: [][]tb.InlineButton{
		{
			enableBtn,
			disableBtn,
			delBtn,
		},
		{
			backBtn,
		},
	}})
}

func EnableWallet(c tb.Context) error {
	id := mathutil.MustUint(c.Data())
	if id <= 0 {
		return c.Send("请求不合法！")
	}
	err := data.ChangeWalletAddressStatus(id, mdb.TokenStatusEnable)
	if err != nil {
		return c.Send(err.Error())
	}
	return WalletList(c)
}

func DisableWallet(c tb.Context) error {
	id := mathutil.MustUint(c.Data())
	if id <= 0 {
		return c.Send("请求不合法！")
	}
	err := data.ChangeWalletAddressStatus(id, mdb.TokenStatusDisable)
	if err != nil {
		return c.Send(err.Error())
	}
	return WalletList(c)
}

func DelWallet(c tb.Context) error {
	id := mathutil.MustUint(c.Data())
	if id <= 0 {
		return c.Send("请求不合法！")
	}
	err := data.DeleteWalletAddressById(id)
	if err != nil {
		return c.Send(err.Error())
	}
	return WalletList(c)
}

func isWalletAddressPending(userID int64) bool {
	if userID <= 0 {
		return false
	}
	value, ok := pendingWalletAddressUsers.Load(userID)
	if !ok {
		return false
	}

	state, ok := value.(pendingWalletAddressState)
	if !ok || time.Since(state.RequestedAt) > pendingWalletAddressTTL {
		pendingWalletAddressUsers.Delete(userID)
		return false
	}
	return true
}
