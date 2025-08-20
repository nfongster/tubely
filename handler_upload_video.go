package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// 1) Set a 1 GB upload limit
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	// 2) Get the video ID from the header and parse as a UUID
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// 3) Authenticate user and get their ID
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// 4) Get video metadata from DB
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Did not find video in DB", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "User ID did not match video's user ID", err)
		return
	}

	// 5) Parse uploaded video file from form data
	videoFile, _, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error extracting video file", err)
		return
	}
	defer videoFile.Close()

	// 6) Check that uploaded file is mp4
	mediaType, _, err := mime.ParseMediaType("video/mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to parse media type", err)
		return
	}

	// 7) Save uploaded file to temp file on disk
	tmpFile, err := os.CreateTemp("", "tubely-upload-*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create temp file", err)
		return
	}
	tmpFileKey := tmpFile.Name()
	defer os.Remove(tmpFileKey)
	defer tmpFile.Close()
	if _, err := io.Copy(tmpFile, videoFile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to copy video file", err)
		return
	}

	// 7.1) Process video (so that it has a fast start encoding)
	tmpFileKey, err = processVideoForFastStart(tmpFileKey)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing temp file", err)
		return
	}
	fmt.Printf("Temp File Key (post-process): %s\n", tmpFileKey)
	prcFile, err := os.Open(tmpFileKey)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error opening processed temp file", err)
		return
	}
	defer prcFile.Close()

	// 7.2) Get the aspect ratio
	ar, err := getVideoAspectRatio(tmpFileKey)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get aspect ratio", err)
		return
	}
	var arPrefix string
	switch ar {
	case string16_9:
		arPrefix = "landscape"
	case string9_16:
		arPrefix = "portrait"
	default:
		arPrefix = "other"
	}
	tmpFileKey = filepath.Join(arPrefix, tmpFileKey)
	fmt.Printf("Temp File Key: %s\n", tmpFileKey)

	// 8) Reset temp file's file pointer to the beginning
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reseting temp file", err)
		return
	}

	// 9) Put video object into S3
	s3Params := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &tmpFileKey,
		Body:        prcFile,
		ContentType: &mediaType,
	}
	if _, err = cfg.s3Client.PutObject(r.Context(), &s3Params); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading video to S3", err)
		return
	}

	// 10) Update video URL in DB
	videoURL := getS3VideoUrl(cfg.s3Bucket, tmpFileKey)
	fmt.Printf("Video URL: %s\n", videoURL)
	video.VideoURL = &videoURL
	video.UpdatedAt = time.Now()
	if err = cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video in DB", err)
		return
	}

	// 11) Sign the URL before sending it over the wire
	video, err = cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error signing video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}

func getS3VideoUrl(bucket, key string) string {
	return fmt.Sprintf("%s,%s", bucket, key)
}
