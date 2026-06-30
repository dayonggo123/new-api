package service

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

var echotikRanklistCleanupRunning atomic.Bool

// StartEchotikRanklistCleanupTask 启动 EchoTik 榜单缓存清理任务（仅主节点执行）。
func StartEchotikRanklistCleanupTask() {
	if !common.IsMasterNode {
		common.SysLog("Echotik ranklist cleanup task skipped on non-master node")
		return
	}

	setting := operation_setting.GetEchotikSetting()
	if !setting.EchotikEnabled || !setting.EchotikCacheEnabled {
		common.SysLog("Echotik ranklist cleanup task skipped: cache not enabled")
		return
	}

	gopool.Go(func() {
		echotikRanklistCleanupTaskLoop()
	})
}

func echotikRanklistCleanupTaskLoop() {
	setting := operation_setting.GetEchotikSetting()
	frequency := setting.EchotikSyncFrequencyHours
	if frequency <= 0 {
		frequency = 24
	}
	interval := time.Duration(frequency) * time.Hour

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	common.SysLog(fmt.Sprintf("Echotik ranklist cleanup task started, interval=%s", interval))

	for {
		StartEchotikRanklistCleanupOnce()
		<-ticker.C
	}
}

// StartEchotikRanklistCleanupOnce 执行一次缓存清理，同一时间只允许一个实例运行。
func StartEchotikRanklistCleanupOnce() {
	if !echotikRanklistCleanupRunning.CompareAndSwap(false, true) {
		common.SysLog("Echotik ranklist cleanup skipped: previous round still running")
		return
	}
	defer echotikRanklistCleanupRunning.Store(false)

	setting := operation_setting.GetEchotikSetting()
	retentionDays := setting.EchotikCacheRetentionDays
	if retentionDays <= 0 {
		retentionDays = 1
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	rowsAffected, err := model.DeleteEchotikRanklistSnapshotsBefore(cutoff)
	if err != nil {
		common.SysError(fmt.Sprintf("Echotik ranklist cleanup failed: %s", err.Error()))
		return
	}

	if rowsAffected > 0 {
		common.SysLog(fmt.Sprintf("Echotik ranklist cleanup finished: removed %d stale snapshots", rowsAffected))
	}
}
