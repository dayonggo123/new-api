package muyu

const (
	ChannelName = "muyu-video"
)

// Base URL for Muyu API
const BaseURL = "https://api.muyu-aigc.bbroot.com"

// API endpoints
const (
	AssetUploadEndpoint = "/api/assets"
	AssetLookupEndpoint  = "/api/assets/lookup"
	CatalogEndpoint       = "/api/catalog"
	TasksEndpoint         = "/api/tasks"
	TaskQueryEndpoint     = "/api/tasks/%s" // with taskId placeholder
)

// Status constants
const (
	StatusGenerating        = "GENERATING"
	StatusSuccess           = "SUCCESS"
	StatusFailed            = "FAILED"
	StatusSubmitting        = "SUBMITTING"
	StatusSubmissionUnknown = "SUBMISSION_UNKNOWN"
)

// Default values
const (
	DefaultAspect      = "16:9"
	DefaultDuration    = 10
	DefaultResolution  = "720p"
)

// All models - Muyu provides dynamic model list via catalog API
// These are placeholder models that will be validated against catalog response
var ModelList = []string{
	// Channel 2 models
	"channel2/video-2.0-fast-720P",
	"channel2/video-2.0-720P",
	"channel2/video-2.0-fast-1080P",
	"channel2/video-2.0-1080P",
	// Channel 3 models
	"channel3/limited_seedance_2_full_720p",
	"channel3/limited_seedance_2_full_1080p",
	"channel3/limited_seedance_2_fast_720p",
	"channel3/limited_seedance_2_fast_1080p",
	// Channel 5 models
	"channel5/zy-sd2-0-fast-9-images",
	"channel5/zy-sd2-0-9-images",
	"channel5/zy-sd2-0-fast-text",
	"channel5/zy-sd2-0-text",
	// Channel 6 models
	"channel6/Seedance2.0",
}
