package model

import (
	"time"

	"gorm.io/gorm"
)

// URL represents a shortened URL in the system
type URL struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	OriginalURL string         `gorm:"type:text;not null" json:"original_url"`
	ShortCode   string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"short_code"`
	VisitCount  int64          `gorm:"default:0" json:"visit_count"`
	ExpiresAt   *time.Time     `json:"expires_at"`
	CreatedByIP string         `gorm:"type:varchar(45)" json:"created_by_ip"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// URLVisit tracks each visit to a shortened URL
type URLVisit struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	URLID     uint      `gorm:"not null" json:"url_id"`
	URL       URL       `gorm:"foreignKey:URLID" json:"-"`
	IP        string    `gorm:"type:varchar(45)" json:"ip"`
	UserAgent string    `gorm:"type:text" json:"user_agent"`
	Referer   string    `gorm:"type:text" json:"referer"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateURLRequest represents the request body for creating a short URL
type CreateURLRequest struct {
	OriginalURL string     `json:"original_url" binding:"required,url"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CustomCode  string     `json:"custom_code"`
}

// CreateURLResponse represents the response body after creating a short URL
type CreateURLResponse struct {
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// GetURLStatsResponse represents the URL statistics response
type GetURLStatsResponse struct {
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	VisitCount  int64      `json:"visit_count"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}
