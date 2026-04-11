package service

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"

	"github.com/assimon/luuu/config"
	tron "github.com/assimon/luuu/crypto"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/model/request"
	"github.com/assimon/luuu/telegram"
	"github.com/assimon/luuu/util/constant"
	"github.com/assimon/luuu/util/http_client"
	"github.com/assimon/luuu/util/log"
	"github.com/assimon/luuu/util/math"
	"github.com/dromara/carbon/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gookit/goutil/stdutil"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
)

const TRC20_USDT_ID = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

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

	resp, err := client.R().SetQueryParams(map[string]string{
		"order_by":      "block_timestamp,desc",
		"limit":         "100",
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

	success := gjson.GetBytes(resp.Body(), "success").Bool()
	if !success {
		panic("TRX API response indicates failure")
	}

	transfers := gjson.GetBytes(resp.Body(), "data").Array()
	if len(transfers) == 0 {
		log.Sugar.Debugf("[TRX][%s] no transfer records found", address)
		return
	}
	log.Sugar.Debugf("[TRX][%s] fetched %d transfer records", address, len(transfers))

	for i, transfer := range transfers {
		if transfer.Get("raw_data.contract.0.type").String() != "TransferContract" {
			continue
		}
		if transfer.Get("ret.0.contractRet").String() != "SUCCESS" {
			continue
		}

		toAddressHex := transfer.Get("raw_data.contract.0.parameter.value.to_address").String()
		toBytes, err := hex.DecodeString(toAddressHex)
		if err != nil {
			log.Sugar.Errorf("[TRX][%s] decode address failed on tx #%d: %v", address, i, err)
			continue
		}
		if tron.EncodeCheck(toBytes) != address {
			continue
		}

		rawAmount := transfer.Get("raw_data.contract.0.parameter.value.amount").String()
		decimalQuant, err := decimal.NewFromString(rawAmount)
		if err != nil {
			log.Sugar.Errorf("[TRX][%s] parse amount failed on tx #%d: %v", address, i, err)
			continue
		}
		amount := math.MustParsePrecFloat64(decimalQuant.Div(decimal.NewFromInt(1000000)).InexactFloat64(), 2)
		if amount <= 0 {
			continue
		}

		txID := transfer.Get("txID").String()
		tradeID, err := data.GetTradeIdByWalletAddressAndAmountAndToken(mdb.NetworkTron, address, "TRX", amount)
		if err != nil {
			panic(err)
		}
		if tradeID == "" {
			log.Sugar.Debugf("[TRX][%s] skip unmatched tx hash=%s amount=%.2f", address, txID, amount)
			continue
		}
		log.Sugar.Infof("[TRX][%s] matched trade_id=%s hash=%s amount=%.2f", address, tradeID, txID, amount)

		order, err := data.GetOrderInfoByTradeId(tradeID)
		if err != nil {
			panic(err)
		}
		blockTimestamp := transfer.Get("block_timestamp").Int()
		createTime := order.CreatedAt.TimestampMilli()
		if blockTimestamp < createTime {
			log.Sugar.Warnf("[TRX][%s] skip tx %s because block time %d is before order create time %d", address, txID, blockTimestamp, createTime)
			continue
		}

		req := &request.OrderProcessingRequest{
			ReceiveAddress:     address,
			Token:              "TRX",
			Network:            mdb.NetworkTron,
			TradeId:            tradeID,
			Amount:             amount,
			BlockTransactionId: txID,
		}
		err = OrderProcessing(req)
		if err != nil {
			if errors.Is(err, constant.OrderBlockAlreadyProcess) || errors.Is(err, constant.OrderStatusConflict) {
				log.Sugar.Infof("[TRX][%s] skip resolved transfer trade_id=%s hash=%s err=%v", address, tradeID, txID, err)
				continue
			}
			panic(err)
		}

		sendPaymentNotification(order)
		log.Sugar.Infof("[TRX][%s] payment processed trade_id=%s hash=%s", address, tradeID, txID)
	}
}

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

	resp, err := client.R().SetQueryParams(map[string]string{
		"order_by":      "block_timestamp,desc",
		"limit":         "100",
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

	success := gjson.GetBytes(resp.Body(), "success").Bool()
	if !success {
		panic("TRC20 API response indicates failure")
	}

	transfers := gjson.GetBytes(resp.Body(), "data").Array()
	if len(transfers) == 0 {
		log.Sugar.Debugf("[TRC20][%s] no transfer records found", address)
		return
	}
	log.Sugar.Debugf("[TRC20][%s] fetched %d transfer records", address, len(transfers))

	for i, transfer := range transfers {
		if transfer.Get("token_info.address").String() != TRC20_USDT_ID {
			continue
		}
		if transfer.Get("to").String() != address {
			continue
		}

		valueStr := transfer.Get("value").String()
		decimalQuant, err := decimal.NewFromString(valueStr)
		if err != nil {
			log.Sugar.Errorf("[TRC20][%s] parse value failed on tx #%d: %v", address, i, err)
			continue
		}
		tokenDecimals := transfer.Get("token_info.decimals").Int()
		amount := math.MustParsePrecFloat64(decimalQuant.Div(decimal.New(1, int32(tokenDecimals))).InexactFloat64(), 2)
		if amount <= 0 {
			continue
		}

		txID := transfer.Get("transaction_id").String()
		tradeID, err := data.GetTradeIdByWalletAddressAndAmountAndToken(mdb.NetworkTron, address, "USDT", amount)
		if err != nil {
			panic(err)
		}
		if tradeID == "" {
			log.Sugar.Debugf("[TRC20][%s] skip unmatched tx hash=%s amount=%.2f", address, txID, amount)
			continue
		}
		log.Sugar.Infof("[TRC20][%s] matched trade_id=%s hash=%s amount=%.2f", address, tradeID, txID, amount)

		order, err := data.GetOrderInfoByTradeId(tradeID)
		if err != nil {
			panic(err)
		}
		blockTimestamp := transfer.Get("block_timestamp").Int()
		createTime := order.CreatedAt.TimestampMilli()
		if blockTimestamp < createTime {
			log.Sugar.Warnf("[TRC20][%s] skip tx %s because block time %d is before order create time %d", address, txID, blockTimestamp, createTime)
			continue
		}

		req := &request.OrderProcessingRequest{
			ReceiveAddress:     address,
			Token:              "USDT",
			Network:            mdb.NetworkTron,
			TradeId:            tradeID,
			Amount:             amount,
			BlockTransactionId: txID,
		}
		err = OrderProcessing(req)
		if err != nil {
			if errors.Is(err, constant.OrderBlockAlreadyProcess) || errors.Is(err, constant.OrderStatusConflict) {
				log.Sugar.Infof("[TRC20][%s] skip resolved transfer trade_id=%s hash=%s err=%v", address, tradeID, txID, err)
				continue
			}
			panic(err)
		}

		sendPaymentNotification(order)
		log.Sugar.Infof("[TRC20][%s] payment processed trade_id=%s hash=%s", address, tradeID, txID)
	}
}

func TryProcessEthereumERC20Transfer(contract common.Address, toAddr common.Address, rawValue *big.Int, txHash string, blockTsMs int64) {
	defer func() {
		if err := recover(); err != nil {
			log.Sugar.Errorf("[ETH-WS] TryProcessEthereumERC20Transfer panic: %v", err)
		}
	}()

	usdt := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	usdc := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	var tokenSym string
	switch contract {
	case usdt:
		tokenSym = "USDT"
	case usdc:
		tokenSym = "USDC"
	default:
		log.Sugar.Warnf("[ETH-WS] skip unsupported contract %s", contract.Hex())
		return
	}

	walletAddr := strings.ToLower(toAddr.Hex())
	if rawValue == nil || rawValue.Sign() <= 0 {
		log.Sugar.Infof("[ETH-%s][%s] skip non-positive or nil amount", tokenSym, walletAddr)
		return
	}
	decimalQuant := decimal.NewFromBigInt(rawValue, 0)
	amount := math.MustParsePrecFloat64(decimalQuant.Div(decimal.NewFromInt(1_000_000)).InexactFloat64(), 2)
	if amount <= 0 {
		log.Sugar.Warnf("[ETH-%s][%s] skip non-positive amount %.2f", tokenSym, walletAddr, amount)
		return
	}

	tradeID, err := data.GetTradeIdByWalletAddressAndAmountAndToken(mdb.NetworkEthereum, walletAddr, tokenSym, amount)
	if err != nil {
		log.Sugar.Warnf("[ETH-%s][%s] lock lookup: %v", tokenSym, walletAddr, err)
		return
	}
	if tradeID == "" {
		log.Sugar.Warnf("[ETH-%s][%s] skip unmatched tx hash=%s amount=%.2f", tokenSym, walletAddr, txHash, amount)
		return
	}

	order, err := data.GetOrderInfoByTradeId(tradeID)
	if err != nil {
		log.Sugar.Warnf("[ETH-%s][%s] load order: %v", tokenSym, walletAddr, err)
		return
	}
	if strings.ToLower(strings.TrimSpace(order.Network)) != mdb.NetworkEthereum {
		log.Sugar.Warnf("[ETH-%s][%s] skip trade_id=%s network=%q", tokenSym, walletAddr, tradeID, order.Network)
		return
	}
	if strings.ToUpper(strings.TrimSpace(order.Token)) != tokenSym {
		log.Sugar.Warnf("[ETH-%s][%s] skip trade_id=%s token mismatch order=%s", tokenSym, walletAddr, tradeID, order.Token)
		return
	}

	req := &request.OrderProcessingRequest{
		ReceiveAddress:     walletAddr,
		Token:              tokenSym,
		Network:            mdb.NetworkEthereum,
		TradeId:            tradeID,
		Amount:             amount,
		BlockTransactionId: txHash,
	}
	err = OrderProcessing(req)
	if err != nil {
		if errors.Is(err, constant.OrderBlockAlreadyProcess) || errors.Is(err, constant.OrderStatusConflict) {
			log.Sugar.Infof("[ETH-%s][%s] skip resolved trade_id=%s hash=%s err=%v", tokenSym, walletAddr, tradeID, txHash, err)
			return
		}
		log.Sugar.Errorf("[ETH-%s][%s] OrderProcessing: %v", tokenSym, walletAddr, err)
		return
	}

	sendPaymentNotification(order)
	log.Sugar.Infof("[ETH-%s][%s] payment processed trade_id=%s hash=%s", tokenSym, walletAddr, tradeID, txHash)
}

func sendPaymentNotification(order *mdb.Orders) {
	msg := fmt.Sprintf(
		"🎉 <b>收款成功通知</b>\n\n"+
			"💰 <b>金额信息</b>\n"+
			"├ 订单金额：<code>%.2f %s</code>\n"+
			"└ 实际到账：<code>%.2f %s</code>\n\n"+
			"📋 <b>订单信息</b>\n"+
			"├ 交易号：<code>%s</code>\n"+
			"├ 订单号：<code>%s</code>\n"+
			"├ 网络：<code>%s</code>\n"+
			"└ 钱包地址：<code>%s</code>\n\n"+
			"⏰ <b>时间信息</b>\n"+
			"├ 创建时间：%s\n"+
			"└ 支付时间：%s",
		order.Amount,
		strings.ToUpper(order.Currency),
		order.ActualAmount,
		strings.ToUpper(order.Token),
		order.TradeId,
		order.OrderId,
		networkDisplay(order.Network),
		order.ReceiveAddress,
		order.CreatedAt.ToDateTimeString(),
		carbon.Now().ToDateTimeString(),
	)
	telegram.SendToBot(msg)
}

func networkDisplay(n string) string {
	switch strings.ToLower(strings.TrimSpace(n)) {
	case mdb.NetworkTron:
		return "Tron"
	case mdb.NetworkSolana:
		return "Solana"
	case mdb.NetworkEthereum:
		return "Ethereum"
	default:
		if n == "" {
			return "Tron"
		}
		return strings.ToUpper(n)
	}
}
