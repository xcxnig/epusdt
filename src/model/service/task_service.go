package service

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"

	tron "github.com/assimon/luuu/crypto"

	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/request"
	"github.com/assimon/luuu/mq"
	"github.com/assimon/luuu/mq/handle"
	"github.com/assimon/luuu/telegram"
	"github.com/assimon/luuu/util/http_client"
	"github.com/assimon/luuu/util/log"
	"github.com/assimon/luuu/util/math"
	"github.com/dromara/carbon/v2"
	"github.com/gookit/goutil/stdutil"
)

const TRC20_USDT_ID = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

// Trc20CallBack trc20回调
func Trc20CallBack(address string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			log.Sugar.Error(err)
		}
	}()

	var innerWg sync.WaitGroup
	innerWg.Add(2)
	go checkTrxTransfers(address, &innerWg)
	go checkTrc20Transfers(address, &innerWg)
	innerWg.Wait()
}

// checkTrxTransfers 查询 TRX 原生转账
func checkTrxTransfers(address string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			log.Sugar.Errorf("[TRX][%s] panic recovered: %v", address, err)
		}
	}()

	client := http_client.GetHttpClient()
	startTime := carbon.Now().AddHours(-24).TimestampMilli()
	endTime := carbon.Now().TimestampMilli()

	url := fmt.Sprintf("https://api.trongrid.io/v1/accounts/%s/transactions", address)
	log.Sugar.Debugf("checkTrxTransfers URL: %s, from %d to %d", url, startTime, endTime)

	resp, err := client.R().SetQueryParams(map[string]string{
		"order_by": "block_timestamp,desc",
		"limit":    "100",
		// "only_confirmed": "true",
		"only_to":       "true",
		"min_timestamp": stdutil.ToString(startTime),
		"max_timestamp": stdutil.ToString(endTime),
	}).SetHeader("TRON-PRO-API-KEY", config.TRON_GRID_API_KEY).Get(url)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != http.StatusOK {
		panic(fmt.Sprintf("TRX API returned status %d", resp.StatusCode()))
	}

	log.Sugar.Debugf("Raw request URL: %s", resp.Request.URL)

	success := gjson.GetBytes(resp.Body(), "success").Bool()
	if !success {
		panic("TRX API response indicates failure")
	}
	dataArray := gjson.GetBytes(resp.Body(), "data").Array()
	log.Sugar.Infof("[TRX][%s] API返回 %d 条交易记录", address, len(dataArray))
	if len(dataArray) == 0 {
		log.Sugar.Infof("[TRX][%s] 没有找到任何交易记录，跳过", address)
		return
	}

	for i, transfer := range dataArray {
		transferType := transfer.Get("raw_data.contract.0.type").String()
		if transferType != "TransferContract" {
			log.Sugar.Debugf("[TRX][%s] 第%d条: 类型=%s, 非TransferContract, 跳过", address, i, transferType)
			continue
		}

		contractRet := transfer.Get("ret.0.contractRet").String()
		if contractRet != "SUCCESS" {
			log.Sugar.Infof("[TRX][%s] 第%d条: contractRet=%s, 非SUCCESS, 跳过", address, i, contractRet)
			continue
		}

		toAddressHex := transfer.Get("raw_data.contract.0.parameter.value.to_address").String()
		toBytes, err := hex.DecodeString(toAddressHex)
		if err != nil {
			log.Sugar.Errorf("[TRX][%s] 第%d条: 解码地址失败: %v", address, i, err)
			continue
		}
		toAddress := tron.EncodeCheck(toBytes)
		if toAddress != address {
			log.Sugar.Debugf("[TRX][%s] 第%d条: 目标地址=%s, 不匹配, 跳过", address, i, toAddress)
			continue
		}

		rawAmount := transfer.Get("raw_data.contract.0.parameter.value.amount").String()
		decimalQuant, err := decimal.NewFromString(rawAmount)
		if err != nil {
			log.Sugar.Errorf("[TRX][%s] 第%d条: 解析金额失败: %v", address, i, err)
			continue
		}
		divisor := decimal.NewFromInt(1000000)
		amount := math.MustParsePrecFloat64(decimalQuant.Div(divisor).InexactFloat64(), 2)
		txID := transfer.Get("txID").String()
		log.Sugar.Infof("[TRX][%s] 第%d条: txID=%s, rawAmount=%s, 解析金额=%.2f", address, i, txID, rawAmount, amount)
		if amount <= 0 {
			log.Sugar.Infof("[TRX][%s] 第%d条: 金额<=0, 跳过", address, i)
			continue
		}

		cacheKey := fmt.Sprintf("wallet:%s_%s_%v", address, "TRX", amount)
		log.Sugar.Infof("[TRX][%s] 第%d条: 查询Redis匹配, cacheKey=%s", address, i, cacheKey)
		tradeId, err := data.GetTradeIdByWalletAddressAndAmountAndToken(address, "TRX", amount)
		if err != nil {
			panic(err)
		}
		if tradeId == "" {
			log.Sugar.Infof("[TRX][%s] 第%d条: Redis未匹配到订单, 金额=%.2f, 跳过", address, i, amount)
			continue
		}
		log.Sugar.Infof("[TRX][%s] 第%d条: Redis匹配到订单! tradeId=%s, 金额=%.2f", address, i, tradeId, amount)
		order, err := data.GetOrderInfoByTradeId(tradeId)
		if err != nil {
			panic(err)
		}
		log.Sugar.Infof("[TRX][%s] 查到订单: tradeId=%s, orderId=%s, status=%d, amount=%.2f, actualAmount=%.2f", address, order.TradeId, order.OrderId, order.Status, order.Amount, order.ActualAmount)

		createTime := order.CreatedAt.TimestampMilli()
		blockTimestamp := transfer.Get("block_timestamp").Int()
		log.Sugar.Infof("[TRX][%s] 时间校验: blockTimestamp=%d, orderCreateTime=%d", address, blockTimestamp, createTime)
		if blockTimestamp < createTime {
			log.Sugar.Errorf("[TRX][%s] 区块时间早于订单创建时间，无法匹配! blockTimestamp=%d < createTime=%d", address, blockTimestamp, createTime)
			panic("Orders cannot actually be matched")
		}

		transferHash := transfer.Get("txID").String()
		log.Sugar.Infof("[TRX][%s] 开始处理订单: tradeId=%s, hash=%s, amount=%.2f", address, tradeId, transferHash, amount)

		req := &request.OrderProcessingRequest{
			ReceiveAddress:     address,
			Token:              "TRX",
			TradeId:            tradeId,
			Amount:             amount,
			BlockTransactionId: transferHash,
		}
		err = OrderProcessing(req)
		if err != nil {
			log.Sugar.Errorf("[TRX][%s] OrderProcessing失败: tradeId=%s, err=%v", address, tradeId, err)
			panic(err)
		}
		log.Sugar.Infof("[TRX][%s] OrderProcessing成功: tradeId=%s", address, tradeId)

		orderCallbackQueue, _ := handle.NewOrderCallbackQueue(order)
		orderNoticeMaxRetry := viper.GetInt("order_notice_max_retry")
		mq.MClient.Enqueue(orderCallbackQueue, asynq.MaxRetry(orderNoticeMaxRetry),
			asynq.Retention(config.GetOrderExpirationTimeDuration()),
		)
		log.Sugar.Infof("[TRX][%s] 回调队列已入队: tradeId=%s", address, tradeId)

		msgTpl := `
🎉 <b>收款成功通知</b>

💰 <b>金额信息</b>
├ 订单金额：<code>%.2f %s</code>
└ 实际到账：<code>%.2f %s</code>

📋 <b>订单信息</b>
├ 交易号：<code>%s</code>
├ 订单号：<code>%s</code>
└ 钱包地址：<code>%s</code>

⏰ <b>时间信息</b>
├ 创建时间：%s
└ 支付时间：%s
`
		msg := fmt.Sprintf(msgTpl, order.Amount, strings.ToUpper(order.Currency), order.ActualAmount, strings.ToUpper(order.Token), order.TradeId, order.OrderId, order.ReceiveAddress, order.CreatedAt.ToDateTimeString(), carbon.Now().ToDateTimeString())
		log.Sugar.Infof("[TRX][%s] 准备发送Telegram通知: tradeId=%s, orderId=%s", address, tradeId, order.OrderId)
		telegram.SendToBot(msg)
	}
}

// checkTrc20Transfers 查询 TRC20 (USDT) 转账
func checkTrc20Transfers(address string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			log.Sugar.Errorf("[TRC20][%s] panic recovered: %v", address, err)
		}
	}()

	client := http_client.GetHttpClient()
	startTime := carbon.Now().AddHours(-24).TimestampMilli()
	endTime := carbon.Now().TimestampMilli()

	url := fmt.Sprintf("https://api.trongrid.io/v1/accounts/%s/transactions/trc20", address)
	log.Sugar.Debugf("checkTrc20Transfers URL: %s, from %d to %d", url, startTime, endTime)

	resp, err := client.R().SetQueryParams(map[string]string{
		"order_by": "block_timestamp,desc",
		"limit":    "100",
		// "only_confirmed": "true",
		"only_to":       "true",
		"min_timestamp": stdutil.ToString(startTime),
		"max_timestamp": stdutil.ToString(endTime),
	}).SetHeader("TRON-PRO-API-KEY", config.TRON_GRID_API_KEY).Get(url)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != http.StatusOK {
		panic(fmt.Sprintf("TRC20 API returned status %d", resp.StatusCode()))
	}

	log.Sugar.Debugf("Raw request URL: %s", resp.Request.URL)

	success := gjson.GetBytes(resp.Body(), "success").Bool()
	if !success {
		panic("TRC20 API response indicates failure")
	}
	dataArray := gjson.GetBytes(resp.Body(), "data").Array()
	log.Sugar.Infof("[TRC20][%s] API返回 %d 条交易记录", address, len(dataArray))
	if len(dataArray) == 0 {
		log.Sugar.Infof("[TRC20][%s] 没有找到任何交易记录，跳过", address)
		return
	}

	for i, transfer := range dataArray {
		// 只处理 USDT
		tokenAddress := transfer.Get("token_info.address").String()
		if tokenAddress != TRC20_USDT_ID {
			log.Sugar.Debugf("[TRC20][%s] 第%d条: tokenAddress=%s, 非USDT, 跳过", address, i, tokenAddress)
			continue
		}

		to := transfer.Get("to").String()
		if to != address {
			log.Sugar.Debugf("[TRC20][%s] 第%d条: to=%s, 不匹配, 跳过", address, i, to)
			continue
		}

		// 解析金额: value / 10^decimals
		valueStr := transfer.Get("value").String()
		decimalQuant, err := decimal.NewFromString(valueStr)
		if err != nil {
			log.Sugar.Errorf("[TRC20][%s] 第%d条: 解析value失败: %v", address, i, err)
			continue
		}
		tokenDecimals := transfer.Get("token_info.decimals").Int()
		divisor := decimal.New(1, int32(tokenDecimals)) // 10^decimals
		amount := math.MustParsePrecFloat64(decimalQuant.Div(divisor).InexactFloat64(), 2)
		txID := transfer.Get("transaction_id").String()
		log.Sugar.Infof("[TRC20][%s] 第%d条: txID=%s, value=%s, decimals=%d, 解析金额=%.2f", address, i, txID, valueStr, tokenDecimals, amount)
		if amount <= 0 {
			log.Sugar.Infof("[TRC20][%s] 第%d条: 金额<=0, 跳过", address, i)
			continue
		}

		cacheKey := fmt.Sprintf("wallet:%s_%s_%v", address, "USDT", amount)
		log.Sugar.Infof("[TRC20][%s] 第%d条: 查询Redis匹配, cacheKey=%s", address, i, cacheKey)
		tradeId, err := data.GetTradeIdByWalletAddressAndAmountAndToken(address, "USDT", amount)
		if err != nil {
			panic(err)
		}
		if tradeId == "" {
			log.Sugar.Infof("[TRC20][%s] 第%d条: Redis未匹配到订单, 金额=%.2f, 跳过", address, i, amount)
			continue
		}
		log.Sugar.Infof("[TRC20][%s] 第%d条: Redis匹配到订单! tradeId=%s, 金额=%.2f", address, i, tradeId, amount)
		order, err := data.GetOrderInfoByTradeId(tradeId)
		if err != nil {
			panic(err)
		}
		log.Sugar.Infof("[TRC20][%s] 查到订单: tradeId=%s, orderId=%s, status=%d, amount=%.2f, actualAmount=%.2f", address, order.TradeId, order.OrderId, order.Status, order.Amount, order.ActualAmount)

		createTime := order.CreatedAt.TimestampMilli()
		blockTimestamp := transfer.Get("block_timestamp").Int()
		log.Sugar.Infof("[TRC20][%s] 时间校验: blockTimestamp=%d, orderCreateTime=%d", address, blockTimestamp, createTime)
		if blockTimestamp < createTime {
			log.Sugar.Errorf("[TRC20][%s] 区块时间早于订单创建时间，无法匹配! blockTimestamp=%d < createTime=%d", address, blockTimestamp, createTime)
			panic("Orders cannot actually be matched")
		}

		transferHash := transfer.Get("transaction_id").String()
		log.Sugar.Infof("[TRC20][%s] 开始处理订单: tradeId=%s, hash=%s, amount=%.2f", address, tradeId, transferHash, amount)

		req := &request.OrderProcessingRequest{
			ReceiveAddress:     address,
			Token:              "USDT",
			TradeId:            tradeId,
			Amount:             amount,
			BlockTransactionId: transferHash,
		}
		err = OrderProcessing(req)
		if err != nil {
			log.Sugar.Errorf("[TRC20][%s] OrderProcessing失败: tradeId=%s, err=%v", address, tradeId, err)
			panic(err)
		}
		log.Sugar.Infof("[TRC20][%s] OrderProcessing成功: tradeId=%s", address, tradeId)

		orderCallbackQueue, _ := handle.NewOrderCallbackQueue(order)
		orderNoticeMaxRetry := viper.GetInt("order_notice_max_retry")
		mq.MClient.Enqueue(orderCallbackQueue, asynq.MaxRetry(orderNoticeMaxRetry),
			asynq.Retention(config.GetOrderExpirationTimeDuration()),
		)
		log.Sugar.Infof("[TRC20][%s] 回调队列已入队: tradeId=%s", address, tradeId)

		msgTpl := `
🎉 <b>收款成功通知</b>

💰 <b>金额信息</b>
├ 订单金额：<code>%.2f %s</code>
└ 实际到账：<code>%.2f %s</code>

📋 <b>订单信息</b>
├ 交易号：<code>%s</code>
├ 订单号：<code>%s</code>
└ 钱包地址：<code>%s</code>

⏰ <b>时间信息</b>
├ 创建时间：%s
└ 支付时间：%s
`
		msg := fmt.Sprintf(msgTpl, order.Amount, strings.ToUpper(order.Currency), order.ActualAmount, strings.ToUpper(order.Token), order.TradeId, order.OrderId, order.ReceiveAddress, order.CreatedAt.ToDateTimeString(), carbon.Now().ToDateTimeString())
		log.Sugar.Infof("[TRC20][%s] 准备发送Telegram通知: tradeId=%s, orderId=%s", address, tradeId, order.OrderId)
		telegram.SendToBot(msg)
	}
}
