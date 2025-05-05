package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"url_shortener/internal/model"

	"gorm.io/gorm"
)

// interface for URL repository operations
type URLRepository interface {
	Create(ctx context.Context, url *model.URL) error
	FindByShortCode(ctx context.Context, shortCode string) (*model.URL, error)
	IncrementVisitCount(ctx context.Context, id uint) error
	CreateVisit(ctx context.Context, visit *model.URLVisit) error
	FindAllByUser(ctx context.Context, userID uint, limit, offset int) ([]model.URL, int64, error)
	DeleteExpired(ctx context.Context) (int64, error)
}

// url repository implements
type URLRepositoryImpl struct {
	db *gorm.DB
}

// create a new URL repository
func NewURLRepository(db *gorm.DB) URLRepository {
	return &URLRepositoryImpl{
		db: db,
	}
}

// create a new url in database
func (r *URLRepositoryImpl) Create(ctx context.Context, url *model.URL) error {
	return r.db.WithContext(ctx).Create(url).Error
}

// find url by short code
func (r *URLRepositoryImpl) FindByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	var url model.URL
	err := r.db.WithContext(ctx).Where("short_code = ?", shortCode).First(&url).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("URL with short code %s not found", shortCode)
		}
		return nil, fmt.Errorf("error finding URL: %w", err)
	}

	if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("URL has expired")
	}

	return &url, nil
}

// increment visit count for the url
func (r *URLRepositoryImpl) IncrementVisitCount(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Model(&model.URL{}).Where("id = ?", id).UpdateColumn("visit_count", gorm.Expr("visit_count + ?", 1)).Error
}

// create a new url visit record
func (r *URLRepositoryImpl) CreateVisit(ctx context.Context, visit *model.URLVisit) error {
	return r.db.WithContext(ctx).Create(visit).Error
}

// find all url created by a specific user
func (r *URLRepositoryImpl) FindAllByUser(ctx context.Context, userID uint, limit, offset int) ([]model.URL, int64, error) {
	var urls []model.URL
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.URL{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("error counting urls: %w", err)
	}

	if err := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&urls).Error; err != nil {
		return nil, 0, fmt.Errorf("error finding urls: %w", err)
	}

	return urls, total, nil
}

// delete all expired urls
func (r *URLRepositoryImpl) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at < ? AND expires_at IS NOT NULL", time.Now()).Delete(&model.URL{})

	return result.RowsAffected, result.Error
}
