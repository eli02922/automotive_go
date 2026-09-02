package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/embuscado/automotive-catalog/internal/api/handlers"
	"github.com/embuscado/automotive-catalog/internal/api/middleware"
	"github.com/embuscado/automotive-catalog/internal/api/routes"
	"github.com/embuscado/automotive-catalog/internal/cache"
	"github.com/embuscado/automotive-catalog/internal/catalog/repository"
	"github.com/embuscado/automotive-catalog/internal/catalog/service"
	"github.com/embuscado/automotive-catalog/internal/database"
	"github.com/embuscado/automotive-catalog/internal/kafka"
	"github.com/embuscado/automotive-catalog/pkg/config"
	"github.com/embuscado/automotive-catalog/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	must(err, "load config")

	log := logger.Init(os.Getenv("ENV"))
	defer log.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Infrastructure ---
	pg, err := database.NewPostgres(ctx, cfg.Postgres)
	must(err, "connect postgres")
	defer pg.Close()

	redisCache, err := cache.New(cfg.Redis)
	must(err, "connect redis")
	defer redisCache.Close()

	producer := kafka.NewProducer(cfg.Kafka, log)
	defer producer.Close()

	consumer := kafka.NewConsumer(cfg.Kafka, log)
	defer consumer.Close()

	// --- Repositories & Services ---
	productRepo := repository.NewProductRepository(pg.Pool)
	fitmentRepo := repository.NewFitmentRepository(pg.Pool)

	productSvc := service.NewProductService(productRepo, fitmentRepo, redisCache, producer, log)

	// --- Handlers ---
	catalogH := handlers.NewCatalogHandler(productSvc)
	searchH := handlers.NewSearchHandler(productSvc)

	// --- Kafka consumers ---
	consumer.Start(ctx)

	// --- HTTP Server ---
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(log))

	routes.Register(r, catalogH, searchH, cfg.Server.JWTSecret)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("API server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced shutdown", zap.Error(err))
	}
	log.Info("server stopped")
}

func must(err error, msg string) {
	if err != nil {
		panic(msg + ": " + err.Error())
	}
}
