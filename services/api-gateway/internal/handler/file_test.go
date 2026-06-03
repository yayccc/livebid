package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockObjectStorage struct {
	putObject func(ctx context.Context, key string, contentType string, body []byte) (string, error)
}

func (m mockObjectStorage) PutObject(ctx context.Context, key string, contentType string, body []byte) (string, error) {
	return m.putObject(ctx, key, contentType, body)
}

func TestFileHandlerUploadStoresFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewFileHandler(mockObjectStorage{
		putObject: func(ctx context.Context, key string, contentType string, body []byte) (string, error) {
			if !strings.HasPrefix(key, "files/") {
				t.Fatalf("unexpected object key: %s", key)
			}
			if !strings.HasSuffix(key, ".jpg") {
				t.Fatalf("expected lower-case extension, got %s", key)
			}
			if contentType != "image/jpeg" {
				t.Fatalf("unexpected content type: %s", contentType)
			}
			if string(body) != "fake image" {
				t.Fatalf("unexpected file body: %s", string(body))
			}
			return "http://127.0.0.1:9000/livebid/" + key, nil
		},
	})

	body, contentType := multipartFileBody(t, "file", "cover.JPG", "image/jpeg", []byte("fake image"))
	w := performRequestWithContentType(handler.Upload, http.MethodPost, "/api/files/upload", body, contentType)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			URL       string `json:"url"`
			ObjectKey string `json:"object_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected code: %d", resp.Code)
	}
	if !strings.HasPrefix(resp.Data.URL, "http://127.0.0.1:9000/livebid/files/") {
		t.Fatalf("unexpected file url: %s", resp.Data.URL)
	}
	if resp.Data.ObjectKey == "" {
		t.Fatalf("expected object key")
	}
}

func TestFileHandlerUploadRequiresFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewFileHandler(mockObjectStorage{
		putObject: func(ctx context.Context, key string, contentType string, body []byte) (string, error) {
			t.Fatalf("storage should not be called")
			return "", nil
		},
	})
	body, contentType := multipartBody(t, map[string]string{"name": "cover"})
	w := performRequestWithContentType(handler.Upload, http.MethodPost, "/api/files/upload", body, contentType)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func multipartFileBody(t *testing.T, fieldName string, filename string, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="`+fieldName+`"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}
