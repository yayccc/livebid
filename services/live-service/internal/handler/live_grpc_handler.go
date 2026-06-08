package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/services/live-service/internal/config"
	"github.com/yayccc/livebid/services/live-service/internal/model"
	"github.com/yayccc/livebid/services/live-service/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

var (
	errInvalidArgument   = errors.New("invalid argument")
	errInvalidCredential = errors.New("invalid credential")
	errForbidden         = errors.New("forbidden")
)

type LiveGRPCHandler struct {
	livev1.UnimplementedLiveServiceServer
	rooms repository.LiveRoomRepository
	ids   *idgen.Generator
	srs   config.SRSConfig
}

func NewLiveGRPCHandler(rooms repository.LiveRoomRepository, ids *idgen.Generator, srs config.SRSConfig) *LiveGRPCHandler {
	if ids == nil {
		ids = idgen.New(0)
	}
	return &LiveGRPCHandler{
		rooms: rooms,
		ids:   ids,
		srs:   srs,
	}
}

func (h *LiveGRPCHandler) CreateLiveRoom(ctx context.Context, req *livev1.CreateLiveRoomRequest) (*livev1.CreateLiveRoomResponse, error) {
	shopID, authErr := currentShopID(ctx)
	log := logger.FromContext(ctx).With(
		zap.String("method", "CreateLiveRoom"),
		zap.Int64("shop_id", shopID),
	)
	log.Info("create live room started")
	if authErr != nil {
		grpcErr := toGRPCError(authErr)
		log.Warn("create live room rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}

	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		grpcErr := toGRPCError(errInvalidArgument)
		log.Warn("create live room rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}

	streamCode, err := newStreamCode()
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Error("create live room failed to generate stream code", zap.Error(grpcErr))
		return nil, grpcErr
	}
	room := &model.LiveRoom{
		ID:                h.ids.Next(),
		ShopID:            shopID,
		Title:             title,
		Cover:             strings.TrimSpace(req.GetCover()),
		Description:       strings.TrimSpace(req.GetDescription()),
		Status:            model.LiveRoomStatusNotLive,
		StreamName:        fmt.Sprintf("live_%d", h.ids.Next()),
		StreamKeyHash:     hashStreamCode(streamCode),
		MediaStreamStatus: model.MediaStreamStatusOffline,
		Extra:             datatypes.JSON([]byte("{}")),
	}
	if err := h.rooms.Create(ctx, room); err != nil {
		grpcErr := toGRPCError(err)
		log.Error("create live room failed", zap.Int64("live_room_id", room.ID), zap.Error(grpcErr))
		return nil, grpcErr
	}
	log.Info("create live room succeeded", zap.Int64("live_room_id", room.ID), zap.String("stream_name", room.StreamName))
	return &livev1.CreateLiveRoomResponse{
		LiveRoom:          toProtoLiveRoom(room),
		InitialStreamCode: streamCode,
		RtmpPushUrl:       h.rtmpPushURL(room),
		WebrtcPlayUrl:     h.webrtcPlayURL(room),
	}, nil
}

func (h *LiveGRPCHandler) GetLiveRoom(ctx context.Context, req *livev1.GetLiveRoomRequest) (*livev1.GetLiveRoomResponse, error) {
	log := logger.FromContext(ctx).With(
		zap.String("method", "GetLiveRoom"),
		zap.Int64("live_room_id", req.GetId()),
	)
	log.Info("get live room started")

	if req.GetId() <= 0 {
		grpcErr := toGRPCError(errInvalidArgument)
		log.Warn("get live room rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}
	room, err := h.rooms.FindByID(ctx, req.GetId())
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Warn("get live room failed", zap.Error(grpcErr))
		return nil, grpcErr
	}
	log.Info("get live room succeeded", zap.Int64("shop_id", room.ShopID), zap.Int8("status", int8(room.Status)))
	return &livev1.GetLiveRoomResponse{LiveRoom: toProtoLiveRoom(room)}, nil
}

func (h *LiveGRPCHandler) ListLiveRooms(ctx context.Context, req *livev1.ListLiveRoomsRequest) (*livev1.ListLiveRoomsResponse, error) {
	statusFilter, err := listLiveRoomStatusFilter(req)
	if err != nil {
		return nil, toGRPCError(err)
	}
	log := logger.FromContext(ctx).With(
		zap.String("method", "ListLiveRooms"),
		zap.Int64("shop_id", req.GetShopId()),
		zap.Int8("status", int8(statusFilter)),
	)
	log.Info("list live rooms started")

	page := int(req.GetPage())
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	rooms, total, err := h.rooms.List(ctx, repository.ListLiveRoomsFilter{
		ShopID: req.GetShopId(),
		Status: statusFilter,
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	})
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Error("list live rooms failed", zap.Int("page", page), zap.Int("page_size", pageSize), zap.Error(grpcErr))
		return nil, grpcErr
	}

	resp := &livev1.ListLiveRoomsResponse{Total: total}
	for _, room := range rooms {
		resp.LiveRooms = append(resp.LiveRooms, toProtoLiveRoom(room))
	}
	log.Info("list live rooms succeeded", zap.Int("page", page), zap.Int("page_size", pageSize), zap.Int("result_count", len(rooms)), zap.Int64("total", total))
	return resp, nil
}

func (h *LiveGRPCHandler) StartLive(ctx context.Context, req *livev1.StartLiveRequest) (*livev1.StartLiveResponse, error) {
	shopID, authErr := currentShopID(ctx)
	log := logger.FromContext(ctx).With(
		zap.String("method", "StartLive"),
		zap.Int64("live_room_id", req.GetId()),
		zap.Int64("shop_id", shopID),
	)
	log.Info("start live started")
	if authErr != nil {
		grpcErr := toGRPCError(authErr)
		log.Warn("start live rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}

	room, err := h.findOwnedRoom(ctx, req.GetId(), shopID)
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Warn("start live rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}
	if room.Status == model.LiveRoomStatusLiving {
		log.Info("start live idempotent succeeded", zap.Int8("status", int8(room.Status)))
		return &livev1.StartLiveResponse{LiveRoom: toProtoLiveRoom(room)}, nil
	}
	if room.Status != model.LiveRoomStatusNotLive {
		grpcErr := toGRPCError(repository.ErrLiveRoomStateConflict)
		log.Warn("start live rejected by state", zap.Int8("status", int8(room.Status)), zap.Error(grpcErr))
		return nil, grpcErr
	}

	now := time.Now()
	room.Status = model.LiveRoomStatusLiving
	room.ActualStartTime = &now
	room.ActualEndTime = nil
	if err := h.rooms.Update(ctx, room); err != nil {
		grpcErr := toGRPCError(err)
		log.Error("start live failed", zap.Error(grpcErr))
		return nil, grpcErr
	}
	log.Info("start live succeeded", zap.Time("actual_start_time", now))
	return &livev1.StartLiveResponse{LiveRoom: toProtoLiveRoom(room)}, nil
}

func (h *LiveGRPCHandler) EndLive(ctx context.Context, req *livev1.EndLiveRequest) (*livev1.EndLiveResponse, error) {
	shopID, authErr := currentShopID(ctx)
	log := logger.FromContext(ctx).With(
		zap.String("method", "EndLive"),
		zap.Int64("live_room_id", req.GetId()),
		zap.Int64("shop_id", shopID),
	)
	log.Info("end live started")
	if authErr != nil {
		grpcErr := toGRPCError(authErr)
		log.Warn("end live rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}

	room, err := h.findOwnedRoom(ctx, req.GetId(), shopID)
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Warn("end live rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}
	if room.Status == model.LiveRoomStatusNotLive {
		log.Info("end live idempotent succeeded", zap.Int8("status", int8(room.Status)))
		return &livev1.EndLiveResponse{LiveRoom: toProtoLiveRoom(room)}, nil
	}
	if room.Status != model.LiveRoomStatusLiving {
		grpcErr := toGRPCError(repository.ErrLiveRoomStateConflict)
		log.Warn("end live rejected by state", zap.Int8("status", int8(room.Status)), zap.Error(grpcErr))
		return nil, grpcErr
	}

	now := time.Now()
	room.Status = model.LiveRoomStatusNotLive
	room.ActualEndTime = &now
	if err := h.rooms.Update(ctx, room); err != nil {
		grpcErr := toGRPCError(err)
		log.Error("end live failed", zap.Error(grpcErr))
		return nil, grpcErr
	}
	log.Info("end live succeeded", zap.Time("actual_end_time", now))
	return &livev1.EndLiveResponse{LiveRoom: toProtoLiveRoom(room)}, nil
}

func (h *LiveGRPCHandler) ValidateLiveRoomForAuction(ctx context.Context, req *livev1.ValidateLiveRoomForAuctionRequest) (*livev1.ValidateLiveRoomForAuctionResponse, error) {
	shopID, authErr := currentShopID(ctx)
	log := logger.FromContext(ctx).With(
		zap.String("method", "ValidateLiveRoomForAuction"),
		zap.Int64("live_room_id", req.GetId()),
		zap.Int64("shop_id", shopID),
		zap.Bool("require_living", req.GetRequireLiving()),
	)
	log.Info("validate live room for auction started")
	if authErr != nil {
		grpcErr := toGRPCError(authErr)
		log.Warn("validate live room for auction rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}

	room, err := h.findOwnedRoom(ctx, req.GetId(), shopID)
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Warn("validate live room for auction rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}
	if req.GetRequireLiving() && room.Status != model.LiveRoomStatusLiving {
		grpcErr := toGRPCError(repository.ErrLiveRoomStateConflict)
		log.Warn("validate live room for auction rejected by state", zap.Int8("status", int8(room.Status)), zap.Error(grpcErr))
		return nil, grpcErr
	}
	log.Info("validate live room for auction succeeded", zap.Int8("status", int8(room.Status)))
	return &livev1.ValidateLiveRoomForAuctionResponse{
		LiveRoom: toProtoLiveRoom(room),
	}, nil
}

func (h *LiveGRPCHandler) GetLiveStreamInfo(ctx context.Context, req *livev1.GetLiveStreamInfoRequest) (*livev1.GetLiveStreamInfoResponse, error) {
	log := logger.FromContext(ctx).With(
		zap.String("method", "GetLiveStreamInfo"),
		zap.Int64("live_room_id", req.GetId()),
	)
	log.Info("get live stream info started")

	if req.GetId() <= 0 {
		grpcErr := toGRPCError(errInvalidArgument)
		log.Warn("get live stream info rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}

	room, err := h.rooms.FindByID(ctx, req.GetId())
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Warn("get live stream info rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}
	log.Info("get live stream info succeeded", zap.String("stream_name", room.StreamName), zap.Int8("media_stream_status", int8(room.MediaStreamStatus)))
	return &livev1.GetLiveStreamInfoResponse{
		StreamInfo: h.streamInfo(room),
	}, nil
}

func (h *LiveGRPCHandler) HandleSRSPublishCallback(ctx context.Context, req *livev1.HandleSRSPublishCallbackRequest) (*livev1.HandleSRSPublishCallbackResponse, error) {
	streamName := strings.TrimSpace(req.GetStreamName())
	streamCode := strings.TrimSpace(req.GetStreamCode())
	log := logger.FromContext(ctx).With(
		zap.String("method", "HandleSRSPublishCallback"),
		zap.String("stream_name", streamName),
		zap.String("client_id", strings.TrimSpace(req.GetClientId())),
		zap.String("ip", strings.TrimSpace(req.GetIp())),
	)
	log.Info("handle srs publish callback started")

	if streamName == "" || streamCode == "" {
		grpcErr := toGRPCError(errInvalidArgument)
		log.Warn("handle srs publish callback rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}
	room, err := h.rooms.FindByStreamName(ctx, streamName)
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Warn("handle srs publish callback failed to find room", zap.Error(grpcErr))
		return nil, grpcErr
	}
	if room.StreamKeyHash != hashStreamCode(streamCode) {
		grpcErr := toGRPCError(errInvalidCredential)
		log.Warn("handle srs publish callback rejected by stream code", zap.Int64("live_room_id", room.ID), zap.Error(grpcErr))
		return nil, grpcErr
	}
	if room.Status != model.LiveRoomStatusLiving {
		grpcErr := toGRPCError(repository.ErrLiveRoomStateConflict)
		log.Warn("handle srs publish callback rejected by room status", zap.Int64("live_room_id", room.ID), zap.Int8("status", int8(room.Status)), zap.Error(grpcErr))
		return nil, grpcErr
	}

	room.MediaStreamStatus = model.MediaStreamStatusOnline
	if err := h.rooms.Update(ctx, room); err != nil {
		grpcErr := toGRPCError(err)
		log.Error("handle srs publish callback failed", zap.Int64("live_room_id", room.ID), zap.Error(grpcErr))
		return nil, grpcErr
	}
	log.Info("handle srs publish callback succeeded", zap.Int64("live_room_id", room.ID), zap.Int8("media_stream_status", int8(room.MediaStreamStatus)))
	return &livev1.HandleSRSPublishCallbackResponse{LiveRoom: toProtoLiveRoom(room)}, nil
}

func (h *LiveGRPCHandler) HandleSRSUnpublishCallback(ctx context.Context, req *livev1.HandleSRSUnpublishCallbackRequest) (*livev1.HandleSRSUnpublishCallbackResponse, error) {
	streamName := strings.TrimSpace(req.GetStreamName())
	log := logger.FromContext(ctx).With(
		zap.String("method", "HandleSRSUnpublishCallback"),
		zap.String("stream_name", streamName),
		zap.String("client_id", strings.TrimSpace(req.GetClientId())),
		zap.String("ip", strings.TrimSpace(req.GetIp())),
	)
	log.Info("handle srs unpublish callback started")

	if streamName == "" {
		grpcErr := toGRPCError(errInvalidArgument)
		log.Warn("handle srs unpublish callback rejected", zap.Error(grpcErr))
		return nil, grpcErr
	}
	room, err := h.rooms.FindByStreamName(ctx, streamName)
	if err != nil {
		grpcErr := toGRPCError(err)
		log.Warn("handle srs unpublish callback failed to find room", zap.Error(grpcErr))
		return nil, grpcErr
	}

	room.MediaStreamStatus = model.MediaStreamStatusOffline
	if err := h.rooms.Update(ctx, room); err != nil {
		grpcErr := toGRPCError(err)
		log.Error("handle srs unpublish callback failed", zap.Int64("live_room_id", room.ID), zap.Error(grpcErr))
		return nil, grpcErr
	}
	log.Info("handle srs unpublish callback succeeded", zap.Int64("live_room_id", room.ID), zap.Int8("media_stream_status", int8(room.MediaStreamStatus)))
	return &livev1.HandleSRSUnpublishCallbackResponse{LiveRoom: toProtoLiveRoom(room)}, nil
}

func (h *LiveGRPCHandler) findOwnedRoom(ctx context.Context, roomID int64, shopID int64) (*model.LiveRoom, error) {
	if roomID <= 0 || shopID <= 0 {
		return nil, errInvalidArgument
	}
	room, err := h.rooms.FindByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room.ShopID != shopID {
		return nil, repository.ErrLiveRoomOwnerMismatch
	}
	return room, nil
}

func currentShopID(ctx context.Context) (int64, error) {
	shopID, ok := identity.ShopID(ctx)
	if !ok {
		return 0, errInvalidCredential
	}
	return shopID, nil
}

func listLiveRoomStatusFilter(req *livev1.ListLiveRoomsRequest) (model.LiveRoomStatus, error) {
	if req.Status == nil {
		if req.GetShopId() > 0 {
			return 0, nil
		}
		return model.LiveRoomStatusLiving, nil
	}
	statusFilter := fromProtoLiveRoomStatus(req.GetStatus())
	if statusFilter == 0 {
		return 0, errInvalidArgument
	}
	return statusFilter, nil
}

func (h *LiveGRPCHandler) streamInfo(room *model.LiveRoom) *livev1.LiveStreamInfo {
	if room == nil {
		return nil
	}
	return &livev1.LiveStreamInfo{
		StreamName:        room.StreamName,
		RtmpPushUrl:       h.rtmpPushURL(room),
		WebrtcPlayUrl:     h.webrtcPlayURL(room),
		MediaStreamStatus: toProtoMediaStreamStatus(room.MediaStreamStatus),
	}
}

func (h *LiveGRPCHandler) rtmpPushURL(room *model.LiveRoom) string {
	if room == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s", strings.TrimRight(h.srs.RTMPPushBaseURL, "/"), room.StreamName)
}

func (h *LiveGRPCHandler) webrtcPlayURL(room *model.LiveRoom) string {
	if room == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s", strings.TrimRight(h.srs.WebRTCPlayBaseURL, "/"), room.StreamName)
}

func newStreamCode() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashStreamCode(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func toProtoLiveRoom(room *model.LiveRoom) *livev1.LiveRoom {
	if room == nil {
		return nil
	}
	return &livev1.LiveRoom{
		Id:                room.ID,
		ShopId:            room.ShopID,
		Title:             room.Title,
		Cover:             room.Cover,
		Description:       room.Description,
		Status:            toProtoLiveRoomStatus(room.Status),
		MediaStreamStatus: toProtoMediaStreamStatus(room.MediaStreamStatus),
		ActualStartTime:   optionalTimestamp(room.ActualStartTime),
		ActualEndTime:     optionalTimestamp(room.ActualEndTime),
		CreatedAt:         timestamppb.New(room.CreatedAt),
		UpdatedAt:         timestamppb.New(room.UpdatedAt),
	}
}

func toProtoLiveRoomStatus(value model.LiveRoomStatus) livev1.LiveRoomStatus {
	switch value {
	case model.LiveRoomStatusNotLive:
		return livev1.LiveRoomStatus_LIVE_ROOM_STATUS_NOT_LIVE
	case model.LiveRoomStatusLiving:
		return livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING
	default:
		return livev1.LiveRoomStatus_LIVE_ROOM_STATUS_UNSPECIFIED
	}
}

func toProtoMediaStreamStatus(value model.MediaStreamStatus) livev1.MediaStreamStatus {
	switch value {
	case model.MediaStreamStatusOffline:
		return livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_OFFLINE
	case model.MediaStreamStatusOnline:
		return livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE
	default:
		return livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_UNSPECIFIED
	}
}

func fromProtoLiveRoomStatus(value livev1.LiveRoomStatus) model.LiveRoomStatus {
	switch value {
	case livev1.LiveRoomStatus_LIVE_ROOM_STATUS_NOT_LIVE:
		return model.LiveRoomStatusNotLive
	case livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING:
		return model.LiveRoomStatusLiving
	default:
		return 0
	}
}

func optionalTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, errInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errInvalidCredential):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, repository.ErrLiveRoomNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, repository.ErrLiveRoomOwnerMismatch), errors.Is(err, errForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, repository.ErrStreamNameDuplicated):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, repository.ErrLiveRoomStateConflict):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
