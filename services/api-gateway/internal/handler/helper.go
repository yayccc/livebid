package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultRPCTimeout = 3 * time.Second

type apiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func normalizeRPCTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultRPCTimeout
	}
	return timeout
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		if err != nil {
			recordRequestError(c, err)
		}
		respondError(c, http.StatusBadRequest, "invalid request")
		return 0, false
	}
	return id, true
}

func parseOptionalInt64Query(c *gin.Context, key string) (*int64, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid request")
		return nil, false
	}
	return &value, true
}

func parseOptionalInt32Query(c *gin.Context, key string) (*int32, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid request")
		return nil, false
	}
	parsed := int32(value)
	return &parsed, true
}

func parsePositiveQueryInt(c *gin.Context, name string, fallback int) int {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func int32ValueOrZero(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func timestampString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}

func respondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, apiResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func respondError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, apiResponse{
		Code:    statusCode,
		Message: message,
	})
}

func respondGRPCError(c *gin.Context, err error) {
	recordRequestError(c, err)
	code := status.Code(err)
	switch code {
	case codes.InvalidArgument:
		respondError(c, http.StatusBadRequest, "invalid request")
	case codes.Unauthenticated:
		respondError(c, http.StatusUnauthorized, "invalid credential")
	case codes.AlreadyExists:
		respondError(c, http.StatusConflict, "shop already exists")
	case codes.NotFound:
		respondError(c, http.StatusNotFound, "resource not found")
	case codes.DeadlineExceeded:
		respondError(c, http.StatusGatewayTimeout, "upstream timeout")
	case codes.Unavailable:
		respondError(c, http.StatusBadGateway, "upstream unavailable")
	default:
		if errors.Is(err, context.DeadlineExceeded) {
			respondError(c, http.StatusGatewayTimeout, "upstream timeout")
			return
		}
		respondError(c, http.StatusInternalServerError, "internal error")
	}
}

func recordRequestError(c *gin.Context, err error) {
	if c == nil || err == nil {
		return
	}
	_ = c.Error(err)
}
