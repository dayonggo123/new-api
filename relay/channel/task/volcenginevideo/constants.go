package volcenginevideo

// ModelList contains the VolcEngine / Doubao video generation models supported
// by the VolcEngine channel. The actual model ID sent to upstream is the value
// configured by the user (after optional channel model mapping).
var ModelList = []string{
	"doubao-seedance-1-0-pro-250528",
	"doubao-seedance-1-0-lite-t2v",
	"doubao-seedance-1-0-lite-i2v",
	"doubao-seedance-1-5-pro-251215",
	"doubao-seedance-2-0-260128",
	"seedance-2-0-260128",
	"doubao-seedance-2-0-fast-260128",
	"seedance-2-0-fast-260128",
}

// ChannelName is the identifier used in logs and channel metadata.
var ChannelName = "volcengine-video"

// videoInputRatioMap stores the discount ratio when the request includes a
// video reference. Administrators should configure the model base price as the
// non-video-reference price; the system will multiply by this ratio when a
// video reference is detected.
var videoInputRatioMap = map[string]float64{
	"doubao-seedance-2-0-260128":      28.0 / 46.0,
	"seedance-2-0-260128":             28.0 / 46.0,
	"doubao-seedance-2-0-fast-260128": 22.0 / 37.0,
	"seedance-2-0-fast-260128":        22.0 / 37.0,
}

// GetVideoInputRatio returns the video-input discount ratio for the given model.
func GetVideoInputRatio(modelName string) (float64, bool) {
	r, ok := videoInputRatioMap[modelName]
	return r, ok
}
