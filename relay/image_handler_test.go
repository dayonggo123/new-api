package relay

import (
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
