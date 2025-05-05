package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url_shortener/internal/config"
	"url_shortener/internal/handler"
	"url_shortener/internal/middleware"
	"url_shortener/internal/model"
	"url_shortener/internal/repository"
	"url_shortener/internal/service"
	"url_shortener/pkg/cache"
	"url_shortener/pkg/database"
	shortener "url_shortener/pkg/shotener"

	"github.com/gin-gonic/gin"
)

// @title URL Shortener API
// @version 1.0
// @description A RESTful API for shortening URLs
// @BasePath /
func main() {
	// load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// set gin mode
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// initialize database connection
	db, err := database.NewPostgresDB(cfg.Database.GetDSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// run database migrations
	if err := db.Migrate(&model.URL{}, &model.URLVisit{}); err != nil {
		log.Fatalf("failed to run database migrations: %v", err)
	}

	// initialize redis cache
	redisClient, err := cache.NewRedisClient(
		cfg.Redis.GetRedisAddr(),
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}

	// initialize url shortener
	urlShortener := shortener.NewShortener(cfg.App.URLLength)

	// initialize repository
	urlRepo := repository.NewURLRepository(db.DB)

	// initialize service
	urlService := service.NewURLService(
		urlRepo,
		redisClient,
		urlShortener,
		cfg.App.ShortURLDomain,
	)

	// initialize handler
	urlHandler := handler.NewURLHandler(urlService)

	// create gin router
	router := gin.New()

	// apply middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())
	router.Use(middleware.Metrics())

	// register routes
	urlHandler.RegisterRoutes(router)

	// add health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// add prometheus metrics endpoint
	// router.GET("/metrics", gin.Wraph(promhttp.Handler()))

	// add swagger documentaion
	// router.GET("/swagger/*any", ginSwagger.WraphHandler(swaggerFiles.Handler))

	// start periodic tasks
	go startPeriodicTasks(urlService)

	// create http server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// start server in a goroutine
	go func() {
		log.Printf("starting server on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server: %v", err)
		}
	}()

	// wait for interrumpt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	// create a deadline to wait for current operation to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	// close database connection
	if err := db.Close(); err != nil {
		log.Fatalf("error closing database connection: %v", err)
	}

	// close redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("error closing redis connection: %v", err)
	}

	log.Println("server  exiting")
}

// start periodic task such as cleaning up expired urls
func startPeriodicTasks(urlService service.URLService) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		count, err := urlService.CleanupExpiredURLs(ctx)
		if err != nil {
			log.Printf("error cleaning up expired urls: %v", err)
		} else {
			log.Printf("cleaned up %d expired URLs", count)
		}

		cancel()
	}
}
