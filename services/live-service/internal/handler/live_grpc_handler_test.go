package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/services/live-service/internal/config"
	"github.com/yayccc/livebid/services/live-service/internal/model"
	"github.com/yayccc/livebid/services/live-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

type mockLiveRoomRepository struct {
	create           func(ctx context.Context, room *model.LiveRoom) error
	findByID         func(ctx context.Context, id int64) (*model.LiveRoom, error)
	findByStreamName func(ctx context.Context, streamName string) (*model.LiveRoom, error)
	list             func(ctx context.Context, filter repository.ListLiveRoomsFilter) ([]*model.LiveRoom, int64, error)
	update           func(ctx context.Context, room *model.LiveRoom) error
}

func (m *mockLiveRoomRepository) Create(ctx context.Context, room *model.LiveRoom) error {
	if m.create == nil {
		return nil
	}
	return m.create(ctx, room)
}

func (m *mockLiveRoomRepository) FindByID(ctx context.Context, id int64) (*model.LiveRoom, error) {
	if m.findByID == nil {
		return nil, repository.ErrLiveRoomNotFound
	}
	return m.findByID(ctx, id)
}

func (m *mockLiveRoomRepository) FindByStreamName(ctx context.Context, streamName string) (*model.LiveRoom, error) {
	if m.findByStreamName == nil {
		return nil, repository.ErrLiveRoomNotFound
	}
	return m.findByStreamName(ctx, streamName)
}

func (m *mockLiveRoomRepository) List(ctx context.Context, filter repository.ListLiveRoomsFilter) ([]*model.LiveRoom, int64, error) {
	if m.list == nil {
		return nil, 0, nil
	}
	return m.list(ctx, filter)
}

func (m *mockLiveRoomRepository) Update(ctx context.Context, room *model.LiveRoom) error {
	if m.update == nil {
		return nil
	}
	return m.update(ctx, room)
}

func TestLiveGRPCHandlerCreateLiveRoom(t *testing.T) {
	initTestLogger(t)

	var created *model.LiveRoom
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		create: func(ctx context.Context, room *model.LiveRoom) error {
			created = room
			return nil
		},
	})

	resp, err := handler.CreateLiveRoom(shopContext(1001), &livev1.CreateLiveRoomRequest{
		ShopId:      9999,
		Title:       "  晚场拍卖  ",
		Cover:       "  https://example.com/cover.jpg  ",
		Description: "  高货专场  ",
	})
	if err != nil {
		t.Fatalf("CreateLiveRoom returned error: %v", err)
	}
	if created == nil {
		t.Fatal("expected repository Create to be called")
	}
	if created.ShopID != 1001 || created.Title != "晚场拍卖" || created.Cover != "https://example.com/cover.jpg" || created.Description != "高货专场" {
		t.Fatalf("unexpected created room: %#v", created)
	}
	if created.Status != model.LiveRoomStatusNotLive || created.MediaStreamStatus != model.MediaStreamStatusOffline {
		t.Fatalf("unexpected initial statuses: %#v", created)
	}
	if created.StreamName == "" || created.StreamKeyHash == "" {
		t.Fatalf("expected stream fields to be generated: %#v", created)
	}
	if string(created.Extra) != "{}" {
		t.Fatalf("unexpected extra: %s", created.Extra)
	}
	if resp.GetLiveRoom().GetId() != created.ID {
		t.Fatalf("unexpected response room id: got %d want %d", resp.GetLiveRoom().GetId(), created.ID)
	}
	if resp.GetInitialStreamCode() == "" {
		t.Fatal("expected initial stream code to be returned once")
	}
	if resp.GetRtmpPushUrl() != "rtmp://srs.test/live/"+created.StreamName {
		t.Fatalf("unexpected rtmp push url: %s", resp.GetRtmpPushUrl())
	}
	if resp.GetWebrtcPlayUrl() != "webrtc://srs.test/live/"+created.StreamName {
		t.Fatalf("unexpected webrtc play url: %s", resp.GetWebrtcPlayUrl())
	}
}

func TestLiveGRPCHandlerCreateLiveRoomRejectsInvalidArgument(t *testing.T) {
	initTestLogger(t)

	handler := newTestLiveHandler(&mockLiveRoomRepository{
		create: func(ctx context.Context, room *model.LiveRoom) error {
			t.Fatal("Create should not be called")
			return nil
		},
	})

	_, err := handler.CreateLiveRoom(shopContext(1001), &livev1.CreateLiveRoomRequest{})
	assertCode(t, err, codes.InvalidArgument)
}

func TestLiveGRPCHandlerCreateLiveRoomRequiresShopIdentity(t *testing.T) {
	initTestLogger(t)

	handler := newTestLiveHandler(&mockLiveRoomRepository{
		create: func(ctx context.Context, room *model.LiveRoom) error {
			t.Fatal("Create should not be called")
			return nil
		},
	})

	_, err := handler.CreateLiveRoom(context.Background(), &livev1.CreateLiveRoomRequest{
		Title: "直播间",
	})
	assertCode(t, err, codes.Unauthenticated)
}

func TestLiveGRPCHandlerListLiveRoomsUsesLivingFilter(t *testing.T) {
	initTestLogger(t)

	livingRoom := testLiveRoom(2001, 1001, model.LiveRoomStatusLiving)
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		list: func(ctx context.Context, filter repository.ListLiveRoomsFilter) ([]*model.LiveRoom, int64, error) {
			if filter.ShopID != 0 {
				t.Fatalf("expected no shop filter, got %d", filter.ShopID)
			}
			if filter.Status != model.LiveRoomStatusLiving {
				t.Fatalf("expected living filter, got %d", filter.Status)
			}
			if filter.Limit != 10 || filter.Offset != 20 {
				t.Fatalf("unexpected pagination filter: %#v", filter)
			}
			return []*model.LiveRoom{livingRoom}, 21, nil
		},
	})

	resp, err := handler.ListLiveRooms(context.Background(), &livev1.ListLiveRoomsRequest{
		Page:     3,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListLiveRooms returned error: %v", err)
	}
	if resp.GetTotal() != 21 || len(resp.GetLiveRooms()) != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.GetLiveRooms()[0].GetStatus() != livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING {
		t.Fatalf("unexpected room status: %s", resp.GetLiveRooms()[0].GetStatus())
	}
}

func TestLiveGRPCHandlerStartLive(t *testing.T) {
	initTestLogger(t)

	room := testLiveRoom(2001, 1001, model.LiveRoomStatusNotLive)
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		findByID: func(ctx context.Context, id int64) (*model.LiveRoom, error) {
			if id != room.ID {
				t.Fatalf("unexpected find id: %d", id)
			}
			return room, nil
		},
		update: func(ctx context.Context, updated *model.LiveRoom) error {
			if updated.Status != model.LiveRoomStatusLiving {
				t.Fatalf("expected living status, got %d", updated.Status)
			}
			if updated.ActualStartTime == nil {
				t.Fatal("expected actual start time to be set")
			}
			if updated.ActualEndTime != nil {
				t.Fatal("expected actual end time to be cleared")
			}
			return nil
		},
	})

	resp, err := handler.StartLive(shopContext(room.ShopID), &livev1.StartLiveRequest{
		Id:     room.ID,
		ShopId: 9999,
	})
	if err != nil {
		t.Fatalf("StartLive returned error: %v", err)
	}
	if resp.GetLiveRoom().GetStatus() != livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING {
		t.Fatalf("unexpected response status: %s", resp.GetLiveRoom().GetStatus())
	}
	if resp.GetLiveRoom().GetActualStartTime() == nil {
		t.Fatal("expected actual start time in response")
	}
}

func TestLiveGRPCHandlerStartLiveRejectsOwnerMismatch(t *testing.T) {
	initTestLogger(t)

	room := testLiveRoom(2001, 1001, model.LiveRoomStatusNotLive)
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		findByID: func(ctx context.Context, id int64) (*model.LiveRoom, error) {
			return room, nil
		},
		update: func(ctx context.Context, room *model.LiveRoom) error {
			t.Fatal("Update should not be called")
			return nil
		},
	})

	_, err := handler.StartLive(shopContext(9999), &livev1.StartLiveRequest{
		Id:     room.ID,
		ShopId: room.ShopID,
	})
	assertCode(t, err, codes.PermissionDenied)
}

func TestLiveGRPCHandlerEndLive(t *testing.T) {
	initTestLogger(t)

	room := testLiveRoom(2001, 1001, model.LiveRoomStatusLiving)
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		findByID: func(ctx context.Context, id int64) (*model.LiveRoom, error) {
			return room, nil
		},
		update: func(ctx context.Context, updated *model.LiveRoom) error {
			if updated.Status != model.LiveRoomStatusNotLive {
				t.Fatalf("expected not_live status, got %d", updated.Status)
			}
			if updated.ActualEndTime == nil {
				t.Fatal("expected actual end time to be set")
			}
			return nil
		},
	})

	resp, err := handler.EndLive(shopContext(room.ShopID), &livev1.EndLiveRequest{
		Id:     room.ID,
		ShopId: 9999,
	})
	if err != nil {
		t.Fatalf("EndLive returned error: %v", err)
	}
	if resp.GetLiveRoom().GetStatus() != livev1.LiveRoomStatus_LIVE_ROOM_STATUS_NOT_LIVE {
		t.Fatalf("unexpected response status: %s", resp.GetLiveRoom().GetStatus())
	}
	if resp.GetLiveRoom().GetActualEndTime() == nil {
		t.Fatal("expected actual end time in response")
	}
}

func TestLiveGRPCHandlerValidateLiveRoomForAuctionRequiresLiving(t *testing.T) {
	initTestLogger(t)

	room := testLiveRoom(2001, 1001, model.LiveRoomStatusNotLive)
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		findByID: func(ctx context.Context, id int64) (*model.LiveRoom, error) {
			return room, nil
		},
	})

	_, err := handler.ValidateLiveRoomForAuction(shopContext(room.ShopID), &livev1.ValidateLiveRoomForAuctionRequest{
		Id:            room.ID,
		ShopId:        9999,
		RequireLiving: true,
	})
	assertCode(t, err, codes.FailedPrecondition)
}

func TestLiveGRPCHandlerGetLiveStreamInfo(t *testing.T) {
	initTestLogger(t)

	room := testLiveRoom(2001, 1001, model.LiveRoomStatusLiving)
	room.MediaStreamStatus = model.MediaStreamStatusOnline
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		findByID: func(ctx context.Context, id int64) (*model.LiveRoom, error) {
			return room, nil
		},
	})

	resp, err := handler.GetLiveStreamInfo(context.Background(), &livev1.GetLiveStreamInfoRequest{
		Id: room.ID,
	})
	if err != nil {
		t.Fatalf("GetLiveStreamInfo returned error: %v", err)
	}
	if resp.GetStreamInfo().GetStreamName() != room.StreamName {
		t.Fatalf("unexpected stream name: %s", resp.GetStreamInfo().GetStreamName())
	}
	if resp.GetStreamInfo().GetRtmpPushUrl() != "rtmp://srs.test/live/"+room.StreamName {
		t.Fatalf("unexpected rtmp push url: %s", resp.GetStreamInfo().GetRtmpPushUrl())
	}
	if resp.GetStreamInfo().GetMediaStreamStatus() != livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE {
		t.Fatalf("unexpected media status: %s", resp.GetStreamInfo().GetMediaStreamStatus())
	}
}

func TestLiveGRPCHandlerHandleSRSPublishCallback(t *testing.T) {
	initTestLogger(t)

	streamCode := "secret-code"
	room := testLiveRoom(2001, 1001, model.LiveRoomStatusLiving)
	room.StreamKeyHash = hashStreamCode(streamCode)
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		findByStreamName: func(ctx context.Context, streamName string) (*model.LiveRoom, error) {
			if streamName != room.StreamName {
				t.Fatalf("unexpected stream name: %s", streamName)
			}
			return room, nil
		},
		update: func(ctx context.Context, updated *model.LiveRoom) error {
			if updated.MediaStreamStatus != model.MediaStreamStatusOnline {
				t.Fatalf("expected media stream online, got %d", updated.MediaStreamStatus)
			}
			return nil
		},
	})

	resp, err := handler.HandleSRSPublishCallback(context.Background(), &livev1.HandleSRSPublishCallbackRequest{
		StreamName: room.StreamName,
		StreamCode: streamCode,
		ClientId:   "cid",
		Ip:         "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("HandleSRSPublishCallback returned error: %v", err)
	}
	if resp.GetLiveRoom().GetMediaStreamStatus() != livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE {
		t.Fatalf("unexpected media stream status: %s", resp.GetLiveRoom().GetMediaStreamStatus())
	}
}

func TestLiveGRPCHandlerHandleSRSPublishCallbackRejectsBadStreamCode(t *testing.T) {
	initTestLogger(t)

	room := testLiveRoom(2001, 1001, model.LiveRoomStatusLiving)
	room.StreamKeyHash = hashStreamCode("good-code")
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		findByStreamName: func(ctx context.Context, streamName string) (*model.LiveRoom, error) {
			return room, nil
		},
		update: func(ctx context.Context, room *model.LiveRoom) error {
			t.Fatal("Update should not be called")
			return nil
		},
	})

	_, err := handler.HandleSRSPublishCallback(context.Background(), &livev1.HandleSRSPublishCallbackRequest{
		StreamName: room.StreamName,
		StreamCode: "bad-code",
	})
	assertCode(t, err, codes.Unauthenticated)
}

func TestLiveGRPCHandlerHandleSRSUnpublishCallback(t *testing.T) {
	initTestLogger(t)

	room := testLiveRoom(2001, 1001, model.LiveRoomStatusLiving)
	room.MediaStreamStatus = model.MediaStreamStatusOnline
	handler := newTestLiveHandler(&mockLiveRoomRepository{
		findByStreamName: func(ctx context.Context, streamName string) (*model.LiveRoom, error) {
			return room, nil
		},
		update: func(ctx context.Context, updated *model.LiveRoom) error {
			if updated.MediaStreamStatus != model.MediaStreamStatusOffline {
				t.Fatalf("expected media stream offline, got %d", updated.MediaStreamStatus)
			}
			return nil
		},
	})

	resp, err := handler.HandleSRSUnpublishCallback(context.Background(), &livev1.HandleSRSUnpublishCallbackRequest{
		StreamName: room.StreamName,
		ClientId:   "cid",
		Ip:         "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("HandleSRSUnpublishCallback returned error: %v", err)
	}
	if resp.GetLiveRoom().GetMediaStreamStatus() != livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_OFFLINE {
		t.Fatalf("unexpected media stream status: %s", resp.GetLiveRoom().GetMediaStreamStatus())
	}
}

func newTestLiveHandler(repo repository.LiveRoomRepository) *LiveGRPCHandler {
	return NewLiveGRPCHandler(repo, idgen.New(9), config.SRSConfig{
		RTMPPushBaseURL:   "rtmp://srs.test/live",
		WebRTCPlayBaseURL: "webrtc://srs.test/live",
	})
}

func testLiveRoom(id int64, shopID int64, roomStatus model.LiveRoomStatus) *model.LiveRoom {
	now := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	return &model.LiveRoom{
		ID:                id,
		ShopID:            shopID,
		Title:             "直播间",
		Cover:             "https://example.com/cover.jpg",
		Description:       "直播简介",
		Status:            roomStatus,
		StreamName:        "live_2001",
		StreamKeyHash:     "hash",
		MediaStreamStatus: model.MediaStreamStatusOffline,
		ActualStartTime:   nil,
		ActualEndTime:     nil,
		Extra:             datatypes.JSON([]byte("{}")),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func shopContext(shopID int64) context.Context {
	return identity.NewContext(context.Background(), identity.Principal{
		Kind: identity.KindShop,
		ID:   shopID,
	})
}

func assertCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %s, got nil", want)
	}
	if got := status.Code(err); got != want {
		t.Fatalf("unexpected error code: got %s want %s, err=%v", got, want, err)
	}
}

func initTestLogger(t *testing.T) {
	t.Helper()
	_, err := logger.Init(logger.Config{
		ServiceName:   "live-service-test",
		Env:           "test",
		Level:         "error",
		Encoding:      logger.EncodingConsole,
		Outputs:       []string{"stdout"},
		ErrorOutputs:  []string{"stderr"},
		DisableCaller: true,
		DisableStack:  true,
	})
	if err != nil {
		t.Fatalf("init logger: %v", err)
	}
	t.Cleanup(func() {
		if syncErr := logger.Sync(); syncErr != nil && !errors.Is(syncErr, context.Canceled) {
			t.Fatalf("sync logger: %v", syncErr)
		}
	})
}
