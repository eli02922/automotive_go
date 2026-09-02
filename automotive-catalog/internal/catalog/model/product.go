package model

import (
	"time"
)

type ProductStatus string

const (
	ProductStatusActive      ProductStatus = "active"
	ProductStatusInactive    ProductStatus = "inactive"
	ProductStatusDiscontinued ProductStatus = "discontinued"
)

type Product struct {
	ID            string        `json:"id" db:"id"`
	PartNumber    string        `json:"part_number" db:"part_number"`
	Brand         string        `json:"brand" db:"brand"`
	Name          string        `json:"name" db:"name"`
	Description   string        `json:"description" db:"description"`
	CategoryID    string        `json:"category_id" db:"category_id"`
	Status        ProductStatus `json:"status" db:"status"`
	Weight        float64       `json:"weight" db:"weight"`
	Dimensions    Dimensions    `json:"dimensions" db:"dimensions"`
	Attributes    Attributes    `json:"attributes" db:"attributes"`
	Images        []string      `json:"images" db:"images"`
	MSRP          float64       `json:"msrp" db:"msrp"`
	Cost          float64       `json:"cost" db:"cost"`
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at" db:"updated_at"`
}

type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
}

type Attributes map[string]any

type Category struct {
	ID       string `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	ParentID string `json:"parent_id,omitempty" db:"parent_id"`
	Level    int    `json:"level" db:"level"`
}

type Inventory struct {
	ProductID       string    `json:"product_id" db:"product_id"`
	WarehouseCode   string    `json:"warehouse_code" db:"warehouse_code"`
	QuantityOnHand  int       `json:"quantity_on_hand" db:"quantity_on_hand"`
	QuantityReserved int      `json:"quantity_reserved" db:"quantity_reserved"`
	QuantityAvailable int     `json:"quantity_available" db:"quantity_available"`
	ReorderPoint    int       `json:"reorder_point" db:"reorder_point"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type ProductFilter struct {
	Brand      string
	CategoryID string
	Status     ProductStatus
	PartNumber string
	Search     string
	Page       int
	PageSize   int
	SortBy     string
	SortOrder  string
}

type ProductPage struct {
	Items      []*Product `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

type CreateProductRequest struct {
	PartNumber  string     `json:"part_number" binding:"required"`
	Brand       string     `json:"brand" binding:"required"`
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	CategoryID  string     `json:"category_id" binding:"required"`
	Weight      float64    `json:"weight"`
	Dimensions  Dimensions `json:"dimensions"`
	Attributes  Attributes `json:"attributes"`
	MSRP        float64    `json:"msrp"`
	Cost        float64    `json:"cost"`
}

type UpdateProductRequest struct {
	Name        *string     `json:"name"`
	Description *string     `json:"description"`
	Status      *ProductStatus `json:"status"`
	Weight      *float64    `json:"weight"`
	Dimensions  *Dimensions `json:"dimensions"`
	Attributes  Attributes  `json:"attributes"`
	MSRP        *float64    `json:"msrp"`
	Cost        *float64    `json:"cost"`
}
