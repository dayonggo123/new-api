package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var (
	r2Client   *s3.Client
	r2Presign  *s3.PresignClient
	r2Bucket   string
	r2Expiry   time.Duration
	r2Enabled  bool
	r2Endpoint string
	r2Region   string
	r2Prefix   string
	r2MaxSize  int64
)

// InitR2 initializes the R2/S3-compatible storage client from environment variables.
// If R2_ENDPOINT is not set, R2 uploads are silently disabled.
func InitR2() error {
	r2Endpoint = strings.TrimRight(os.Getenv("R2_ENDPOINT"), "/")
	if r2Endpoint == "" {
		common.SysLog("R2 storage not configured (R2_ENDPOINT empty), image upload endpoint disabled")
		r2Enabled = false
		return nil
	}

	r2Bucket = os.Getenv("R2_BUCKET")
	if r2Bucket == "" {
		return fmt.Errorf("R2_BUCKET is required when R2_ENDPOINT is set")
	}

	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("R2_ACCESS_KEY_ID and R2_SECRET_ACCESS_KEY are required")
	}

	r2Region = os.Getenv("R2_REGION")
	if r2Region == "" {
		r2Region = "auto"
	}

	r2Expiry = time.Duration(getEnvInt("R2_URL_EXPIRY_SECONDS", 600)) * time.Second

	r2Prefix = os.Getenv("R2_PATH_PREFIX")
	if r2Prefix == "" {
		r2Prefix = "tmp/"
	}
	if !strings.HasSuffix(r2Prefix, "/") {
		r2Prefix += "/"
	}

	r2MaxSize = int64(getEnvInt("R2_MAX_FILE_SIZE_MB", 15)) * 1024 * 1024

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(r2Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return fmt.Errorf("load R2 config failed: %w", err)
	}

	r2Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(r2Endpoint)
		o.UsePathStyle = true
	})
	r2Presign = s3.NewPresignClient(r2Client)
	r2Enabled = true

	common.SysLog(fmt.Sprintf("R2 storage initialized: bucket=%s endpoint=%s expiry=%ds prefix=%s max_size=%dMB",
		r2Bucket, r2Endpoint, int(r2Expiry.Seconds()), r2Prefix, r2MaxSize/1024/1024))
	return nil
}

// R2Enabled reports whether R2-backed image uploads are available.
func R2Enabled() bool {
	return r2Enabled
}

// R2URLExpiry returns the configured presigned URL expiry duration.
func R2URLExpiry() time.Duration {
	return r2Expiry
}

// UploadImage reads image data from reader and uploads it to R2, returning a presigned URL.
func UploadImage(reader io.Reader, contentType string, size int64) (url string, key string, err error) {
	if !r2Enabled {
		return "", "", fmt.Errorf("R2 storage is not configured")
	}
	if size > r2MaxSize {
		return "", "", fmt.Errorf("file too large: max %d bytes", r2MaxSize)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", "", fmt.Errorf("read upload data failed: %w", err)
	}
	return UploadImageBytes(data, contentType)
}

// UploadImageBytes uploads image bytes to R2 and returns a presigned GET URL plus object key.
func UploadImageBytes(data []byte, contentType string) (url string, key string, err error) {
	if !r2Enabled {
		return "", "", fmt.Errorf("R2 storage is not configured")
	}
	if int64(len(data)) > r2MaxSize {
		return "", "", fmt.Errorf("file too large: %d > %d bytes", len(data), r2MaxSize)
	}

	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		return "", "", fmt.Errorf("invalid content type: %s", contentType)
	}

	ext := extFromContentType(contentType)
	key = r2Prefix + uuid.New().String() + "." + ext

	_, err = r2Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        &r2Bucket,
		Key:           &key,
		Body:          bytes.NewReader(data),
		ContentType:   &contentType,
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return "", key, fmt.Errorf("R2 PutObject failed: %w", err)
	}

	presigned, err := r2Presign.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &r2Bucket,
		Key:    &key,
	}, s3.WithPresignExpires(r2Expiry))
	if err != nil {
		return "", key, fmt.Errorf("R2 PresignGetObject failed: %w", err)
	}

	return presigned.URL, key, nil
}

// PresignedURL generates a new presigned URL for an existing R2 object key.
func PresignedURL(key string) (string, error) {
	if !r2Enabled {
		return "", fmt.Errorf("R2 storage is not configured")
	}
	if key == "" {
		return "", fmt.Errorf("key is required")
	}
	presigned, err := r2Presign.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &r2Bucket,
		Key:    &key,
	}, s3.WithPresignExpires(r2Expiry))
	if err != nil {
		return "", fmt.Errorf("R2 PresignGetObject failed: %w", err)
	}
	return presigned.URL, nil
}

func getEnvInt(name string, defaultVal int) int {
	s := os.Getenv(name)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

func extFromContentType(mime string) string {
	parts := strings.Split(strings.ToLower(mime), "/")
	if len(parts) != 2 {
		return "bin"
	}
	sub := strings.TrimSpace(parts[1])
	switch sub {
	case "png", "jpeg", "jpg", "gif", "webp", "bmp", "heic", "heif", "svg+xml":
		if sub == "svg+xml" {
			return "svg"
		}
		return sub
	default:
		return "bin"
	}
}
