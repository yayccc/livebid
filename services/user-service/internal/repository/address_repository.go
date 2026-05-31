package repository

import (
	"context"
	"errors"

	"github.com/yayccc/livebid/services/user-service/internal/model"
	"gorm.io/gorm"
)

var ErrAddressNotFound = errors.New("address not found")

type ListAddressFilter struct {
	UserID   int64
	Page     int
	PageSize int
}

type AddressRepository interface {
	Create(ctx context.Context, address *model.UserAddress) error
	FindByIDForUser(ctx context.Context, id int64, userID int64) (*model.UserAddress, error)
	Update(ctx context.Context, address *model.UserAddress) error
	Delete(ctx context.Context, id int64, userID int64) error
	List(ctx context.Context, filter ListAddressFilter) ([]*model.UserAddress, int64, error)
	SetDefault(ctx context.Context, id int64, userID int64) (*model.UserAddress, error)
}

type GormAddressRepository struct {
	db *gorm.DB
}

func NewGormAddressRepository(db *gorm.DB) *GormAddressRepository {
	return &GormAddressRepository{db: db}
}

func (r *GormAddressRepository) Create(ctx context.Context, address *model.UserAddress) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if address.IsDefault {
			if err := unsetDefaultAddresses(tx, address.UserID, address.ID); err != nil {
				return err
			}
		}
		return tx.Create(address).Error
	})
}

func (r *GormAddressRepository) FindByIDForUser(ctx context.Context, id int64, userID int64) (*model.UserAddress, error) {
	var address model.UserAddress
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND is_deleted = 0", id, userID).
		First(&address).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAddressNotFound
	}
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func (r *GormAddressRepository) Update(ctx context.Context, address *model.UserAddress) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if address.IsDefault {
			if err := unsetDefaultAddresses(tx, address.UserID, address.ID); err != nil {
				return err
			}
		}
		updates := map[string]any{
			"receiver_name":  address.ReceiverName,
			"receiver_phone": address.ReceiverPhone,
			"province":       address.Province,
			"city":           address.City,
			"district":       address.District,
			"detail_address": address.DetailAddress,
			"postal_code":    address.PostalCode,
			"is_default":     address.IsDefault,
		}
		result := tx.Model(&model.UserAddress{}).
			Where("id = ? AND user_id = ? AND is_deleted = 0", address.ID, address.UserID).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAddressNotFound
		}
		return nil
	})
}

func (r *GormAddressRepository) Delete(ctx context.Context, id int64, userID int64) error {
	tx := r.db.WithContext(ctx).
		Model(&model.UserAddress{}).
		Where("id = ? AND user_id = ? AND is_deleted = 0", id, userID).
		Update("is_deleted", true)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrAddressNotFound
	}
	return nil
}

func (r *GormAddressRepository) List(ctx context.Context, filter ListAddressFilter) ([]*model.UserAddress, int64, error) {
	page, pageSize := normalizePagination(filter.Page, filter.PageSize)
	query := r.db.WithContext(ctx).
		Model(&model.UserAddress{}).
		Where("user_id = ? AND is_deleted = 0", filter.UserID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*model.UserAddress
	err := query.
		Order("is_default DESC, created_at DESC, id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *GormAddressRepository) SetDefault(ctx context.Context, id int64, userID int64) (*model.UserAddress, error) {
	var address *model.UserAddress
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.UserAddress
		err := tx.Where("id = ? AND user_id = ? AND is_deleted = 0", id, userID).First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAddressNotFound
		}
		if err != nil {
			return err
		}
		if err := unsetDefaultAddresses(tx, userID, id); err != nil {
			return err
		}
		result := tx.Model(&model.UserAddress{}).
			Where("id = ? AND user_id = ? AND is_deleted = 0", id, userID).
			Update("is_default", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAddressNotFound
		}
		current.IsDefault = true
		address = &current
		return nil
	})
	if err != nil {
		return nil, err
	}
	return address, nil
}

func unsetDefaultAddresses(tx *gorm.DB, userID int64, keepID int64) error {
	return tx.Model(&model.UserAddress{}).
		Where("user_id = ? AND id <> ? AND is_deleted = 0", userID, keepID).
		Update("is_default", false).Error
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
