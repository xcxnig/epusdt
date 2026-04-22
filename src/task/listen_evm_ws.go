package task

import (
	"context"
	"time"

	"github.com/GMWalletApp/epusdt/util/log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// runEvmWsLogListener connects to wsURL, subscribes to Transfer logs,
// and dispatches each log to handleLog. It retries on transient errors
// with exponential backoff. The ctx lets the caller trigger a clean
// exit — e.g. when admin disables the chain, the caller cancels the
// context and the function returns instead of reconnecting forever.
func runEvmWsLogListener(ctx context.Context, logPrefix, wsURL string, query ethereum.FilterQuery, handleLog func(*ethclient.Client, types.Log)) {
	const (
		minBackoff = 2 * time.Second
		maxBackoff = 60 * time.Second
		rejoinWait = 3 * time.Second
	)
	failWait := minBackoff

	for {
		if ctx.Err() != nil {
			return
		}

		client, err := ethclient.Dial(wsURL)
		if err != nil {
			log.Sugar.Warnf("%s dial: %v, retry in %s", logPrefix, err, failWait)
			if !sleepOrDone(ctx, failWait) {
				return
			}
			failWait = nextBackoff(failWait, maxBackoff)
			continue
		}

		logsCh := make(chan types.Log)
		sub, err := client.SubscribeFilterLogs(ctx, query, logsCh)
		if err != nil {
			client.Close()
			log.Sugar.Warnf("%s subscribe: %v, retry in %s", logPrefix, err, failWait)
			if !sleepOrDone(ctx, failWait) {
				return
			}
			failWait = nextBackoff(failWait, maxBackoff)
			continue
		}
		failWait = minBackoff

		log.Sugar.Infof("%s connected, subscribed to Transfer logs", logPrefix)

		recvLoop(ctx, client, sub, logsCh, logPrefix, handleLog)

		if ctx.Err() != nil {
			return
		}
		if !sleepOrDone(ctx, rejoinWait) {
			return
		}
	}
}

func recvLoop(ctx context.Context, client *ethclient.Client, sub ethereum.Subscription, logsCh <-chan types.Log, logPrefix string, handleLog func(*ethclient.Client, types.Log)) {
	defer func() {
		sub.Unsubscribe()
		client.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Sugar.Infof("%s context cancelled, stopping", logPrefix)
			return
		case err := <-sub.Err():
			if err != nil {
				log.Sugar.Warnf("%s subscription error: %v, reconnecting", logPrefix, err)
			} else {
				log.Sugar.Warnf("%s subscription closed, reconnecting", logPrefix)
			}
			return
		case vLog, ok := <-logsCh:
			if !ok {
				log.Sugar.Warnf("%s log channel closed, reconnecting", logPrefix)
				return
			}
			handleLog(client, vLog)
		}
	}
}

// sleepOrDone waits for d or for ctx cancellation, whichever comes
// first. Returns true if the sleep completed normally, false if ctx
// was cancelled (caller should exit).
func sleepOrDone(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func nextBackoff(cur, max time.Duration) time.Duration {
	n := cur * 2
	if n > max {
		return max
	}
	return n
}
