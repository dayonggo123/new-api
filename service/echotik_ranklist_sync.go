package service

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

var echotikRanklistSyncRunning atomic.Bool

// StartEchotikRanklistSyncTask 启动 EchoTik 榜单定时预热同步任务（仅主节点执行）。
func StartEchotikRanklistSyncTask() {
	if os.Getenv("DISABLE_ECHOTIK_SYNC") == "true" {
		common.SysLog("Echotik ranklist sync task disabled by DISABLE_ECHOTIK_SYNC")
		return
	}
	if !common.IsMasterNode {
		common.SysLog("Echotik ranklist sync task skipped on non-master node")
		return
	}

	setting := operation_setting.GetEchotikSetting()
	if !setting.EchotikEnabled {
		common.SysLog("Echotik ranklist sync task skipped: EchoTik is not enabled")
		return
	}
	if !setting.EchotikCacheEnabled {
		common.SysLog("Echotik ranklist sync task skipped: cache is not enabled")
		return
	}
	if !setting.EchotikSyncEnabled {
		common.SysLog("Echotik ranklist sync task skipped: sync is not enabled")
		return
	}

	gopool.Go(func() {
		syncEchotikRanklistTaskLoop()
	})
}

func syncEchotikRanklistTaskLoop() {
	setting := operation_setting.GetEchotikSetting()
	frequency := setting.EchotikSyncFrequencyHours
	if frequency <= 0 {
		frequency = 24
	}
	interval := time.Duration(frequency) * time.Hour

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	common.SysLog(fmt.Sprintf("Echotik ranklist sync task started, interval=%s", interval))

	for {
		runEchotikRanklistSyncOnce()
		<-ticker.C
	}
}

// runEchotikRanklistSyncOnce 执行一轮预热同步，同一时间只允许一个实例运行。
func runEchotikRanklistSyncOnce() {
	if !echotikRanklistSyncRunning.CompareAndSwap(false, true) {
		common.SysLog("Echotik ranklist sync skipped: previous round still running")
		return
	}
	defer echotikRanklistSyncRunning.Store(false)

	setting := operation_setting.GetEchotikSetting()
	paramsList := buildEchotikSyncParamMatrix()
	if len(paramsList) == 0 {
		common.SysLog("Echotik ranklist sync: no params to sync")
		return
	}

	qps := setting.EchotikSyncQPS
	if qps <= 0 {
		qps = 1
	}
	rateInterval := time.Duration(float64(time.Second) / float64(qps))

	ctx := context.Background()
	successCount := 0
	skipCount := 0
	failCount := 0

	for _, params := range paramsList {
		// 跳过已存在未过期缓存的组合。
		key := ranklistParamsToKey(&params)
		fresh, err := model.GetFreshEchotikRanklistSnapshot(key)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("Echotik sync failed to check fresh snapshot for %v: %s", params, err.Error()))
		}
		if fresh != nil {
			skipCount++
			time.Sleep(rateInterval)
			continue
		}

		_, err = FetchAndSave(ctx, &params)
		if err != nil {
			failCount++
			logger.LogError(ctx, fmt.Sprintf("Echotik sync failed for %v: %s", params, err.Error()))
		} else {
			successCount++
		}

		time.Sleep(rateInterval)
	}

	common.SysLog(fmt.Sprintf("Echotik ranklist sync round finished: total=%d, success=%d, skipped=%d, failed=%d",
		len(paramsList), successCount, skipCount, failCount))

	// 每轮同步后触发一次清理。
	StartEchotikRanklistCleanupOnce()
}

// buildEchotikSyncParamMatrix 根据配置构建预同步参数矩阵。
func buildEchotikSyncParamMatrix() []dto.EchotikRanklistParams {
	setting := operation_setting.GetEchotikSetting()

	regions := setting.EchotikSyncRegions
	if len(regions) == 0 {
		regions = []string{"US"}
	}
	rankFields := setting.EchotikSyncRankFields
	if len(rankFields) == 0 {
		rankFields = []int{1, 2}
	}
	rankTypes := setting.EchotikSyncRankTypes
	if len(rankTypes) == 0 {
		rankTypes = []int{1, 2, 3}
	}
	categoryIDs := setting.EchotikSyncProductCategoryIDs
	if len(categoryIDs) == 0 {
		categoryIDs = []string{""}
	}
	createdByAIOptions := setting.EchotikSyncCreatedByAIOptions
	if len(createdByAIOptions) == 0 {
		createdByAIOptions = []string{""}
	}

	maxPages := setting.EchotikSyncMaxPages
	if maxPages <= 0 {
		maxPages = 1
	}
	pageSize := setting.EchotikSyncPageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	dateDays := setting.EchotikSyncDateDays
	if dateDays <= 0 {
		dateDays = 3
	}

	dates := buildRecentDates(dateDays)

	var paramsList []dto.EchotikRanklistParams
	for _, date := range dates {
		for _, region := range regions {
			for _, field := range rankFields {
				for _, rankType := range rankTypes {
					for _, categoryID := range categoryIDs {
						for _, createdByAI := range createdByAIOptions {
							for pageNum := 1; pageNum <= maxPages; pageNum++ {
								paramsList = append(paramsList, dto.EchotikRanklistParams{
									Date:              date,
									Region:            region,
									VideoRankField:    field,
									RankType:          rankType,
									ProductCategoryID: categoryID,
									CreatedByAI:       createdByAI,
									PageNum:           pageNum,
									PageSize:          pageSize,
								})
							}
						}
					}
				}
			}
		}
	}

	return paramsList
}

// buildRecentDates 生成最近 N 天的日期字符串（yyyy-MM-dd，包含今天）。
func buildRecentDates(days int) []string {
	if days <= 0 {
		days = 3
	}
	now := time.Now()
	result := make([]string, 0, days)
	for i := days - 1; i >= 0; i-- {
		t := now.AddDate(0, 0, -i)
		result = append(result, t.Format("2006-01-02"))
	}
	return result
}
