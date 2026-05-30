package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockLiveClient struct {
	createLiveRoom             func(ctx context.Context, in *livev1.CreateLiveRoomRequest, opts ...grpc.CallOption) (*livev1.CreateLiveRoomResponse, error)
	getLiveRoom                func(ctx context.Context, in *livev1.GetLiveRoomRequest, opts ...grpc.CallOption) (*livev1.GetLiveRoomResponse, error)
	listLiveRooms              func(ctx context.Context, in *livev1.ListLiveRoomsRequest, opts ...grpc.CallOption) (*livev1.ListLiveRoomsResponse, error)
	startLive                  func(ctx context.Context, in *livev1.StartLiveRequest, opts ...grpc.CallOption) (*livev1.StartLiveResponse, error)
	endLive                    func(ctx context.Context, in *livev1.EndLiveRequest, opts ...grpc.CallOption) (*livev1.EndLiveResponse, error)
	getLiveStreamInfo          func(ctx context.Context, in *livev1.GetLiveStreamInfoRequest, opts ...grpc.CallOption) (*livev1.GetLiveStreamInfoResponse, error)
	handleSRSPublishCallback   func(ctx context.Context, in *livev1.HandleSRSPublishCallbackRequest, opts ...grpc.CallOption) (*livev1.HandleSRSPublishCallbackResponse, error)
	handleSRSUnpublishCallback func(ctx context.Context, in *livev1.HandleSRSUnpublishCallbackRequest, opts ...grpc.CallOption) (*livev1.HandleSRSUnpublishCallbackResponse, error)
}

func (m mockLiveClient) CreateLiveRoom(ctx context.Context, in *livev1.CreateLiveRoomRequest, opts ...grpc.CallOption) (*livev1.CreateLiveRoomResponse, error) {
	return m.createLiveRoom(ctx, in, opts...)
}
func (m mockLiveClient) GetLiveRoom(ctx context.Context, in *livev1.GetLiveRoomRequest, opts ...grpc.CallOption) (*livev1.GetLiveRoomResponse, error) {
	return m.getLiveRoom(ctx, in, opts...)
}
func (m mockLiveClient) ListLiveRooms(ctx context.Context, in *livev1.ListLiveRoomsRequest, opts ...grpc.CallOption) (*livev1.ListLiveRoomsResponse, error) {
	return m.listLiveRooms(ctx, in, opts...)
}
func (m mockLiveClient) StartLive(ctx context.Context, in *livev1.StartLiveRequest, opts ...grpc.CallOption) (*livev1.StartLiveResponse, error) {
	return m.startLive(ctx, in, opts...)
}
func (m mockLiveClient) EndLive(ctx context.Context, in *livev1.EndLiveRequest, opts ...grpc.CallOption) (*livev1.EndLiveResponse, error) {
	return m.endLive(ctx, in, opts...)
}
func (m mockLiveClient) GetLiveStreamInfo(ctx context.Context, in *livev1.GetLiveStreamInfoRequest, opts ...grpc.CallOption) (*livev1.GetLiveStreamInfoResponse, error) {
	return m.getLiveStreamInfo(ctx, in, opts...)
}
func (m mockLiveClient) HandleSRSPublishCallback(ctx context.Context, in *livev1.HandleSRSPublishCallbackRequest, opts ...grpc.CallOption) (*livev1.HandleSRSPublishCallbackResponse, error) {
	return m.handleSRSPublishCallback(ctx, in, opts...)
}
func (m mockLiveClient) HandleSRSUnpublishCallback(ctx context.Context, in *livev1.HandleSRSUnpublishCallbackRequest, opts ...grpc.CallOption) (*livev1.HandleSRSUnpublishCallbackResponse, error) {
	return m.handleSRSUnpublishCallback(ctx, in, opts...)
}

func TestLiveHandlerCreateLiveRoomUsesJWTSubjectAsShopID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewLiveHandler(mockLiveClient{
		createLiveRoom: func(ctx context.Context, in *livev1.CreateLiveRoomRequest, opts ...grpc.CallOption) (*livev1.CreateLiveRoomResponse, error) {
			if in.GetShopId() != 1001 || in.GetTitle() != "直播间" {
				t.Fatalf("unexpected request: %#v", in)
			}
			return &livev1.CreateLiveRoomResponse{
				LiveRoom:          testLiveProtoRoom(),
				InitialStreamCode: "stream-code",
				RtmpPushUrl:       "rtmp://srs.test/live/live_2001",
				WebrtcPlayUrl:     "webrtc://srs.test/live/live_2001",
			}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"title":"直播间"}`)
	w := performLiveRequestWithShopID(handler.CreateLiveRoom, http.MethodPost, "/api/live/rooms", body, 1001)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			InitialStreamCode string `json:"initial_stream_code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.InitialStreamCode != "stream-code" {
		t.Fatalf("unexpected initial stream code: %q", resp.Data.InitialStreamCode)
	}
}

func TestLiveHandlerListLiveRooms(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewLiveHandler(mockLiveClient{
		listLiveRooms: func(ctx context.Context, in *livev1.ListLiveRoomsRequest, opts ...grpc.CallOption) (*livev1.ListLiveRoomsResponse, error) {
			if in.GetPage() != 2 || in.GetPageSize() != 10 {
				t.Fatalf("unexpected request: %#v", in)
			}
			return &livev1.ListLiveRoomsResponse{
				LiveRooms: []*livev1.LiveRoom{testLiveProtoRoom()},
				Total:     1,
			}, nil
		},
	}, time.Second)

	w := performLiveRequest(handler.ListLiveRooms, http.MethodGet, "/api/live/rooms?page=2&page_size=10", bytes.NewBuffer(nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestLiveHandlerHandleSRSPublishCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewLiveHandler(mockLiveClient{
		handleSRSPublishCallback: func(ctx context.Context, in *livev1.HandleSRSPublishCallbackRequest, opts ...grpc.CallOption) (*livev1.HandleSRSPublishCallbackResponse, error) {
			if in.GetStreamName() != "live_2001" || in.GetStreamCode() != "abc123" || in.GetClientId() != "cid" {
				t.Fatalf("unexpected request: %#v", in)
			}
			return &livev1.HandleSRSPublishCallbackResponse{LiveRoom: testLiveProtoRoom()}, nil
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"action":"on_publish","client_id":"cid","ip":"127.0.0.1","app":"live","stream":"live_2001","param":"?token=abc123"}`)
	w := performLiveRequest(handler.HandleSRSPublishCallback, http.MethodPost, "/api/srs/callbacks/publish", body)
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != "0" {
		t.Fatalf("unexpected response: status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestLiveHandlerHandleSRSPublishCallbackRejectsInvalidCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewLiveHandler(mockLiveClient{
		handleSRSPublishCallback: func(ctx context.Context, in *livev1.HandleSRSPublishCallbackRequest, opts ...grpc.CallOption) (*livev1.HandleSRSPublishCallbackResponse, error) {
			return nil, status.Error(codes.Unauthenticated, "invalid credential")
		},
	}, time.Second)

	body := bytes.NewBufferString(`{"stream":"live_2001","param":"?token=bad"}`)
	w := performLiveRequest(handler.HandleSRSPublishCallback, http.MethodPost, "/api/srs/callbacks/publish", body)
	if w.Code != http.StatusForbidden || strings.TrimSpace(w.Body.String()) != "1" {
		t.Fatalf("unexpected response: status=%d body=%q", w.Code, w.Body.String())
	}
}

func performLiveRequest(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	return performLiveRequestWithShopID(handlerFunc, method, path, body, 0)
}

func performLiveRequestWithShopID(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, shopID int64) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, body)
	c.Request.Header.Set("Content-Type", "application/json")
	if shopID > 0 {
		principal := identity.Principal{
			Kind: identity.KindShop,
			ID:   shopID,
		}
		c.Request = c.Request.WithContext(identity.NewContext(c.Request.Context(), principal))
	}
	handlerFunc(c)
	return w
}

func testLiveProtoRoom() *livev1.LiveRoom {
	now := timestamppb.New(time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC))
	return &livev1.LiveRoom{
		Id:                2001,
		ShopId:            1001,
		Title:             "直播间",
		Status:            livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING,
		MediaStreamStatus: livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}
