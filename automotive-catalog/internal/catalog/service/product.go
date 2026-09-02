package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/embuscado/automotive-catalog/internal/cache"
	"github.com/embuscado/automotive-catalog/internal/catalog/model"
	"github.com/embuscado/automotive-catalog/internal/catalog/repository"
	"github.com/embuscado/automotive-catalog/internal/kafka"
	apperr "github.com/embuscado/automotive-catalog/pkg/errors"
)

type ProductService struct {
	repo     *repository.ProductRepository
	fitments *repository.FitmentRepository
	cache    *cache.Cache
	producer *kafka.Producer
	log      *zap.Logger
}

func NewProductService(
	repo *repository.ProductRepository,
	fitments *repository.FitmentRepository,
	cache *cache.Cache,
	producer *kafka.Producer,
	log *zap.Logger,
) *ProductService {
	return &ProductService{repo: repo, fitments: fitments, cache: cache, producer: producer, log: log}
}

func (s *ProductService) Create(ctx context.Context, req *model.CreateProductRequest) (*model.Product, error) {
	existing, err := s.repo.GetByPartNumber(ctx, req.PartNumber)
	if err == nil && existing != nil {
		return nil, apperr.Conflict(fmt.Sprintf("part number %s already exists", req.PartNumber))
	}

	p, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	if err := s.producer.PublishProductEvent(ctx, kafka.EventProductCreated, p); err != nil {
		s.log.Warn("failed to publish product created event", zap.String("product_id", p.ID), zap.Error(err))
	}

	s.log.Info("product created", zap.String("id", p.ID), zap.String("part_number", p.PartNumber))
	return p, nil
}

func (s *ProductService) GetByID(ctx context.Context, id string) (*model.Product, error) {
	cacheKey := fmt.Sprintf("product:%s", id)

	var p model.Product
	if err := s.cache.Get(ctx, cacheKey, &p); err == nil {
		return &p, nil
	}

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("product")
	}

	if err := s.cache.Set(ctx, cacheKey, product, 15*time.Minute); err != nil {
		s.log.Warn("cache set failed", zap.Error(err))
	}
	return product, nil
}

func (s *ProductService) List(ctx context.Context, f model.ProductFilter) (*model.ProductPage, error) {
	return s.repo.List(ctx, f)
}

func (s *ProductService) Update(ctx context.Context, id string, req *model.UpdateProductRequest) (*model.Product, error) {
	p, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	_ = s.cache.Delete(ctx, fmt.Sprintf("product:%s", id))

	if err := s.producer.PublishProductEvent(ctx, kafka.EventProductUpdated, p); err != nil {
		s.log.Warn("failed to publish product updated event", zap.String("product_id", id), zap.Error(err))
	}
	return p, nil
}

func (s *ProductService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Internal(err)
	}
	_ = s.cache.Delete(ctx, fmt.Sprintf("product:%s", id))
	return nil
}

func (s *ProductService) VehicleSearch(ctx context.Context, year int, make, vehicleModel string) ([]*model.Product, error) {
	cacheKey := fmt.Sprintf("fitment:%d:%s:%s", year, make, vehicleModel)

	var products []*model.Product
	if err := s.cache.Get(ctx, cacheKey, &products); err == nil {
		return products, nil
	}

	products, err := s.fitments.VehicleSearch(ctx, year, make, vehicleModel)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	if err := s.cache.Set(ctx, cacheKey, products, 30*time.Minute); err != nil {
		s.log.Warn("vehicle search cache set failed", zap.Error(err))
	}
	return products, nil
}

func (s *ProductService) UpsertFitment(ctx context.Context, req *model.UpsertFitmentRequest) (*model.Fitment, error) {
	if _, err := s.GetByID(ctx, req.ProductID); err != nil {
		return nil, apperr.NotFound("product")
	}

	f, err := s.fitments.Upsert(ctx, req)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	_ = s.cache.InvalidatePattern(ctx, fmt.Sprintf("fitment:*"))

	if err := s.producer.PublishFitmentEvent(ctx, kafka.EventFitmentUpserted, f); err != nil {
		s.log.Warn("failed to publish fitment event", zap.Error(err))
	}
	return f, nil
}

func isCacheMiss(err error) bool {
	return err == redis.Nil
}
