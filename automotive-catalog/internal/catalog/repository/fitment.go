package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/embuscado/automotive-catalog/internal/catalog/model"
)

type FitmentRepository struct {
	pool *pgxpool.Pool
}

func NewFitmentRepository(pool *pgxpool.Pool) *FitmentRepository {
	return &FitmentRepository{pool: pool}
}

func (r *FitmentRepository) Upsert(ctx context.Context, req *model.UpsertFitmentRequest) (*model.Fitment, error) {
	qualJSON, err := json.Marshal(req.Qualifiers)
	if err != nil {
		return nil, fmt.Errorf("fitment repo: marshal qualifiers: %w", err)
	}

	f := &model.Fitment{
		ID:         uuid.New().String(),
		ProductID:  req.ProductID,
		Year:       req.Year,
		Make:       req.Make,
		Model:      req.Model,
		SubModel:   req.SubModel,
		Engine:     req.Engine,
		Notes:      req.Notes,
		Qualifiers: req.Qualifiers,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	query := `
		INSERT INTO fitments (id, product_id, year, make, model, sub_model, engine, notes, qualifiers, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (product_id, year, make, model, sub_model, engine) DO UPDATE SET
			notes = EXCLUDED.notes,
			qualifiers = EXCLUDED.qualifiers,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at`

	err = r.pool.QueryRow(ctx, query,
		f.ID, f.ProductID, f.Year, f.Make, f.Model, f.SubModel, f.Engine, f.Notes, qualJSON, f.CreatedAt, f.UpdatedAt,
	).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("fitment repo: upsert: %w", err)
	}
	return f, nil
}

func (r *FitmentRepository) ListByProduct(ctx context.Context, productID string, page, pageSize int) (*model.FitmentPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM fitments WHERE product_id = $1", productID).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, product_id, year, make, model, sub_model, engine, notes, qualifiers, created_at, updated_at
		FROM fitments WHERE product_id = $1 ORDER BY year DESC, make, model LIMIT $2 OFFSET $3`,
		productID, pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("fitment repo: list by product: %w", err)
	}
	defer rows.Close()

	var items []*model.Fitment
	for rows.Next() {
		var f model.Fitment
		var qualJSON []byte
		if err := rows.Scan(&f.ID, &f.ProductID, &f.Year, &f.Make, &f.Model, &f.SubModel, &f.Engine, &f.Notes, &qualJSON, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(qualJSON, &f.Qualifiers)
		items = append(items, &f)
	}

	return &model.FitmentPage{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	}, nil
}

// VehicleSearch finds all products that fit a given vehicle.
func (r *FitmentRepository) VehicleSearch(ctx context.Context, year int, make, vehicleModel string) ([]*model.Product, error) {
	query := `
		SELECT DISTINCT p.id, p.part_number, p.brand, p.name, p.description,
			p.category_id, p.status, p.weight, p.dimensions, p.attributes, p.msrp, p.cost, p.created_at, p.updated_at
		FROM fitments f
		JOIN products p ON p.id = f.product_id
		WHERE f.year = $1 AND LOWER(f.make) = LOWER($2) AND LOWER(f.model) = LOWER($3)
			AND p.deleted_at IS NULL AND p.status = 'active'
		ORDER BY p.brand, p.name`

	rows, err := r.pool.Query(ctx, query, year, make, vehicleModel)
	if err != nil {
		return nil, fmt.Errorf("fitment repo: vehicle search: %w", err)
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}
