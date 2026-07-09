package gemini

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIContent2GeminiParts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("text content", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		parts, err := OpenAIContent2GeminiParts(c, []dto.MediaContent{
			{Type: dto.ContentTypeText, Text: "hello"},
		}, &relaycommon.RelayInfo{})
		require.NoError(t, err)
		require.Len(t, parts, 1)
		assert.Equal(t, "hello", parts[0].Text)
	})

	t.Run("file content with Google file URI", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		parts, err := OpenAIContent2GeminiParts(c, []dto.MediaContent{
			{
				Type: dto.ContentTypeFile,
				File: &dto.MessageFile{FileData: "files/abc123"},
			},
		}, &relaycommon.RelayInfo{})
		require.NoError(t, err)
		require.Len(t, parts, 1)
		require.NotNil(t, parts[0].FileData)
		assert.Contains(t, parts[0].FileData.FileUri, "files/abc123")
	})

	t.Run("EasyRouter forces file URI to inlineData", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		parts, err := OpenAIContent2GeminiParts(c, []dto.MediaContent{
			{
				Type: dto.ContentTypeFile,
				File: &dto.MessageFile{FileData: "files/abc123"},
			},
		}, &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType: constant.ChannelTypeEasyRouter,
			},
		})
		require.NoError(t, err)
		require.Len(t, parts, 1)
		require.Nil(t, parts[0].FileData)
		require.NotNil(t, parts[0].InlineData)
		assert.Equal(t, "application/octet-stream", parts[0].InlineData.MimeType)
	})
}

func TestBuildInlineData(t *testing.T) {
	t.Run("raw base64 data", func(t *testing.T) {
		inline, err := BuildInlineData("aGVsbG8=")
		require.NoError(t, err)
		assert.Equal(t, "aGVsbG8=", inline.Data)
	})

	t.Run("data URI", func(t *testing.T) {
		inline, err := BuildInlineData("data:image/png;base64,iVBORw0KGgo=")
		require.NoError(t, err)
		assert.Equal(t, "image/png", inline.MimeType)
		assert.Equal(t, "iVBORw0KGgo=", inline.Data)
	})

	t.Run("rejects URL", func(t *testing.T) {
		_, err := BuildInlineData("https://example.com/image.png")
		require.Error(t, err)
	})
}

func TestBuildFileData(t *testing.T) {
	t.Run("full Google file URI", func(t *testing.T) {
		fd := BuildFileData("files/abc123", "video/mp4")
		require.NotNil(t, fd)
		assert.Contains(t, fd.FileUri, "generativelanguage.googleapis.com/files/abc123")
		assert.Equal(t, "video/mp4", fd.MimeType)
	})

	t.Run("empty URI returns nil", func(t *testing.T) {
		assert.Nil(t, BuildFileData("", ""))
	})
}
