package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

const (
	ratio16_9 float64 = 16.0 / 9.0
	ratio9_16 float64 = 9.0 / 16.0
	tolerance float64 = 0.001

	string16_9                   string = "16:9"
	string9_16                   string = "9:16"
	stringUnsupportedAspectRatio string = "other"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return "", err
	}

	sd := streamData{}
	if err := json.Unmarshal(b.Bytes(), &sd); err != nil {
		return "", err
	}

	width, height := sd.Streams[0].Width, sd.Streams[0].Height
	if height == 0 {
		return "", fmt.Errorf("height of video was 0")
	}
	div := float64(width) / float64(height)
	switch {
	case math.Abs(div-float64(ratio16_9)) < tolerance:
		return string16_9, nil
	case math.Abs(div-float64(ratio9_16)) < tolerance:
		return string9_16, nil
	default:
		return stringUnsupportedAspectRatio, nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return outputPath, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	s := strings.Split(*video.VideoURL, ",")
	if len(s) != 2 {
		return video, fmt.Errorf("db video URL was in improper format")
	}

	bucket, key := s[0], s[1]
	psURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Second)
	if err != nil {
		return video, err
	}

	video.VideoURL = &psURL
	return video, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	objInput := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	req, err := presignClient.PresignGetObject(context.Background(), &objInput, s3.WithPresignExpires(expireTime))
	return req.URL, err
}

type streamData struct {
	Streams []struct {
		Index              int    `json:"index"`
		CodecName          string `json:"codec_name"`
		CodecLongName      string `json:"codec_long_name"`
		CodecType          string `json:"codec_type"`
		CodecTagString     string `json:"codec_tag_string"`
		CodecTag           string `json:"codec_tag"`
		Width              int    `json:"width"`
		Height             int    `json:"height"`
		CodedWidth         int    `json:"coded_width"`
		CodedHeight        int    `json:"coded_height"`
		ClosedCaptions     int    `json:"closed_captions"`
		FilmGrain          int    `json:"film_grain"`
		HasBFrames         int    `json:"has_b_frames"`
		SampleAspectRatio  string `json:"sample_aspect_ratio"`
		DisplayAspectRatio string `json:"display_aspect_ratio"`
		PixFmt             string `json:"pix_fmt"`
		Level              int    `json:"level"`
		ColorRange         string `json:"color_range"`
		ColorSpace         string `json:"color_space"`
		Refs               int    `json:"refs"`
		RFrameRate         string `json:"r_frame_rate"`
		AvgFrameRate       string `json:"avg_frame_rate"`
		TimeBase           string `json:"time_base"`
		Disposition        struct {
			Default         int `json:"default"`
			Dub             int `json:"dub"`
			Original        int `json:"original"`
			Comment         int `json:"comment"`
			Lyrics          int `json:"lyrics"`
			Karaoke         int `json:"karaoke"`
			Forced          int `json:"forced"`
			HearingImpaired int `json:"hearing_impaired"`
			VisualImpaired  int `json:"visual_impaired"`
			CleanEffects    int `json:"clean_effects"`
			AttachedPic     int `json:"attached_pic"`
			TimedThumbnails int `json:"timed_thumbnails"`
			NonDiegetic     int `json:"non_diegetic"`
			Captions        int `json:"captions"`
			Descriptions    int `json:"descriptions"`
			Metadata        int `json:"metadata"`
			Dependent       int `json:"dependent"`
			StillImage      int `json:"still_image"`
		} `json:"disposition"`
	} `json:"streams"`
}
