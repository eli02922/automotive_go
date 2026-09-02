package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/embuscado/automotive-catalog/internal/cache"
	"github.com/embuscado/automotive-catalog/internal/catalog/repository"
	"github.com/embuscado/automotive-catalog/internal/database"
	"github.com/embuscado/automotive-catalog/internal/etl"
	"github.com/embuscado/automotive-catalog/internal/kafka"
	"github.com/embuscado/automotive-catalog/pkg/config"
	"github.com/embuscado/automotive-catalog/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("load config: " + err.Error())
	}

	log := logger.Init(os.Getenv("ENV"))
	defer log.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pg, err := database.NewPostgres(ctx, cfg.Postgres)
	if err != nil {
		log.Fatal("connect postgres", zap.Error(err))
	}
	defer pg.Close()

	redisCache, err := cache.New(cfg.Redis)
	if err != nil {
		log.Fatal("connect redis", zap.Error(err))
	}
	defer redisCache.Close()

	producer := kafka.NewProducer(cfg.Kafka, log)
	defer producer.Close()

	productRepo := repository.NewProductRepository(pg.Pool)
	fitmentRepo := repository.NewFitmentRepository(pg.Pool)

	pipeline := etl.NewPipeline(productRepo, fitmentRepo, producer, log)
	_ = pipeline

	worker := etl.NewBackgroundWorker(log)

	worker.Register(etl.ScheduledJob{
		Name:     "inventory-sync",
		Interval: 5 * time.Minute,
		Run: func(ctx context.Context) error {
			log.Info("running inventory sync job")
			// TODO: pull from SQL Server, transform, push to Postgres + Kafka
			return nil
		},
	})

	worker.Register(etl.ScheduledJob{
		Name:     "fitment-validation",
		Interval: 1 * time.Hour,
		Run: func(ctx context.Context) error {
			log.Info("running fitment validation job")
			return nil
		},
	})

	worker.Register(etl.ScheduledJob{
		Name:     "cache-warmup",
		Interval: 30 * time.Minute,
		Run: func(ctx context.Context) error {
			log.Info("running cache warmup job")
			return nil
		},
	})

	go func() {
		if err := worker.Start(ctx); err != nil {
			log.Error("worker error", zap.Error(err))
		}
	}()

	log.Info("background worker started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("worker shutting down...")
	cancel()
	time.Sleep(3 * time.Second)
	log.Info("worker stopped")
}
