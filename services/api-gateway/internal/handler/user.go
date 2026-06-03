package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	userv1 "github.com/yayccc/livebid/gen/proto/user/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"google.golang.org/grpc"
)

type UserHandler struct {
	userClient userServiceClient
	rpcTimeout time.Duration
}

type userServiceClient interface {
	RegisterUser(ctx context.Context, in *userv1.RegisterUserRequest, opts ...grpc.CallOption) (*userv1.RegisterUserResponse, error)
	LoginUser(ctx context.Context, in *userv1.LoginUserRequest, opts ...grpc.CallOption) (*userv1.LoginUserResponse, error)
	GetUser(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.GetUserResponse, error)
	UpdateCurrentUser(ctx context.Context, in *userv1.UpdateCurrentUserRequest, opts ...grpc.CallOption) (*userv1.UpdateCurrentUserResponse, error)
	CreateAddress(ctx context.Context, in *userv1.CreateAddressRequest, opts ...grpc.CallOption) (*userv1.CreateAddressResponse, error)
	ListAddresses(ctx context.Context, in *userv1.ListAddressesRequest, opts ...grpc.CallOption) (*userv1.ListAddressesResponse, error)
	GetAddress(ctx context.Context, in *userv1.GetAddressRequest, opts ...grpc.CallOption) (*userv1.GetAddressResponse, error)
	UpdateAddress(ctx context.Context, in *userv1.UpdateAddressRequest, opts ...grpc.CallOption) (*userv1.UpdateAddressResponse, error)
	DeleteAddress(ctx context.Context, in *userv1.DeleteAddressRequest, opts ...grpc.CallOption) (*userv1.DeleteAddressResponse, error)
	SetDefaultAddress(ctx context.Context, in *userv1.SetDefaultAddressRequest, opts ...grpc.CallOption) (*userv1.SetDefaultAddressResponse, error)
}

func NewUserHandler(userClient userServiceClient, rpcTimeout time.Duration) *UserHandler {
	return &UserHandler{
		userClient: userClient,
		rpcTimeout: normalizeRPCTimeout(rpcTimeout),
	}
}

type registerUserRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
	Nickname string `json:"nickname" form:"nickname"`
	Avatar   string `json:"avatar" form:"avatar"`
	Phone    string `json:"phone" form:"phone"`
	Email    string `json:"email" form:"email"`
}

type loginUserRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}

type updateCurrentUserRequest struct {
	Nickname *string `json:"nickname" form:"nickname"`
	Avatar   *string `json:"avatar" form:"avatar"`
	Gender   *int32  `json:"gender" form:"gender"`
	Birthday *string `json:"birthday" form:"birthday"`
	Phone    *string `json:"phone" form:"phone"`
	Email    *string `json:"email" form:"email"`
}

type createAddressRequest struct {
	ReceiverName  string `json:"receiverName" form:"receiverName" binding:"required"`
	ReceiverPhone string `json:"receiverPhone" form:"receiverPhone" binding:"required"`
	Province      string `json:"province" form:"province" binding:"required"`
	City          string `json:"city" form:"city" binding:"required"`
	District      string `json:"district" form:"district" binding:"required"`
	DetailAddress string `json:"detailAddress" form:"detailAddress" binding:"required"`
	PostalCode    string `json:"postalCode" form:"postalCode"`
	IsDefault     int32  `json:"isDefault" form:"isDefault"`
}

type updateAddressRequest struct {
	ReceiverName  *string `json:"receiverName" form:"receiverName"`
	ReceiverPhone *string `json:"receiverPhone" form:"receiverPhone"`
	Province      *string `json:"province" form:"province"`
	City          *string `json:"city" form:"city"`
	District      *string `json:"district" form:"district"`
	DetailAddress *string `json:"detailAddress" form:"detailAddress"`
	PostalCode    *string `json:"postalCode" form:"postalCode"`
	IsDefault     *int32  `json:"isDefault" form:"isDefault"`
}

type userResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Gender    int32  `json:"gender"`
	Birthday  string `json:"birthday"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Status    int32  `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type loginUserResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in,omitempty"`
}

type addressResponse struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"userId,omitempty"`
	ReceiverName  string `json:"receiverName"`
	ReceiverPhone string `json:"receiverPhone"`
	Province      string `json:"province"`
	City          string `json:"city"`
	District      string `json:"district"`
	DetailAddress string `json:"detailAddress"`
	PostalCode    string `json:"postalCode"`
	IsDefault     int32  `json:"isDefault"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type addressListResponse struct {
	Total    int64             `json:"total"`
	Page     int32             `json:"page"`
	PageSize int32             `json:"page_size"`
	List     []addressResponse `json:"list"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req registerUserRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "注册参数无效：username 和 password 为必填项")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.RegisterUser(ctx, &userv1.RegisterUserRequest{
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Phone:    req.Phone,
		Email:    req.Email,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	respondOKWithMessage(c, "注册成功", gin.H{
		"userId": resp.GetUser().GetId(),
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req loginUserRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "登录参数无效：username 和 password 为必填项")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.LoginUser(ctx, &userv1.LoginUserRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	if resp.GetAccessToken() == "" {
		respondError(c, http.StatusBadGateway, "用户服务返回登录凭证为空，请稍后重试")
		return
	}

	respondOKWithMessage(c, "登录成功", loginUserResponse{
		Token:     resp.GetAccessToken(),
		ExpiresIn: resp.GetExpiresIn(),
	})
}

func (h *UserHandler) Get(c *gin.Context) {
	userID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: userID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOKWithMessage(c, "查询成功", toUserResponse(resp.GetUser()))
}

func (h *UserHandler) UpdateCurrent(c *gin.Context) {
	pathUserID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if pathUserID != userID {
		respondError(c, http.StatusForbidden, "只能修改当前登录用户自己的信息")
		return
	}

	var req updateCurrentUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "修改用户参数无效，请提交合法的 JSON 请求体")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.UpdateCurrentUser(ctx, &userv1.UpdateCurrentUserRequest{
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Gender:   req.Gender,
		Birthday: req.Birthday,
		Phone:    req.Phone,
		Email:    req.Email,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOKWithMessage(c, "修改成功", toUserResponse(resp.GetUser()))
}

func (h *UserHandler) CreateAddress(c *gin.Context) {
	if _, ok := currentUserID(c); !ok {
		return
	}

	var req createAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "新增地址参数无效，请检查必填字段")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.CreateAddress(ctx, &userv1.CreateAddressRequest{
		ReceiverName:  req.ReceiverName,
		ReceiverPhone: req.ReceiverPhone,
		Province:      req.Province,
		City:          req.City,
		District:      req.District,
		DetailAddress: req.DetailAddress,
		PostalCode:    req.PostalCode,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOKWithMessage(c, "新增成功", gin.H{
		"addressId": resp.GetAddress().GetId(),
		"address":   toAddressResponse(resp.GetAddress()),
	})
}

func (h *UserHandler) ListAddresses(c *gin.Context) {
	if _, ok := currentUserID(c); !ok {
		return
	}
	page, ok := parseOptionalInt32Query(c, "page")
	if !ok {
		return
	}
	pageSize, ok := parseOptionalInt32Query(c, "page_size")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.ListAddresses(ctx, &userv1.ListAddressesRequest{
		Page:     int32ValueOrZero(page),
		PageSize: int32ValueOrZero(pageSize),
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOKWithMessage(c, "查询成功", addressListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		List:     toAddressResponseList(resp.GetList()),
	})
}

func (h *UserHandler) GetAddress(c *gin.Context) {
	if _, ok := currentUserID(c); !ok {
		return
	}
	addressID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.GetAddress(ctx, &userv1.GetAddressRequest{Id: addressID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOKWithMessage(c, "查询成功", toAddressResponse(resp.GetAddress()))
}

func (h *UserHandler) UpdateAddress(c *gin.Context) {
	if _, ok := currentUserID(c); !ok {
		return
	}
	addressID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req updateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "修改地址参数无效，请提交合法的 JSON 请求体")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.UpdateAddress(ctx, &userv1.UpdateAddressRequest{
		Id:            addressID,
		ReceiverName:  req.ReceiverName,
		ReceiverPhone: req.ReceiverPhone,
		Province:      req.Province,
		City:          req.City,
		District:      req.District,
		DetailAddress: req.DetailAddress,
		PostalCode:    req.PostalCode,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOKWithMessage(c, "修改成功", toAddressResponse(resp.GetAddress()))
}

func (h *UserHandler) DeleteAddress(c *gin.Context) {
	if _, ok := currentUserID(c); !ok {
		return
	}
	addressID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	if _, err := h.userClient.DeleteAddress(ctx, &userv1.DeleteAddressRequest{Id: addressID}); err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOKWithMessage(c, "删除成功", gin.H{})
}

func (h *UserHandler) SetDefaultAddress(c *gin.Context) {
	if _, ok := currentUserID(c); !ok {
		return
	}
	addressID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.userClient.SetDefaultAddress(ctx, &userv1.SetDefaultAddressRequest{Id: addressID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOKWithMessage(c, "设置默认地址成功", toAddressResponse(resp.GetAddress()))
}

func currentUserID(c *gin.Context) (int64, bool) {
	userID, ok := identity.UserID(c.Request.Context())
	if !ok {
		respondError(c, http.StatusUnauthorized, "未获取到用户身份，请先登录")
		return 0, false
	}
	return userID, true
}

func toUserResponse(user *userv1.User) userResponse {
	if user == nil {
		return userResponse{}
	}
	return userResponse{
		ID:        user.GetId(),
		Username:  user.GetUsername(),
		Nickname:  user.GetNickname(),
		Avatar:    user.GetAvatar(),
		Gender:    user.GetGender(),
		Birthday:  user.GetBirthday(),
		Phone:     user.GetPhone(),
		Email:     user.GetEmail(),
		Status:    user.GetStatus(),
		CreatedAt: timestampString(user.GetCreatedAt()),
		UpdatedAt: timestampString(user.GetUpdatedAt()),
	}
}

func toAddressResponse(address *userv1.UserAddress) addressResponse {
	if address == nil {
		return addressResponse{}
	}
	return addressResponse{
		ID:            address.GetId(),
		UserID:        address.GetUserId(),
		ReceiverName:  address.GetReceiverName(),
		ReceiverPhone: address.GetReceiverPhone(),
		Province:      address.GetProvince(),
		City:          address.GetCity(),
		District:      address.GetDistrict(),
		DetailAddress: address.GetDetailAddress(),
		PostalCode:    address.GetPostalCode(),
		IsDefault:     address.GetIsDefault(),
		CreatedAt:     timestampString(address.GetCreatedAt()),
		UpdatedAt:     timestampString(address.GetUpdatedAt()),
	}
}

func toAddressResponseList(list []*userv1.UserAddress) []addressResponse {
	result := make([]addressResponse, 0, len(list))
	for _, address := range list {
		result = append(result, toAddressResponse(address))
	}
	return result
}
