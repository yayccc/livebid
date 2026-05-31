package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/yayccc/livebid/services/user-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUsernameDuplicated = errors.New("username already exists")
	ErrPhoneDuplicated    = errors.New("phone already exists")
	ErrEmailDuplicated    = errors.New("email already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id int64) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.ensureUnique(ctx, user.ID, user.Username, user.Phone, user.Email); err != nil {
		return err
	}
	err := r.db.WithContext(ctx).Create(user).Error
	return mapUserMySQLError(err)
}

func (r *GormUserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = 0", id).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("username = ? AND is_deleted = 0", username).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) Update(ctx context.Context, user *model.User) error {
	if err := r.ensureUnique(ctx, user.ID, user.Username, user.Phone, user.Email); err != nil {
		return err
	}
	updates := map[string]any{
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
		"gender":   user.Gender,
		"birthday": user.Birthday,
		"phone":    user.Phone,
		"email":    user.Email,
	}
	tx := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND is_deleted = 0", user.ID).
		Updates(updates)
	if tx.Error != nil {
		return mapUserMySQLError(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *GormUserRepository) ensureUnique(ctx context.Context, currentID int64, username string, phone string, email string) error {
	if strings.TrimSpace(username) != "" {
		if err := r.ensureFieldUnique(ctx, currentID, "username", username, ErrUsernameDuplicated); err != nil {
			return err
		}
	}
	if strings.TrimSpace(phone) != "" {
		if err := r.ensureFieldUnique(ctx, currentID, "phone", phone, ErrPhoneDuplicated); err != nil {
			return err
		}
	}
	if strings.TrimSpace(email) != "" {
		if err := r.ensureFieldUnique(ctx, currentID, "email", email, ErrEmailDuplicated); err != nil {
			return err
		}
	}
	return nil
}

func (r *GormUserRepository) ensureFieldUnique(ctx context.Context, currentID int64, field string, value string, duplicated error) error {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where(field+" = ? AND id <> ? AND is_deleted = 0", value, currentID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return duplicated
	}
	return nil
}

func mapUserMySQLError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.Contains(message, "Duplicate entry") && strings.Contains(message, "uk_username") {
		return ErrUsernameDuplicated
	}
	return err
}
