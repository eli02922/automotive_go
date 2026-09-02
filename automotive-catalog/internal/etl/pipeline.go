package etl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/embuscado/automotive-catalog/internal/catalog/model"
	"github.com/embuscado/automotive-catalog/internal/catalog/repository"
	"github.com/embuscado/automotive-catalog/internal/kafka"
)

// Pipeline orchestrates concurrent ETL for product and fitment data ingestion.
type Pipeline struct {
	productRepo *repository.ProductRepository
	fitmentRepo *repository.FitmentRepository
	producer    *kafka.Producer
	log         *zap.Logger

	workers    int
	batchSize  int
}

func NewPipeline(
	productRepo *repository.ProductRepository,
	fitmentRepo *repository.FitmentRepository,
	producer *kafka.Producer,
	log *zap.Logger,
) *Pipeline {
	return &Pipeline{
		productRepo: productRepo,
		fitmentRepo: fitmentRepo,
		producer:    producer,
		log:         log,
		workers:     10,
		batchSize:   500,
	}
}

type ImportResult struct {
	Processed int64
	Failed    int64
	Duration  time.Duration
	Errors    []string
}

// ImportProducts ingests a large slice of products using a fan-out worker pool.
func (p *Pipeline) ImportProducts(ctx context.Context, products []*model.Product) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	jobs := make(chan []*model.Product, p.workers*2)
	results := make(chan batchResult, p.workers*2)

	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.productWorker(ctx, jobs, results)
		}()
	}

	go func() {
		defer close(jobs)
		for i := 0; i < len(products); i += p.batchSize {
			end := i + p.batchSize
			if end > len(products) {
				end = len(products)
			}
			select {
			case jobs <- products[i:end]:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		result.Processed += r.processed
		result.Failed += r.failed
		result.Errors = append(result.Errors, r.errors...)
	}

	result.Duration = time.Since(start)
	p.log.Info("product import complete",
		zap.Int64("processed", result.Processed),
		zap.Int64("failed", result.Failed),
		zap.Duration("duration", result.Duration),
	)
	return result, nil
}

type batchResult struct {
	processed int64
	failed    int64
	errors    []string
}

func (p *Pipeline) productWorker(ctx context.Context, jobs <-chan []*model.Product, results chan<- batchResult) {
	for batch := range jobs {
		r := batchResult{}
		validated := make([]*model.Product, 0, len(batch))

		for _, prod := range batch {
			if err := validateProduct(prod); err != nil {
				r.failed++
				r.errors = append(r.errors, fmt.Sprintf("product %s: %v", prod.PartNumber, err))
				continue
			}
			validated = append(validated, prod)
		}

		if len(validated) > 0 {
			n, err := p.productRepo.BulkUpsert(ctx, validated)
			if err != nil {
				r.failed += int64(len(validated))
				r.errors = append(r.errors, fmt.Sprintf("bulk upsert: %v", err))
			} else {
				r.processed += n
				for _, prod := range validated {
					if err := p.producer.PublishProductEvent(ctx, kafka.EventProductUpdated, prod); err != nil {
						p.log.Warn("publish failed after bulk upsert", zap.String("product_id", prod.ID), zap.Error(err))
					}
				}
			}
		}

		select {
		case results <- r:
		case <-ctx.Done():
			return
		}
	}
}

func validateProduct(p *model.Product) error {
	if p.PartNumber == "" {
		return fmt.Errorf("part_number required")
	}
	if p.Brand == "" {
		return fmt.Errorf("brand required")
	}
	if p.Name == "" {
		return fmt.Errorf("name required")
	}
	return nil
}
