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
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/services/api-gateway/internal/config"
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

type mockUserLiveShopClient struct {
	get   func(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error)
	batch func(ctx context.Context, in *shopv1.BatchGetPublicShopsRequest, opts ...grpc.CallOption) (*shopv1.BatchGetPublicShopsResponse, error)
}

func (m mockUserLiveShopClient) GetShop(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error) {
	return m.get(ctx, in, opts...)
}

func (m mockUserLiveShopClient) BatchGetPublicShops(ctx context.Context, in *shopv1.BatchGetPublicShopsRequest, opts ...grpc.CallOption) (*shopv1.BatchGetPublicShopsResponse, error) {
	return m.batch(ctx, in, opts...)
}

type mockUserLiveGoodsClient struct {
	get   func(ctx context.Context, in *goodsv1.GetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.GetGoodsResponse, error)
	batch func(ctx context.Context, in *goodsv1.BatchGetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.BatchGetGoodsResponse, error)
}

func (m mockUserLiveGoodsClient) CreateGoods(ctx context.Context, in *goodsv1.CreateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.CreateGoodsResponse, error) {
	panic("not implemented")
}

func (m mockUserLiveGoodsClient) UpdateGoods(ctx context.Context, in *goodsv1.UpdateGoodsRequest, opts ...grpc.CallOption) (*goodsv1.UpdateGoodsResponse, error) {
	panic("not implemented")
}

func (m mockUserLiveGoodsClient) DeleteGoods(ctx context.Context, in *goodsv1.DeleteGoodsRequest, opts ...grpc.CallOption) (*goodsv1.DeleteGoodsResponse, error) {
	panic("not implemented")
}

func (m mockUserLiveGoodsClient) GetGoods(ctx context.Context, in *goodsv1.GetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.GetGoodsResponse, error) {
	return m.get(ctx, in, opts...)
}

func (m mockUserLiveGoodsClient) ListGoods(ctx context.Context, in *goodsv1.ListGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListGoodsResponse, error) {
	panic("not implemented")
}

func (m mockUserLiveGoodsClient) ListShopGoods(ctx context.Context, in *goodsv1.ListShopGoodsRequest, opts ...grpc.CallOption) (*goodsv1.ListShopGoodsResponse, error) {
	panic("not implemented")
}

func (m mockUserLiveGoodsClient) BatchGetGoods(ctx context.Context, in *goodsv1.BatchGetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.BatchGetGoodsResponse, error) {
	return m.batch(ctx, in, opts...)
}

func (m mockUserLiveGoodsClient) PutGoodsOnSale(ctx context.Context, in *goodsv1.PutGoodsOnSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOnSaleResponse, error) {
	panic("not implemented")
}

func (m mockUserLiveGoodsClient) PutGoodsOffSale(ctx context.Context, in *goodsv1.PutGoodsOffSaleRequest, opts ...grpc.CallOption) (*goodsv1.PutGoodsOffSaleResponse, error) {
	panic("not implemented")
}

type mockUserLiveAuctionClient struct {
	current      func(ctx context.Context, in *auctionv1.GetCurrentAuctionByRoomRequest, opts ...grpc.CallOption) (*auctionv1.GetCurrentAuctionByRoomResponse, error)
	batchCurrent func(ctx context.Context, in *auctionv1.BatchGetCurrentAuctionsByRoomRequest, opts ...grpc.CallOption) (*auctionv1.BatchGetCurrentAuctionsByRoomResponse, error)
	runtime      func(ctx context.Context, in *auctionv1.GetAuctionRuntimeRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionRuntimeResponse, error)
}

func (m mockUserLiveAuctionClient) GetCurrentAuctionByRoom(ctx context.Context, in *auctionv1.GetCurrentAuctionByRoomRequest, opts ...grpc.CallOption) (*auctionv1.GetCurrentAuctionByRoomResponse, error) {
	return m.current(ctx, in, opts...)
}

func (m mockUserLiveAuctionClient) BatchGetCurrentAuctionsByRoom(ctx context.Context, in *auctionv1.BatchGetCurrentAuctionsByRoomRequest, opts ...grpc.CallOption) (*auctionv1.BatchGetCurrentAuctionsByRoomResponse, error) {
	return m.batchCurrent(ctx, in, opts...)
}

func (m mockUserLiveAuctionClient) GetAuctionRuntime(ctx context.Context, in *auctionv1.GetAuctionRuntimeRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionRuntimeResponse, error) {
	return m.runtime(ctx, in, opts...)
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

func TestLiveHandlerGetLiveStreamInfoChecksShopOwnershipAtGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewLiveHandler(mockLiveClient{
		getLiveRoom: func(ctx context.Context, in *livev1.GetLiveRoomRequest, opts ...grpc.CallOption) (*livev1.GetLiveRoomResponse, error) {
			if in.GetId() != 2001 {
				t.Fatalf("unexpected get room request: %#v", in)
			}
			return &livev1.GetLiveRoomResponse{LiveRoom: testLiveProtoRoom()}, nil
		},
		getLiveStreamInfo: func(ctx context.Context, in *livev1.GetLiveStreamInfoRequest, opts ...grpc.CallOption) (*livev1.GetLiveStreamInfoResponse, error) {
			t.Fatal("GetLiveStreamInfo should not be called when shop ownership mismatches")
			return nil, nil
		},
	}, time.Second)

	w := performLiveRequestWithShopIDAndParams(handler.GetLiveStreamInfo, http.MethodGet, "/api/live/rooms/2001/stream", bytes.NewBuffer(nil), 9999, gin.Params{
		{Key: "id", Value: "2001"},
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLiveHandlerGetUserLivePreviewFiltersPushFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewLiveHandler(mockLiveClient{
		getLiveRoom: func(ctx context.Context, in *livev1.GetLiveRoomRequest, opts ...grpc.CallOption) (*livev1.GetLiveRoomResponse, error) {
			if in.GetId() != 2001 {
				t.Fatalf("unexpected get room request: %#v", in)
			}
			return &livev1.GetLiveRoomResponse{LiveRoom: testLiveProtoRoom()}, nil
		},
		getLiveStreamInfo: func(ctx context.Context, in *livev1.GetLiveStreamInfoRequest, opts ...grpc.CallOption) (*livev1.GetLiveStreamInfoResponse, error) {
			if in.GetId() != 2001 || in.GetShopId() != 0 {
				t.Fatalf("unexpected stream request: %#v", in)
			}
			return &livev1.GetLiveStreamInfoResponse{
				StreamInfo: &livev1.LiveStreamInfo{
					StreamName:        "live_2001",
					RtmpPushUrl:       "rtmp://srs.test/live/live_2001",
					WebrtcPlayUrl:     "webrtc://srs.test/live/live_2001",
					MediaStreamStatus: livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE,
				},
			}, nil
		},
	}, time.Second)

	w := performLiveRequestWithParams(handler.GetUserLivePreview, http.MethodGet, "/api/user/live/rooms/2001/preview", bytes.NewBuffer(nil), gin.Params{
		{Key: "id", Value: "2001"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "rtmp_push_url") || strings.Contains(body, "stream_name") {
		t.Fatalf("user preview leaked push fields: %s", body)
	}
	if !strings.Contains(body, `"webrtc_play_url":"webrtc://srs.test/live/live_2001"`) {
		t.Fatalf("expected webrtc play url in response: %s", body)
	}
}

func TestLiveHandlerGetUserLiveFeedAggregatesPublicFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewLiveHandlerWithAggregates(mockLiveClient{
		listLiveRooms: func(ctx context.Context, in *livev1.ListLiveRoomsRequest, opts ...grpc.CallOption) (*livev1.ListLiveRoomsResponse, error) {
			if in.GetPage() != 1 || in.GetPageSize() != 10 {
				t.Fatalf("unexpected list request: %#v", in)
			}
			return &livev1.ListLiveRoomsResponse{
				Total:     1,
				LiveRooms: []*livev1.LiveRoom{testLiveProtoRoom()},
			}, nil
		},
		getLiveStreamInfo: func(ctx context.Context, in *livev1.GetLiveStreamInfoRequest, opts ...grpc.CallOption) (*livev1.GetLiveStreamInfoResponse, error) {
			return &livev1.GetLiveStreamInfoResponse{StreamInfo: testLiveProtoStreamInfo()}, nil
		},
	}, mockUserLiveShopClient{
		batch: func(ctx context.Context, in *shopv1.BatchGetPublicShopsRequest, opts ...grpc.CallOption) (*shopv1.BatchGetPublicShopsResponse, error) {
			if len(in.GetIds()) != 1 || in.GetIds()[0] != 1001 {
				t.Fatalf("unexpected shop batch request: %#v", in)
			}
			return &shopv1.BatchGetPublicShopsResponse{List: []*shopv1.Shop{testShopProto()}}, nil
		},
	}, mockUserLiveGoodsClient{
		batch: func(ctx context.Context, in *goodsv1.BatchGetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.BatchGetGoodsResponse, error) {
			if len(in.GetIds()) != 1 || in.GetIds()[0] != 3001 {
				t.Fatalf("unexpected goods batch request: %#v", in)
			}
			return &goodsv1.BatchGetGoodsResponse{List: []*goodsv1.Goods{testGoodsProto()}}, nil
		},
	}, mockUserLiveAuctionClient{
		batchCurrent: func(ctx context.Context, in *auctionv1.BatchGetCurrentAuctionsByRoomRequest, opts ...grpc.CallOption) (*auctionv1.BatchGetCurrentAuctionsByRoomResponse, error) {
			if len(in.GetRoomIds()) != 1 || in.GetRoomIds()[0] != 2001 {
				t.Fatalf("unexpected auction batch request: %#v", in)
			}
			return &auctionv1.BatchGetCurrentAuctionsByRoomResponse{List: []*auctionv1.Auction{testAuctionProto()}}, nil
		},
	}, config.UserLiveConfig{}, time.Second)

	w := performLiveRequest(handler.GetUserLiveFeed, http.MethodGet, "/api/user/live/feed?page=1&page_size=10", bytes.NewBuffer(nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "rtmp_push_url") || strings.Contains(body, "stream_name") || strings.Contains(body, "phone") {
		t.Fatalf("feed leaked private fields: %s", body)
	}
	var resp struct {
		Data userLiveFeedResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data.List) != 1 || resp.Data.List[0].Shop.ShopName != "云上珠宝" || resp.Data.List[0].CurrentAuctionHint == nil {
		t.Fatalf("unexpected feed response: %#v", resp.Data)
	}
	if resp.Data.List[0].CurrentAuctionHint.GoodsTitle != "高冰翡翠手镯" || resp.Data.List[0].PreviewStream.PlayStatus != "available" {
		t.Fatalf("unexpected feed item: %#v", resp.Data.List[0])
	}
}

func TestLiveHandlerGetUserLiveEntryAggregatesSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	winner := int64(4001)
	serverTime := timestamppb.New(time.Date(2026, 5, 27, 10, 0, 10, 0, time.UTC))
	expireAt := timestamppb.New(time.Date(2026, 5, 27, 10, 0, 25, 0, time.UTC))

	handler := NewLiveHandlerWithAggregates(mockLiveClient{
		getLiveRoom: func(ctx context.Context, in *livev1.GetLiveRoomRequest, opts ...grpc.CallOption) (*livev1.GetLiveRoomResponse, error) {
			return &livev1.GetLiveRoomResponse{LiveRoom: testLiveProtoRoom()}, nil
		},
		getLiveStreamInfo: func(ctx context.Context, in *livev1.GetLiveStreamInfoRequest, opts ...grpc.CallOption) (*livev1.GetLiveStreamInfoResponse, error) {
			return &livev1.GetLiveStreamInfoResponse{StreamInfo: testLiveProtoStreamInfo()}, nil
		},
	}, mockUserLiveShopClient{
		get: func(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error) {
			return &shopv1.GetShopResponse{Shop: testShopProto()}, nil
		},
	}, mockUserLiveGoodsClient{
		get: func(ctx context.Context, in *goodsv1.GetGoodsRequest, opts ...grpc.CallOption) (*goodsv1.GetGoodsResponse, error) {
			return &goodsv1.GetGoodsResponse{Goods: testGoodsProto()}, nil
		},
	}, mockUserLiveAuctionClient{
		current: func(ctx context.Context, in *auctionv1.GetCurrentAuctionByRoomRequest, opts ...grpc.CallOption) (*auctionv1.GetCurrentAuctionByRoomResponse, error) {
			return &auctionv1.GetCurrentAuctionByRoomResponse{Auction: testAuctionProto()}, nil
		},
		runtime: func(ctx context.Context, in *auctionv1.GetAuctionRuntimeRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionRuntimeResponse, error) {
			return &auctionv1.GetAuctionRuntimeResponse{Runtime: &auctionv1.AuctionRuntime{
				AuctionId:    5001,
				Status:       1,
				CurrentPrice: 18000,
				BidCount:     8,
				WinnerUserId: &winner,
				ServerTime:   serverTime,
				ExpireAt:     expireAt,
				Version:      12,
			}}, nil
		},
	}, config.UserLiveConfig{WSURL: "wss://example.com/ws/live", HeartbeatIntervalSeconds: 20}, time.Second)

	w := performLiveRequestWithUserIDAndParams(handler.GetUserLiveEntry, http.MethodGet, "/api/user/live/rooms/2001/entry", bytes.NewBuffer(nil), 3001, gin.Params{
		{Key: "id", Value: "2001"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "rtmp_push_url") || strings.Contains(body, "stream_name") {
		t.Fatalf("entry leaked push fields: %s", body)
	}
	var resp struct {
		Data struct {
			Viewer         userLiveViewerResponse  `json:"viewer"`
			CurrentAuction userLiveAuctionResponse `json:"current_auction"`
			Goods          userLiveGoodsResponse   `json:"goods"`
			Runtime        userLiveRuntimeResponse `json:"runtime"`
			WS             userLiveWSResponse      `json:"ws"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Data.Viewer.IsLoggedIn || resp.Data.Viewer.UserID != 3001 || !resp.Data.Viewer.CanBid {
		t.Fatalf("unexpected viewer: %#v", resp.Data.Viewer)
	}
	if resp.Data.CurrentAuction.NextBidPrice != 19000 || resp.Data.Runtime.Version != 12 || resp.Data.Goods.Title != "高冰翡翠手镯" {
		t.Fatalf("unexpected entry snapshot: %#v", resp.Data)
	}
	if resp.Data.WS.URL != "wss://example.com/ws/live" || resp.Data.WS.HeartbeatIntervalSeconds != 20 {
		t.Fatalf("unexpected ws config: %#v", resp.Data.WS)
	}
}

func TestLiveHandlerGetUserLiveAuctionSnapshotReturnsNullWhenNoAuction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewLiveHandlerWithAggregates(mockLiveClient{
		getLiveRoom: func(ctx context.Context, in *livev1.GetLiveRoomRequest, opts ...grpc.CallOption) (*livev1.GetLiveRoomResponse, error) {
			return &livev1.GetLiveRoomResponse{LiveRoom: testLiveProtoRoom()}, nil
		},
	}, nil, nil, mockUserLiveAuctionClient{
		current: func(ctx context.Context, in *auctionv1.GetCurrentAuctionByRoomRequest, opts ...grpc.CallOption) (*auctionv1.GetCurrentAuctionByRoomResponse, error) {
			return nil, status.Error(codes.NotFound, "auction not found")
		},
	}, config.UserLiveConfig{}, time.Second)

	w := performLiveRequestWithParams(handler.GetUserLiveAuctionSnapshot, http.MethodGet, "/api/user/live/rooms/2001/auction-snapshot", bytes.NewBuffer(nil), gin.Params{
		{Key: "id", Value: "2001"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			CurrentAuction *userLiveAuctionResponse `json:"current_auction"`
			Goods          *userLiveGoodsResponse   `json:"goods"`
			Runtime        *userLiveRuntimeResponse `json:"runtime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.CurrentAuction != nil || resp.Data.Goods != nil || resp.Data.Runtime != nil {
		t.Fatalf("expected null auction snapshot, got %#v", resp.Data)
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

func performLiveRequestWithParams(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, params gin.Params) *httptest.ResponseRecorder {
	return performLiveRequestWithShopIDAndParams(handlerFunc, method, path, body, 0, params)
}

func performLiveRequestWithShopID(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, shopID int64) *httptest.ResponseRecorder {
	return performLiveRequestWithShopIDAndParams(handlerFunc, method, path, body, shopID, nil)
}

func performLiveRequestWithShopIDAndParams(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, shopID int64, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = params
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

func performLiveRequestWithUserIDAndParams(handlerFunc gin.HandlerFunc, method string, path string, body *bytes.Buffer, userID int64, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = params
	c.Request = httptest.NewRequest(method, path, body)
	c.Request.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		principal := identity.Principal{
			Kind: identity.KindUser,
			ID:   userID,
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
		Cover:             "https://example.com/live-cover.jpg",
		Description:       "天然翡翠专场",
		Status:            livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING,
		MediaStreamStatus: livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE,
		ActualStartTime:   now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func testLiveProtoStreamInfo() *livev1.LiveStreamInfo {
	return &livev1.LiveStreamInfo{
		StreamName:        "live_2001",
		RtmpPushUrl:       "rtmp://srs.test/live/live_2001",
		WebrtcPlayUrl:     "webrtc://srs.test/live/live_2001",
		MediaStreamStatus: livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE,
	}
}

func testShopProto() *shopv1.Shop {
	return &shopv1.Shop{
		Id:          1001,
		Username:    "merchant",
		ShopName:    "云上珠宝",
		Logo:        "https://example.com/shop-logo.jpg",
		Description: "专注天然翡翠",
		Phone:       "18800000000",
		Email:       "merchant@example.com",
	}
}

func testGoodsProto() *goodsv1.Goods {
	return &goodsv1.Goods{
		Id:          3001,
		ShopId:      1001,
		Title:       "高冰翡翠手镯",
		CoverUrl:    "https://example.com/goods-cover.jpg",
		Description: "高冰种翡翠手镯，直播专拍",
		Status:      1,
	}
}

func testAuctionProto() *auctionv1.Auction {
	startTime := timestamppb.New(time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC))
	endTime := timestamppb.New(time.Date(2026, 5, 27, 10, 30, 0, 0, time.UTC))
	winner := int64(4001)
	return &auctionv1.Auction{
		Id:           5001,
		GoodsId:      3001,
		ShopId:       1001,
		RoomId:       2001,
		StartPrice:   10000,
		BidIncrement: 1000,
		CurrentPrice: 18000,
		BidCount:     8,
		Status:       1,
		StartTime:    startTime,
		EndTime:      endTime,
		WinnerUserId: &winner,
		Version:      12,
	}
}
