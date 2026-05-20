package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ChannelHealth tracks per-(channelID, model) request outcomes in memory.
// No DB persistence — data fades via DecayChannelHealth and rebuilds after restart.
type ChannelHealth struct {
	SuccessCount     int
	FailCount        int
	ConsecutiveFails int
	LastRequestTime  time.Time
}

var (
	channelHealthStats = make(map[string]*ChannelHealth)
	channelHealthMu    sync.RWMutex
)

func channelHealthKey(channelID int, model string) string {
	return fmt.Sprintf("%d:%s", channelID, model)
}

// RecordChannelRequestResult updates the health stats for a channel+model after a request.
func RecordChannelRequestResult(channelID int, model string, success bool) {
	key := channelHealthKey(channelID, model)
	channelHealthMu.Lock()
	defer channelHealthMu.Unlock()

	health, ok := channelHealthStats[key]
	if !ok {
		health = &ChannelHealth{}
		channelHealthStats[key] = health
	}

	health.LastRequestTime = time.Now()
	if success {
		health.SuccessCount++
		health.ConsecutiveFails = 0
	} else {
		health.FailCount++
		health.ConsecutiveFails++
	}
}

// GetChannelEffectiveWeight returns a weight adjusted by the channel's recent success rate.
// If there are fewer than minSamples total requests, the base weight is returned unchanged.
func GetChannelEffectiveWeight(channelID int, model string, baseWeight int) int {
	if baseWeight <= 0 {
		return baseWeight
	}

	health := getChannelHealth(channelID, model)
	if health == nil {
		return baseWeight
	}

	total := health.SuccessCount + health.FailCount
	if total < common.SmartSwitchMinSamples {
		return baseWeight
	}

	successRate := float64(health.SuccessCount) / float64(total)
	// Factor ranges from 0.2 (0% success) to 2.0 (100% success)
	// 50% success -> factor = 1.0 (no change)
	factor := 0.2 + successRate*1.8

	effectiveWeight := int(float64(baseWeight) * factor)
	if effectiveWeight < 1 {
		effectiveWeight = 1
	}
	return effectiveWeight
}

func getChannelHealth(channelID int, model string) *ChannelHealth {
	key := channelHealthKey(channelID, model)
	channelHealthMu.RLock()
	defer channelHealthMu.RUnlock()
	return channelHealthStats[key]
}

// DecayChannelHealth halves all counters to fade old data.
// If both SuccessCount and FailCount reach 0, ConsecutiveFails is also reset.
func DecayChannelHealth() {
	channelHealthMu.Lock()
	defer channelHealthMu.Unlock()

	for _, health := range channelHealthStats {
		health.SuccessCount = health.SuccessCount / 2
		health.FailCount = health.FailCount / 2
		if health.SuccessCount == 0 && health.FailCount == 0 {
			health.ConsecutiveFails = 0
		}
	}
}

// StartChannelHealthDecay starts a background goroutine that periodically decays health stats.
func StartChannelHealthDecay(interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	go func() {
		for {
			time.Sleep(interval)
			DecayChannelHealth()
		}
	}()
}
