package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultRPCTimeout = 3 * time.Second

type apiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
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
		respondError(c, http.StatusBadRequest, "请求路径中的ID无效，请传入大于0的数字ID")
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
		respondError(c, http.StatusBadRequest, "查询参数"+key+"格式无效，请传入数字")
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
		respondError(c, http.StatusBadRequest, "查询参数"+key+"格式无效，请传入数字")
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

func currentShopID(c *gin.Context) (int64, bool) {
	shopID, ok := identity.ShopID(c.Request.Context())
	if !ok {
		respondError(c, http.StatusUnauthorized, "未获取到商铺身份，请先登录")
		return 0, false
	}
	return shopID, true
}

func timestampString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}

func documentTimeString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format("2006-01-02 15:04:05")
}

func respondOK(c *gin.Context, data any) {
	respondOKWithMessage(c, "ok", data)
}

func respondOKWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, apiResponse{
		Code:    0,
		Message: message,
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
		respondError(c, http.StatusBadRequest, grpcMessageOrDefault(err, "请求参数无效，请检查后重试"))
	case codes.Unauthenticated:
		respondError(c, http.StatusUnauthorized, grpcMessageOrDefault(err, "未获取到有效身份，请先登录"))
	case codes.AlreadyExists:
		respondError(c, http.StatusConflict, grpcMessageOrDefault(err, "资源已存在，请更换后重试"))
	case codes.NotFound:
		respondError(c, http.StatusNotFound, grpcMessageOrDefault(err, "资源不存在或已被删除"))
	case codes.PermissionDenied:
		respondError(c, http.StatusForbidden, grpcMessageOrDefault(err, "无权执行该操作"))
	case codes.FailedPrecondition:
		respondError(c, http.StatusBadRequest, grpcMessageOrDefault(err, "当前状态不允许执行该操作"))
	case codes.DeadlineExceeded:
		respondError(c, http.StatusGatewayTimeout, "下游服务响应超时，请稍后重试")
	case codes.Unavailable:
		respondError(c, http.StatusBadGateway, "下游服务暂不可用，请稍后重试")
	default:
		if errors.Is(err, context.DeadlineExceeded) {
			respondError(c, http.StatusGatewayTimeout, "下游服务响应超时，请稍后重试")
			return
		}
		respondError(c, http.StatusInternalServerError, "服务内部错误，请稍后重试")
	}
}

func grpcMessageOrDefault(err error, fallback string) string {
	message := strings.TrimSpace(status.Convert(err).Message())
	if message == "" {
		return fallback
	}
	return message
}

func recordRequestError(c *gin.Context, err error) {
	if c == nil || err == nil {
		return
	}
	_ = c.Error(err)
}
