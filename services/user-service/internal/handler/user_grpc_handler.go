package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"time"

	userv1 "github.com/yayccc/livebid/gen/proto/user/v1"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/services/user-service/internal/model"
	"github.com/yayccc/livebid/services/user-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

const defaultTokenTTL = 7 * 24 * time.Hour

var (
	errInvalidArgument   = errors.New("invalid argument")
	errInvalidCredential = errors.New("invalid credential")
	errMissingSubject    = errors.New("missing auth subject")
	errUserUnavailable   = errors.New("user unavailable")
	errUserDuplicated    = errors.New("user duplicated")
)

type UserGRPCHandler struct {
	userv1.UnimplementedUserServiceServer
	users     repository.UserRepository
	addresses repository.AddressRepository
	ids       *idgen.Generator
	jwt       *auth.JWTManager
	tokenTTL  time.Duration
	now       func() time.Time
}

func NewUserGRPCHandler(users repository.UserRepository, addresses repository.AddressRepository, ids *idgen.Generator, jwt *auth.JWTManager, tokenTTL time.Duration) *UserGRPCHandler {
	if ids == nil {
		ids = idgen.New(0)
	}
	if tokenTTL <= 0 {
		tokenTTL = defaultTokenTTL
	}
	return &UserGRPCHandler{
		users:     users,
		addresses: addresses,
		ids:       ids,
		jwt:       jwt,
		tokenTTL:  tokenTTL,
		now:       time.Now,
	}
}

func (h *UserGRPCHandler) RegisterUser(ctx context.Context, req *userv1.RegisterUserRequest) (*userv1.RegisterUserResponse, error) {
	username := strings.TrimSpace(req.GetUsername())
	password := req.GetPassword()
	nickname := strings.TrimSpace(req.GetNickname())
	if username == "" || len(password) < 6 {
		return nil, toGRPCError(errInvalidArgument)
	}
	if nickname == "" {
		nickname = randomNickname()
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, toGRPCError(err)
	}

	user := &model.User{
		ID:           h.ids.Next(),
		Username:     username,
		PasswordHash: string(passwordHash),
		Nickname:     nickname,
		Avatar:       strings.TrimSpace(req.GetAvatar()),
		Phone:        strings.TrimSpace(req.GetPhone()),
		Email:        strings.TrimSpace(req.GetEmail()),
		Status:       model.UserStatusEnabled,
		Extra:        datatypes.JSON([]byte("{}")),
	}
	if err := h.users.Create(ctx, user); err != nil {
		if isDuplicatedUserError(err) {
			return nil, toGRPCError(errUserDuplicated)
		}
		return nil, toGRPCError(err)
	}
	return &userv1.RegisterUserResponse{User: toProtoUser(user)}, nil
}

func (h *UserGRPCHandler) LoginUser(ctx context.Context, req *userv1.LoginUserRequest) (*userv1.LoginUserResponse, error) {
	username := strings.TrimSpace(req.GetUsername())
	password := req.GetPassword()
	if username == "" || password == "" {
		return nil, toGRPCError(errInvalidArgument)
	}

	user, err := h.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, toGRPCError(errInvalidCredential)
		}
		return nil, toGRPCError(err)
	}
	if user.Status != model.UserStatusEnabled {
		return nil, toGRPCError(errUserUnavailable)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, toGRPCError(errInvalidCredential)
	}
	if h.jwt == nil {
		return nil, toGRPCError(auth.ErrMissingSecret)
	}

	expiresAt := h.now().Add(h.tokenTTL)
	token, err := h.jwt.Sign(auth.Claims{
		Subject:   strconv.FormatInt(user.ID, 10),
		ID:        strconv.FormatInt(h.ids.Next(), 10),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.LoginUserResponse{
		User:        toProtoUser(user),
		AccessToken: token,
		ExpiresIn:   int64(h.tokenTTL.Seconds()),
	}, nil
}

func (h *UserGRPCHandler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	user, err := h.users.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.GetUserResponse{User: toProtoUser(user)}, nil
}

func (h *UserGRPCHandler) UpdateCurrentUser(ctx context.Context, req *userv1.UpdateCurrentUserRequest) (*userv1.UpdateCurrentUserResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	if req.Nickname != nil {
		nickname := strings.TrimSpace(*req.Nickname)
		if nickname == "" {
			return nil, toGRPCError(errInvalidArgument)
		}
		user.Nickname = nickname
	}
	if req.Avatar != nil {
		user.Avatar = strings.TrimSpace(*req.Avatar)
	}
	if req.Gender != nil {
		gender, err := parseGender(req.GetGender())
		if err != nil {
			return nil, toGRPCError(err)
		}
		user.Gender = gender
	}
	if req.Birthday != nil {
		birthday, err := parseBirthday(strings.TrimSpace(*req.Birthday))
		if err != nil {
			return nil, toGRPCError(err)
		}
		user.Birthday = birthday
	}
	if req.Phone != nil {
		user.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Email != nil {
		user.Email = strings.TrimSpace(*req.Email)
	}

	if err := h.users.Update(ctx, user); err != nil {
		if isDuplicatedUserError(err) {
			return nil, toGRPCError(errUserDuplicated)
		}
		return nil, toGRPCError(err)
	}
	updated, err := h.users.FindByID(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.UpdateCurrentUserResponse{User: toProtoUser(updated)}, nil
}

func (h *UserGRPCHandler) CreateAddress(ctx context.Context, req *userv1.CreateAddressRequest) (*userv1.CreateAddressResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	isDefault, err := parseDefault(req.GetIsDefault())
	if err != nil {
		return nil, toGRPCError(err)
	}
	address := &model.UserAddress{
		ID:            h.ids.Next(),
		UserID:        userID,
		ReceiverName:  strings.TrimSpace(req.GetReceiverName()),
		ReceiverPhone: strings.TrimSpace(req.GetReceiverPhone()),
		Province:      strings.TrimSpace(req.GetProvince()),
		City:          strings.TrimSpace(req.GetCity()),
		District:      strings.TrimSpace(req.GetDistrict()),
		DetailAddress: strings.TrimSpace(req.GetDetailAddress()),
		PostalCode:    strings.TrimSpace(req.GetPostalCode()),
		IsDefault:     isDefault,
		Extra:         datatypes.JSON([]byte("{}")),
	}
	if !validAddress(address) {
		return nil, toGRPCError(errInvalidArgument)
	}
	if err := h.addresses.Create(ctx, address); err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.CreateAddressResponse{Address: toProtoAddress(address)}, nil
}

func (h *UserGRPCHandler) ListAddresses(ctx context.Context, req *userv1.ListAddressesRequest) (*userv1.ListAddressesResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	page, pageSize := normalizePagination(int(req.GetPage()), int(req.GetPageSize()))
	list, total, err := h.addresses.List(ctx, repository.ListAddressFilter{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.ListAddressesResponse{
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
		List:     toProtoAddressList(list),
	}, nil
}

func (h *UserGRPCHandler) GetAddress(ctx context.Context, req *userv1.GetAddressRequest) (*userv1.GetAddressResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	address, err := h.addresses.FindByIDForUser(ctx, req.GetId(), userID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.GetAddressResponse{Address: toProtoAddress(address)}, nil
}

func (h *UserGRPCHandler) UpdateAddress(ctx context.Context, req *userv1.UpdateAddressRequest) (*userv1.UpdateAddressResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	address, err := h.addresses.FindByIDForUser(ctx, req.GetId(), userID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	applyAddressUpdate(address, req)
	if req.IsDefault != nil {
		isDefault, err := parseDefault(req.GetIsDefault())
		if err != nil {
			return nil, toGRPCError(err)
		}
		address.IsDefault = isDefault
	}
	if !validAddress(address) {
		return nil, toGRPCError(errInvalidArgument)
	}
	if err := h.addresses.Update(ctx, address); err != nil {
		return nil, toGRPCError(err)
	}
	updated, err := h.addresses.FindByIDForUser(ctx, req.GetId(), userID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.UpdateAddressResponse{Address: toProtoAddress(updated)}, nil
}

func (h *UserGRPCHandler) DeleteAddress(ctx context.Context, req *userv1.DeleteAddressRequest) (*userv1.DeleteAddressResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	if err := h.addresses.Delete(ctx, req.GetId(), userID); err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.DeleteAddressResponse{}, nil
}

func (h *UserGRPCHandler) SetDefaultAddress(ctx context.Context, req *userv1.SetDefaultAddressRequest) (*userv1.SetDefaultAddressResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	address, err := h.addresses.SetDefault(ctx, req.GetId(), userID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.SetDefaultAddressResponse{Address: toProtoAddress(address)}, nil
}

func currentUserID(ctx context.Context) (int64, error) {
	if userID, ok := identity.UserID(ctx); ok {
		return userID, nil
	}
	return 0, errMissingSubject
}

func applyAddressUpdate(address *model.UserAddress, req *userv1.UpdateAddressRequest) {
	if req.ReceiverName != nil {
		address.ReceiverName = strings.TrimSpace(*req.ReceiverName)
	}
	if req.ReceiverPhone != nil {
		address.ReceiverPhone = strings.TrimSpace(*req.ReceiverPhone)
	}
	if req.Province != nil {
		address.Province = strings.TrimSpace(*req.Province)
	}
	if req.City != nil {
		address.City = strings.TrimSpace(*req.City)
	}
	if req.District != nil {
		address.District = strings.TrimSpace(*req.District)
	}
	if req.DetailAddress != nil {
		address.DetailAddress = strings.TrimSpace(*req.DetailAddress)
	}
	if req.PostalCode != nil {
		address.PostalCode = strings.TrimSpace(*req.PostalCode)
	}
}

func validAddress(address *model.UserAddress) bool {
	return address.UserID > 0 &&
		address.ReceiverName != "" &&
		address.ReceiverPhone != "" &&
		address.Province != "" &&
		address.City != "" &&
		address.District != "" &&
		address.DetailAddress != ""
}

func parseGender(value int32) (model.Gender, error) {
	switch model.Gender(value) {
	case model.GenderUnknown, model.GenderMale, model.GenderFemale:
		return model.Gender(value), nil
	default:
		return model.GenderUnknown, errInvalidArgument
	}
}

func parseDefault(value int32) (bool, error) {
	switch value {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errInvalidArgument
	}
}

func parseBirthday(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, errInvalidArgument
	}
	return &parsed, nil
}

func normalizePagination(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func randomNickname() string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	buf := make([]byte, 6)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "用户_000000"
		}
		buf[i] = alphabet[n.Int64()]
	}
	return "用户_" + string(buf)
}

func isDuplicatedUserError(err error) bool {
	return errors.Is(err, repository.ErrUsernameDuplicated) ||
		errors.Is(err, repository.ErrPhoneDuplicated) ||
		errors.Is(err, repository.ErrEmailDuplicated)
}

func toProtoUser(user *model.User) *userv1.User {
	if user == nil {
		return nil
	}
	return &userv1.User{
		Id:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Gender:    int32(user.Gender),
		Birthday:  formatBirthday(user.Birthday),
		Phone:     user.Phone,
		Email:     user.Email,
		Status:    int32(user.Status),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}

func toProtoAddress(address *model.UserAddress) *userv1.UserAddress {
	if address == nil {
		return nil
	}
	isDefault := int32(0)
	if address.IsDefault {
		isDefault = 1
	}
	return &userv1.UserAddress{
		Id:            address.ID,
		UserId:        address.UserID,
		ReceiverName:  address.ReceiverName,
		ReceiverPhone: address.ReceiverPhone,
		Province:      address.Province,
		City:          address.City,
		District:      address.District,
		DetailAddress: address.DetailAddress,
		PostalCode:    address.PostalCode,
		IsDefault:     isDefault,
		CreatedAt:     timestamppb.New(address.CreatedAt),
		UpdatedAt:     timestamppb.New(address.UpdatedAt),
	}
}

func toProtoAddressList(list []*model.UserAddress) []*userv1.UserAddress {
	result := make([]*userv1.UserAddress, 0, len(list))
	for _, address := range list {
		result = append(result, toProtoAddress(address))
	}
	return result
}

func formatBirthday(birthday *time.Time) string {
	if birthday == nil || birthday.IsZero() {
		return ""
	}
	return birthday.Format("2006-01-02")
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, errInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errInvalidCredential), errors.Is(err, errMissingSubject):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, errUserUnavailable):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, repository.ErrUserNotFound), errors.Is(err, repository.ErrAddressNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, errUserDuplicated):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
