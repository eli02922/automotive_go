package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/embuscado/automotive-catalog/internal/catalog/model"
)

type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

func (r *ProductRepository) Create(ctx context.Context, req *model.CreateProductRequest) (*model.Product, error) {
	attrs, err := json.Marshal(req.Attributes)
	if err != nil {
		return nil, fmt.Errorf("product repo: marshal attributes: %w", err)
	}
	dims, err := json.Marshal(req.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("product repo: marshal dimensions: %w", err)
	}

	p := &model.Product{
		ID:          uuid.New().String(),
		PartNumber:  req.PartNumber,
		Brand:       req.Brand,
		Name:        req.Name,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Status:      model.ProductStatusActive,
		Weight:      req.Weight,
		Dimensions:  req.Dimensions,
		Attributes:  req.Attributes,
		MSRP:        req.MSRP,
		Cost:        req.Cost,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	query := `
		INSERT INTO products (id, part_number, brand, name, description, category_id, status, weight, dimensions, attributes, msrp, cost, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`

	_, err = r.pool.Exec(ctx, query,
		p.ID, p.PartNumber, p.Brand, p.Name, p.Description, p.CategoryID, p.Status,
		p.Weight, dims, attrs, p.MSRP, p.Cost, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("product repo: create: %w", err)
	}
	return p, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*model.Product, error) {
	query := `
		SELECT id, part_number, brand, name, description, category_id, status, weight, dimensions, attributes, msrp, cost, created_at, updated_at
		FROM products WHERE id = $1 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, id)
	return scanProduct(row)
}

func (r *ProductRepository) GetByPartNumber(ctx context.Context, partNumber string) (*model.Product, error) {
	query := `
		SELECT id, part_number, brand, name, description, category_id, status, weight, dimensions, attributes, msrp, cost, created_at, updated_at
		FROM products WHERE part_number = $1 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, partNumber)
	return scanProduct(row)
}

func (r *ProductRepository) List(ctx context.Context, f model.ProductFilter) (*model.ProductPage, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 50
	}

	args := []any{}
	where := []string{"deleted_at IS NULL"}
	argIdx := 1

	if f.Brand != "" {
		where = append(where, fmt.Sprintf("brand = $%d", argIdx))
		args = append(args, f.Brand)
		argIdx++
	}
	if f.CategoryID != "" {
		where = append(where, fmt.Sprintf("category_id = $%d", argIdx))
		args = append(args, f.CategoryID)
		argIdx++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, f.Status)
		argIdx++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("to_tsvector('english', name || ' ' || description) @@ plainto_tsquery($%d)", argIdx))
		args = append(args, f.Search)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")
	offset := (f.Page - 1) * f.PageSize

	sortCol := "created_at"
	validSorts := map[string]bool{"name": true, "created_at": true, "part_number": true, "brand": true, "msrp": true}
	if validSorts[f.SortBy] {
		sortCol = f.SortBy
	}
	sortOrder := "DESC"
	if strings.ToUpper(f.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products WHERE %s", whereClause)
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("product repo: count: %w", err)
	}

	args = append(args, f.PageSize, offset)
	dataQuery := fmt.Sprintf(`
		SELECT id, part_number, brand, name, description, category_id, status, weight, dimensions, attributes, msrp, cost, created_at, updated_at
		FROM products WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		whereClause, sortCol, sortOrder, argIdx, argIdx+1)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("product repo: list query: %w", err)
	}
	defer rows.Close()

	var items []*model.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}

	return &model.ProductPage{
		Items:      items,
		Total:      total,
		Page:       f.Page,
		PageSize:   f.PageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(f.PageSize))),
	}, nil
}

func (r *ProductRepository) Update(ctx context.Context, id string, req *model.UpdateProductRequest) (*model.Product, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.MSRP != nil {
		sets = append(sets, fmt.Sprintf("msrp = $%d", argIdx))
		args = append(args, *req.MSRP)
		argIdx++
	}
	if req.Cost != nil {
		sets = append(sets, fmt.Sprintf("cost = $%d", argIdx))
		args = append(args, *req.Cost)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE products SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(sets, ", "), argIdx)

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product repo: update: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE products SET deleted_at = NOW() WHERE id = $1", id)
	return err
}

// BulkUpsert performs a high-performance batch upsert using pgx CopyFrom.
func (r *ProductRepository) BulkUpsert(ctx context.Context, products []*model.Product) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("product repo: bulk upsert begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows := make([][]any, 0, len(products))
	for _, p := range products {
		dims, _ := json.Marshal(p.Dimensions)
		attrs, _ := json.Marshal(p.Attributes)
		rows = append(rows, []any{
			p.ID, p.PartNumber, p.Brand, p.Name, p.Description,
			p.CategoryID, p.Status, p.Weight, dims, attrs,
			p.MSRP, p.Cost, p.CreatedAt, p.UpdatedAt,
		})
	}

	cols := []string{"id", "part_number", "brand", "name", "description", "category_id", "status", "weight", "dimensions", "attributes", "msrp", "cost", "created_at", "updated_at"}
	n, err := tx.CopyFrom(ctx,
		pgx.Identifier{"products_staging"},
		cols,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("product repo: copy from: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO products SELECT * FROM products_staging
		ON CONFLICT (part_number) DO UPDATE SET
			name = EXCLUDED.name, description = EXCLUDED.description,
			status = EXCLUDED.status, msrp = EXCLUDED.msrp, cost = EXCLUDED.cost,
			updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		return 0, fmt.Errorf("product repo: merge staging: %w", err)
	}

	_, err = tx.Exec(ctx, "TRUNCATE products_staging")
	if err != nil {
		return 0, err
	}

	return n, tx.Commit(ctx)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProduct(row scanner) (*model.Product, error) {
	var p model.Product
	var dimsJSON, attrsJSON []byte

	err := row.Scan(
		&p.ID, &p.PartNumber, &p.Brand, &p.Name, &p.Description,
		&p.CategoryID, &p.Status, &p.Weight, &dimsJSON, &attrsJSON,
		&p.MSRP, &p.Cost, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("product repo: scan: %w", err)
	}

	if err := json.Unmarshal(dimsJSON, &p.Dimensions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(attrsJSON, &p.Attributes); err != nil {
		return nil, err
	}
	return &p, nil
}
