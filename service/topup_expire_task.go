package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
)

const (
	topUpExpireTickInterval = 5 * time.Minute // 每5分钟检查一次
	topUpExpireAfterSeconds = 30 * 60         // 30分钟未支付视为过期
	topUpExpireBatchSize    = 500             // 每批处理500条
)

var (
	topUpExpireOnce    sync.Once
	topUpExpireRunning atomic.Bool
)

// StartTopUpExpireTask 启动支付订单自动过期检查任务
func StartTopUpExpireTask() {
	topUpExpireOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("topup expire task started: tick=%s, expire_after=%ds", topUpExpireTickInterval, topUpExpireAfterSeconds))
			ticker := time.NewTicker(topUpExpireTickInterval)
			defer ticker.Stop()

			runTopUpExpireOnce()
			for range ticker.C {
				runTopUpExpireOnce()
			}
		})
	})
}

func runTopUpExpireOnce() {
	if !topUpExpireRunning.CompareAndSwap(false, true) {
		return
	}
	defer topUpExpireRunning.Store(false)

	ctx := context.Background()
	totalExpired := int64(0)
	for {
		n, err := model.ExpirePendingTopUps(topUpExpireAfterSeconds, topUpExpireBatchSize)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("topup expire task failed: %v", err))
			return
		}
		if n == 0 {
			break
		}
		totalExpired += n
		if n < int64(topUpExpireBatchSize) {
			break
		}
	}
	if common.DebugEnabled && totalExpired > 0 {
		logger.LogDebug(ctx, "topup expire task: expired_count=%d", totalExpired)
	}
}
