package relay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	model.DB = db
	model.LOG_DB = db

	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	if err := db.AutoMigrate(
		&model.Task{},
		&model.User{},
		&model.Token{},
		&model.Log{},
		&model.Channel{},
		&model.UserSubscription{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	os.Exit(m.Run())
}

func makeRelayInfoWithNilTaskRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		UserId:     1,
		UsingGroup: "default",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenAI,
			ChannelId:   1,
		},
		OriginModelName: "dall-e-3",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		Request: &dto.ImageRequest{
			Model:  "dall-e-3",
			Prompt: "a cute cat",
			Size:   "1024x1024",
		},
		PriceData: types.PriceData{
			Quota: 1000,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
			OtherRatios: map[string]float64{},
		},
	}
}

// TestImageHelper_GPTImage2_OpenAI verifies that gpt-image-2 on an OpenAI
// channel enters the sync-to-async task branch and returns a non-empty task_id.
func TestImageHelper_GPTImage2_OpenAI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	// Set the channel context values that InitChannelMeta reads.
	c.Set(string(constant.ContextKeyChannelType), constant.ChannelTypeOpenAI)
	c.Set(string(constant.ContextKeyChannelId), 1)
	c.Set(string(constant.ContextKeyOriginalModel), "gpt-image-2")
	c.Set(string(constant.ContextKeyChannelBaseUrl), "https://api.openai.com")
	c.Set(string(constant.ContextKeyChannelKey), "sk-test")

	info := &relaycommon.RelayInfo{
		UserId:     1,
		UsingGroup: "default",
		OriginModelName: "gpt-image-2",
		RelayMode:  relayconstant.RelayModeImagesGenerations,
		Request: &dto.ImageRequest{
			Model:  "gpt-image-2",
			Prompt: "a futuristic city",
			Size:   "1024x1024",
		},
		PriceData: types.PriceData{
			Quota: 1000,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
			OtherRatios: map[string]float64{},
		},
	}

	err := ImageHelper(c, info)

	// If gpt-image-2 is routed through handleSyncImageAsTaskRelay, the
	// response is written and err is nil. If it falls through to runSyncImageRelay,
	// it would fail to reach the upstream and return an error.
	require.Nil(t, err, "ImageHelper should route gpt-image-2 through sync-to-async task flow")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.TaskID, "gpt-image-2 submit response must contain a non-empty task_id")
	assert.Equal(t, dto.VideoStatusQueued, resp.Status)
	assert.Equal(t, "image.generation", resp.Object)
	assert.Equal(t, "gpt-image-2", resp.Model)
}

// TestImageHelper_GPTImage2_OpenAICompatibleChannels verifies that gpt-image-2
// on OpenAI-compatible channels (OpenAIMax, OpenRouter, Xinference, Custom)
// enters the sync-to-async task branch and returns a non-empty task_id.
func TestImageHelper_GPTImage2_OpenAICompatibleChannels(t *testing.T) {
	channelTypes := []int{
		constant.ChannelTypeOpenAI,
		constant.ChannelTypeOpenAIMax,
		constant.ChannelTypeOpenRouter,
		constant.ChannelTypeXinference,
		constant.ChannelTypeCustom,
	}

	for _, channelType := range channelTypes {
		t.Run(fmt.Sprintf("channel_type_%d", channelType), func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

			c.Set(string(constant.ContextKeyChannelType), channelType)
			c.Set(string(constant.ContextKeyChannelId), 1)
			c.Set(string(constant.ContextKeyOriginalModel), "gpt-image-2")
			c.Set(string(constant.ContextKeyChannelBaseUrl), "https://api.example.com")
			c.Set(string(constant.ContextKeyChannelKey), "sk-test")

			info := &relaycommon.RelayInfo{
				UserId:     1,
				UsingGroup: "default",
				OriginModelName: "gpt-image-2",
				RelayMode:  relayconstant.RelayModeImagesGenerations,
				Request: &dto.ImageRequest{
					Model:  "gpt-image-2",
					Prompt: "a futuristic city",
					Size:   "1024x1024",
				},
				PriceData: types.PriceData{
					Quota: 1000,
					GroupRatioInfo: types.GroupRatioInfo{
						GroupRatio: 1,
					},
					OtherRatios: map[string]float64{},
				},
			}

			err := ImageHelper(c, info)
			require.Nil(t, err, "ImageHelper should route gpt-image-2 on OpenAI-compatible channel through sync-to-async task flow")
			assert.Equal(t, http.StatusOK, w.Code)

			var resp dto.OpenAIVideo
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &resp))
			assert.NotEmpty(t, resp.TaskID, "submit response must contain a non-empty task_id")
			assert.Equal(t, dto.VideoStatusQueued, resp.Status)
			assert.Equal(t, "image.generation", resp.Object)
		})
	}
}

// TestIsSyncImageAsyncChannel_ExcludesTaskImageChannels verifies that
// isSyncImageAsyncChannel returns false for task image channels (APIMart etc.)
// even though their API type is OpenAI-compatible.
func TestIsSyncImageAsyncChannel_ExcludesTaskImageChannels(t *testing.T) {
	taskImageChannels := []int{
		constant.ChannelTypeAPIMart,
		constant.ChannelTypeDuoYuanTanSuo,
		constant.ChannelTypeZhangyuge,
		constant.ChannelTypeVeo,
	}
	for _, channelType := range taskImageChannels {
		assert.False(t, isSyncImageAsyncChannel(channelType), "channel type %d should not be treated as sync-to-async", channelType)
	}
}

// TestIsSyncImageAsyncChannel_IncludesOpenAICompatibleChannels verifies that
// isSyncImageAsyncChannel returns true for OpenAI-compatible channels.
func TestIsSyncImageAsyncChannel_IncludesOpenAICompatibleChannels(t *testing.T) {
	openAICompatibleChannels := []int{
		constant.ChannelTypeOpenAI,
		constant.ChannelTypeOpenAIMax,
		constant.ChannelTypeOpenRouter,
		constant.ChannelTypeXinference,
		constant.ChannelTypeCustom,
	}
	for _, channelType := range openAICompatibleChannels {
		assert.True(t, isSyncImageAsyncChannel(channelType), "channel type %d should be treated as sync-to-async", channelType)
	}
}

// TestHandleSyncImageAsTaskRelay_NilTaskRelayInfo verifies the fix for the
// nil pointer dereference panic: handleSyncImageAsTaskRelay must initialize
// info.TaskRelayInfo before assigning PublicTaskID, and must complete the
// request successfully when TaskRelayInfo is nil.
func TestHandleSyncImageAsTaskRelay_NilTaskRelayInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := makeRelayInfoWithNilTaskRelayInfo()
	require.Nil(t, info.TaskRelayInfo, "precondition: TaskRelayInfo should be nil")

	// Must not panic and should return nil (success).
	err := handleSyncImageAsTaskRelay(c, info)

	require.Nil(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, info.TaskRelayInfo, "TaskRelayInfo should be initialized")
	assert.NotEmpty(t, info.TaskRelayInfo.PublicTaskID, "PublicTaskID should be set")

	// Verify the response body contains the generated public task ID.
	var resp dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, info.TaskRelayInfo.PublicTaskID, resp.ID)
	assert.Equal(t, info.TaskRelayInfo.PublicTaskID, resp.TaskID)
	assert.Equal(t, dto.VideoStatusQueued, resp.Status)
	assert.Equal(t, "image.generation", resp.Object)
}

// TestHandleSyncImageAsTaskRelay_GPTImage2 ensures that gpt-image-2 on an
// OpenAI channel is wrapped as a sync-to-async task and returns a non-empty
// task_id in the submit response.
func TestHandleSyncImageAsTaskRelay_GPTImage2(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := &relaycommon.RelayInfo{
		UserId:     1,
		UsingGroup: "default",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenAI,
			ChannelId:   1,
		},
		OriginModelName: "gpt-image-2",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		Request: &dto.ImageRequest{
			Model:  "gpt-image-2",
			Prompt: "a futuristic city",
			Size:   "1024x1024",
		},
		PriceData: types.PriceData{
			Quota: 1000,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
			OtherRatios: map[string]float64{},
		},
	}
	require.Nil(t, info.TaskRelayInfo)

	err := handleSyncImageAsTaskRelay(c, info)

	require.Nil(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, info.TaskRelayInfo)
	assert.NotEmpty(t, info.TaskRelayInfo.PublicTaskID)

	var resp dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, info.TaskRelayInfo.PublicTaskID, resp.ID)
	assert.Equal(t, info.TaskRelayInfo.PublicTaskID, resp.TaskID)
	assert.NotEmpty(t, resp.TaskID, "gpt-image-2 submit response must contain a non-empty task_id")
	assert.Equal(t, dto.VideoStatusQueued, resp.Status)
	assert.Equal(t, "image.generation", resp.Object)
	assert.Equal(t, "gpt-image-2", resp.Model)
}
