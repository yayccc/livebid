package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	apiconfig "github.com/yayccc/livebid/services/api-gateway/internal/config"
)

const maxUploadFileSize = 10 << 20

type FileHandler struct {
	storage objectStorage
}

type objectStorage interface {
	PutObject(ctx context.Context, key string, contentType string, body []byte) (string, error)
}

type uploadFileResponse struct {
	URL       string `json:"url"`
	ObjectKey string `json:"object_key"`
}

type uploadGoodsCoverResponse struct {
	CoverURL string `json:"cover_url"`
}

type s3ObjectStorage struct {
	client        *s3.Client
	bucket        string
	endpoint      string
	publicBaseURL string
}

func NewFileHandler(storage objectStorage) *FileHandler {
	return &FileHandler{storage: storage}
}

func NewS3ObjectStorage(ctx context.Context, cfg apiconfig.StorageConfig) (*s3ObjectStorage, error) {
	endpoint := normalizeEndpoint(cfg.Endpoint)
	if endpoint == "" {
		return nil, errors.New("storage endpoint is required")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("storage bucket is required")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, errors.New("storage access key id and secret access key are required")
	}

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = cfg.PathStyle
	})

	return &s3ObjectStorage{
		client:        client,
		bucket:        strings.TrimSpace(cfg.Bucket),
		endpoint:      endpoint,
		publicBaseURL: strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"),
	}, nil
}

func (h *FileHandler) Upload(c *gin.Context) {
	fileURL, objectKey, ok := h.upload(c)
	if !ok {
		return
	}

	respondOK(c, uploadFileResponse{
		URL:       fileURL,
		ObjectKey: objectKey,
	})
}

func (h *FileHandler) UploadGoodsCover(c *gin.Context) {
	fileURL, _, ok := h.upload(c)
	if !ok {
		return
	}

	respondOK(c, uploadGoodsCoverResponse{
		CoverURL: fileURL,
	})
}

func (h *FileHandler) upload(c *gin.Context) (string, string, bool) {
	if h.storage == nil {
		respondError(c, http.StatusInternalServerError, "file storage is not configured")
		return "", "", false
	}

	file, err := c.FormFile("file")
	if err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "file is required")
		return "", "", false
	}
	if file.Size > maxUploadFileSize {
		respondError(c, http.StatusBadRequest, "file too large")
		return "", "", false
	}

	content, err := readMultipartFile(file)
	if err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "invalid file")
		return "", "", false
	}
	if int64(len(content)) > maxUploadFileSize {
		respondError(c, http.StatusBadRequest, "file too large")
		return "", "", false
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}

	objectKey := uploadObjectKey(file.Filename)
	fileURL, err := h.storage.PutObject(c.Request.Context(), objectKey, contentType, content)
	if err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadGateway, "file storage unavailable")
		return "", "", false
	}
	return fileURL, objectKey, true
}

func (s *s3ObjectStorage) PutObject(ctx context.Context, key string, contentType string, body []byte) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}
	return s.objectURL(key), nil
}

func readMultipartFile(file *multipart.FileHeader) ([]byte, error) {
	opened, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	var body bytes.Buffer
	_, err = io.Copy(&body, io.LimitReader(opened, maxUploadFileSize+1))
	if err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func uploadObjectKey(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".jpg"
	}
	return fmt.Sprintf("files/%d%s", time.Now().UnixNano(), ext)
}

func (s *s3ObjectStorage) objectURL(key string) string {
	base := s.publicBaseURL
	if base == "" {
		base = strings.TrimRight(s.endpoint, "/") + "/" + pathEscape(s.bucket)
	}
	return strings.TrimRight(base, "/") + "/" + pathEscape(key)
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return strings.TrimRight(endpoint, "/")
	}
	return "http://" + strings.TrimRight(endpoint, "/")
}

func pathEscape(value string) string {
	parts := strings.Split(path.Clean("/"+value), "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/")
}
