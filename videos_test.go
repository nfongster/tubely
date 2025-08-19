package main

import (
	"testing"
)

func TestGetVideoAspectRatio(t *testing.T) {
	tests := []struct {
		filePath            string
		expectedAspectRatio string
	}{
		{"./samples/boots-video-horizontal.mp4", "16:9"}, //1280 x 720
		{"./samples/boots-video-vertical.mp4", "9:16"},   //608 x 1080
	}

	for _, tt := range tests {
		actualAspectRatio, err := getVideoAspectRatio(tt.filePath)
		if err != nil {
			t.Errorf("Failed get aspect ratio: %v", err)
		}
		if tt.expectedAspectRatio != actualAspectRatio {
			t.Errorf("Expected aspect ratio does not match actual; %s != %s", tt.expectedAspectRatio, actualAspectRatio)
		}
	}
}
