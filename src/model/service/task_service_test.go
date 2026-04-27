package service

import (
	"strings"
	"testing"
	"time"

	"github.com/GMWalletApp/epusdt/internal/testutil"
	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/GMWalletApp/epusdt/notify"
)

func TestSendPaymentNotificationUsesLatestOrderUpdatedAt(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	const channelType = "test-pay-success-time"
	got := make(chan string, 1)
	notify.RegisterSender(channelType, func(config, text string) error {
		got <- text
		return nil
	})

	if err := dao.Mdb.Create(&mdb.NotificationChannel{
		Type:    channelType,
		Name:    "test",
		Config:  "{}",
		Events:  `{"pay_success":true}`,
		Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed notification channel: %v", err)
	}

	order := &mdb.Orders{
		TradeId:        "T202604270001",
		OrderId:        "ORD202604270001",
		Amount:         100,
		Currency:       "cny",
		ActualAmount:   14.28,
		Token:          "USDT",
		Network:        mdb.NetworkTron,
		ReceiveAddress: "TTestAddress",
		Status:         mdb.StatusWaitPay,
	}
	if err := dao.Mdb.Create(order).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}

	const createdAt = "2026-04-27 09:00:00"
	const staleUpdatedAt = "2026-04-27 09:01:00"
	const paidAt = "2026-04-27 10:20:30"
	if err := dao.Mdb.Exec("UPDATE orders SET created_at = ?, updated_at = ? WHERE trade_id = ?", createdAt, staleUpdatedAt, order.TradeId).Error; err != nil {
		t.Fatalf("set initial timestamps: %v", err)
	}

	var staleOrderModel mdb.Orders
	if err := dao.Mdb.Where("trade_id = ?", order.TradeId).Take(&staleOrderModel).Error; err != nil {
		t.Fatalf("load stale order: %v", err)
	}

	if err := dao.Mdb.Exec("UPDATE orders SET status = ?, updated_at = ? WHERE trade_id = ?", mdb.StatusPaySuccess, paidAt, order.TradeId).Error; err != nil {
		t.Fatalf("set paid timestamp: %v", err)
	}

	sendPaymentNotification(&staleOrderModel)

	select {
	case text := <-got:
		if !strings.Contains(text, "支付时间："+paidAt) {
			t.Fatalf("notification payment time = %q, want %s", text, paidAt)
		}
		if strings.Contains(text, "支付时间："+staleUpdatedAt) {
			t.Fatalf("notification used stale payment time: %q", text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}
