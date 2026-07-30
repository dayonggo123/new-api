package muyu

import (
	"testing"
)

func TestStatusConstants(t *testing.T) {
	// Verify status constants are properly defined
	if StatusGenerating == "" {
		t.Error("StatusGenerating should not be empty")
	}
	if StatusSuccess == "" {
		t.Error("StatusSuccess should not be empty")
	}
	if StatusFailed == "" {
		t.Error("StatusFailed should not be empty")
	}
}

func TestModelListNotEmpty(t *testing.T) {
	if len(ModelList) == 0 {
		t.Error("ModelList should not be empty")
	}

	// Check all models have valid prefixes
	for _, model := range ModelList {
		if model == "" {
			t.Error("ModelList contains empty string")
		}
		// All Muyu models should start with "channel"
		if len(model) < 8 || model[:8] != "channel" {
			t.Errorf("Model %q should start with 'channel'", model)
		}
	}
}

func TestEndpoints(t *testing.T) {
	if TasksEndpoint == "" {
		t.Error("TasksEndpoint should not be empty")
	}
	if AssetUploadEndpoint == "" {
		t.Error("AssetUploadEndpoint should not be empty")
	}
	if CatalogEndpoint == "" {
		t.Error("CatalogEndpoint should not be empty")
	}
}

func TestDefaultValues(t *testing.T) {
	if DefaultAspect == "" {
		t.Error("DefaultAspect should not be empty")
	}
	if DefaultDuration <= 0 {
		t.Error("DefaultDuration should be positive")
	}
	if DefaultResolution == "" {
		t.Error("DefaultResolution should not be empty")
	}
}

func TestAspectNormalization(t *testing.T) {
	adaptor := &TaskAdaptor{}

	tests := []struct {
		input    string
		expected string
	}{
		{"16:9", "16:9"},
		{"16/9", "16:9"},
		{"9:16", "9:16"},
		{"9/16", "9:16"},
		{"1:1", "1:1"},
		{"unknown", "16:9"}, // defaults to DefaultAspect
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := adaptor.normalizeAspect(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeAspect(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolutionNormalization(t *testing.T) {
	adaptor := &TaskAdaptor{}

	tests := []struct {
		input    string
		expected string
	}{
		{"1080p", "1080p"},
		{"1080", "1080p"},
		{"720p", "720p"},
		{"720", "720p"},
		{"480p", "480p"},
		{"unknown", "720p"}, // defaults to DefaultResolution
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := adaptor.normalizeResolution(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeResolution(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
