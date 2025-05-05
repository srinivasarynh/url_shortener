package handler

import (
	"net/http"

	"url_shortener/internal/model"
	"url_shortener/internal/service"

	"github.com/gin-gonic/gin"
)

// handles http request relate to urls
type URLHandler struct {
	urlService service.URLService
}

// create a new url handler
func NewURLHandler(urlService service.URLService) *URLHandler {
	return &URLHandler{
		urlService: urlService,
	}
}

// RegisterRoutes registers the routes for the URL handler
func (h *URLHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/api/urls", h.CreateShortURL)
	router.GET("/api/urls/:shortCode/stats", h.GetURLStats)
	router.GET("/:shortCode", h.RedirectToOriginalURL)
}

// CreateShortURL handles the request to create a short URL
// @Summary Create a short URL
// @Description Creates a shortened URL from a long URL
// @Tags URLs
// @Accept json
// @Produce json
// @Param body body model.CreateURLRequest true "URL to shorten"
// @Success 201 {object} model.CreateURLResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/urls [post]
func (h *URLHandler) CreateShortURL(c *gin.Context) {
	var req model.CreateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get client IP address
	clientIP := c.ClientIP()

	// Create short URL
	resp, err := h.urlService.CreateShortURL(c.Request.Context(), req, clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// RedirectToOriginalURL redirects a short URL to its original URL
// @Summary Redirect to original URL
// @Description Redirects a short URL to its original URL
// @Tags URLs
// @Param shortCode path string true "Short URL code"
// @Success 302 {string} string "Redirect to original URL"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /{shortCode} [get]
func (h *URLHandler) RedirectToOriginalURL(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Get original URL
	originalURL, err := h.urlService.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found or expired"})
		return
	}

	// Get URL entity for visit recording
	url, err := h.urlService.GetURLStats(c.Request.Context(), shortCode)
	if err == nil {
		// Record visit in background (don't block the redirect)
		go h.urlService.RecordVisit(
			c.Request.Context(),
			uint(url.VisitCount), // Using visit count as URL ID for simplicity
			c.ClientIP(),
			c.Request.UserAgent(),
			c.Request.Referer(),
		)
	}

	// Redirect to original URL
	c.Redirect(http.StatusFound, originalURL)
}

// GetURLStats gets statistics for a short URL
// @Summary Get URL statistics
// @Description Gets statistics for a short URL
// @Tags URLs
// @Param shortCode path string true "Short URL code"
// @Produce json
// @Success 200 {object} model.GetURLStatsResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/urls/{shortCode}/stats [get]
func (h *URLHandler) GetURLStats(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Get URL stats
	stats, err := h.urlService.GetURLStats(c.Request.Context(), shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
