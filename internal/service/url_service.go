package service

import (
	"context"
	"fmt"
	"time"

	"url_shortener/internal/model"
	"url_shortener/internal/repository"
	"url_shortener/pkg/cache"
	shortener "url_shortener/pkg/shotener"
)

const (
	DefaultCacheTTL = 24 * time.Hour
	CacheKeyPrefix  = "url:"
)

// interface for URL service operations
type URLService interface {
	CreateShortURL(ctx context.Context, req model.CreateURLRequest, ip string) (*model.CreateURLResponse, error)
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
	RecordVisit(ctx context.Context, urlID uint, ip, userAgent, referer string) error
	GetURLStats(ctx context.Context, shortCode string) (*model.GetURLStatsResponse, error)
	CleanupExpiredURLs(ctx context.Context) (int64, error)
}

// implements URLService interface
type URLServiceImpl struct {
	urlRepo    repository.URLRepository
	cache      *cache.RedisClient
	shortener  *shortener.Shortener
	domainName string
}

// create a new URL service
func NewURLService(urlRepo repository.URLRepository, cache *cache.RedisClient, shortener *shortener.Shortener, domainName string) URLService {
	return &URLServiceImpl{
		urlRepo:    urlRepo,
		cache:      cache,
		shortener:  shortener,
		domainName: domainName,
	}
}

// create a new shortened url
func (s *URLServiceImpl) CreateShortURL(ctx context.Context, req model.CreateURLRequest, ip string) (*model.CreateURLResponse, error) {
	var shortCode string
	var err error

	if req.CustomCode != "" {
		if !s.shortener.IsValidCustomCode(req.CustomCode) {
			return nil, fmt.Errorf("invalid custom code")
		}

		_, err := s.urlRepo.FindByShortCode(ctx, req.CustomCode)
		if err != nil {
			return nil, fmt.Errorf("custom code already in use")
		}

		shortCode = req.CustomCode
	} else {
		for i := 0; i < 5; i++ {
			shortCode, err = s.shortener.Generate()
			if err != nil {
				return nil, fmt.Errorf("failed to generate short code: %w", err)
			}

			_, err := s.urlRepo.FindByShortCode(ctx, shortCode)
			if err != nil {
				break
			}

			if i == 4 {
				return nil, fmt.Errorf("failed to generate unique short code")
			}
		}
	}

	url := &model.URL{
		OriginalURL: req.OriginalURL,
		ShortCode:   shortCode,
		ExpiresAt:   req.ExpiresAt,
		CreatedByIP: ip,
	}

	if err := s.urlRepo.Create(ctx, url); err != nil {
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	cacheKey := fmt.Sprintf("%s%s", CacheKeyPrefix, shortCode)
	cacheTTL := DefaultCacheTTL
	if url.ExpiresAt != nil {
		expityTime := time.Until(*url.ExpiresAt)
		if expityTime > 0 && expityTime < DefaultCacheTTL {
			cacheTTL = expityTime
		}
	}

	if err := s.cache.SetWithTTL(ctx, cacheKey, url.OriginalURL, cacheTTL); err != nil {
		fmt.Printf("error caching url: %v\n", err)
	}

	shortURL := fmt.Sprintf("%s/%s", s.domainName, shortCode)

	response := &model.CreateURLResponse{
		ShortURL:    shortURL,
		OriginalURL: url.OriginalURL,
		ShortCode:   shortCode,
		ExpiresAt:   url.ExpiresAt,
		CreatedAt:   url.CreatedAt,
	}

	return response, nil
}

// GetOriginalURL retrieves the original URL from a short code
func (s *URLServiceImpl) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("%s%s", CacheKeyPrefix, shortCode)
	cachedURL, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		// URL found in cache
		return cachedURL, nil
	}

	// Not in cache, get from database
	url, err := s.urlRepo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}

	// Increment visit count in background
	go func() {
		bgCtx := context.Background()
		if err := s.urlRepo.IncrementVisitCount(bgCtx, url.ID); err != nil {
			fmt.Printf("Error incrementing visit count: %v\n", err)
		}
	}()

	// Cache the URL for future requests
	cacheTTL := DefaultCacheTTL
	if url.ExpiresAt != nil {
		expiryTime := time.Until(*url.ExpiresAt)
		if expiryTime > 0 && expiryTime < DefaultCacheTTL {
			cacheTTL = expiryTime
		}
	}

	if err := s.cache.SetWithTTL(ctx, cacheKey, url.OriginalURL, cacheTTL); err != nil {
		// Log error but continue; this is not critical
		fmt.Printf("Error caching URL: %v\n", err)
	}

	return url.OriginalURL, nil
}

// RecordVisit records a visit to a shortened URL
func (s *URLServiceImpl) RecordVisit(ctx context.Context, urlID uint, ip, userAgent, referer string) error {
	visit := &model.URLVisit{
		URLID:     urlID,
		IP:        ip,
		UserAgent: userAgent,
		Referer:   referer,
	}

	return s.urlRepo.CreateVisit(ctx, visit)
}

// GetURLStats gets statistics for a shortened URL
func (s *URLServiceImpl) GetURLStats(ctx context.Context, shortCode string) (*model.GetURLStatsResponse, error) {
	url, err := s.urlRepo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	shortURL := fmt.Sprintf("%s/%s", s.domainName, shortCode)

	stats := &model.GetURLStatsResponse{
		ShortURL:    shortURL,
		OriginalURL: url.OriginalURL,
		VisitCount:  url.VisitCount,
		CreatedAt:   url.CreatedAt,
		ExpiresAt:   url.ExpiresAt,
	}

	return stats, nil
}

// CleanupExpiredURLs removes expired URLs from the database
func (s *URLServiceImpl) CleanupExpiredURLs(ctx context.Context) (int64, error) {
	return s.urlRepo.DeleteExpired(ctx)
}
