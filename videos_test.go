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
			t.Errorf("Failed to get aspect ratio: %v", err)
		}
		if tt.expectedAspectRatio != actualAspectRatio {
			t.Errorf("Expected aspect ratio does not match actual; %s != %s", tt.expectedAspectRatio, actualAspectRatio)
		}
	}
}

func TestProcessVideoForFastStart(t *testing.T) {
	inputPath := "./samples/boots-video-horizontal.mp4"
	expectedOutputPath := inputPath + ".processing"
	outputPath, err := processVideoForFastStart(inputPath)
	if err != nil {
		t.Errorf("Failed to get output path: %v", err)
	}
	if outputPath != expectedOutputPath {
		t.Errorf("Expected path does not match actual: %s != %s", expectedOutputPath, outputPath)
	}
	// TODO: Delete file after test run!
}
