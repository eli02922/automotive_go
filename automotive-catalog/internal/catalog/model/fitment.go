package model

import "time"

// Fitment represents a product-to-vehicle application record (ACES standard).
type Fitment struct {
	ID         string            `json:"id" db:"id"`
	ProductID  string            `json:"product_id" db:"product_id"`
	VehicleID  string            `json:"vehicle_id" db:"vehicle_id"`
	Year       int               `json:"year" db:"year"`
	Make       string            `json:"make" db:"make"`
	Model      string            `json:"model" db:"model"`
	SubModel   string            `json:"sub_model" db:"sub_model"`
	Engine     string            `json:"engine" db:"engine"`
	Notes      string            `json:"notes" db:"notes"`
	Qualifiers map[string]string `json:"qualifiers" db:"qualifiers"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at" db:"updated_at"`
}

type Vehicle struct {
	ID       string `json:"id" db:"id"`
	Year     int    `json:"year" db:"year"`
	Make     string `json:"make" db:"make"`
	Model    string `json:"model" db:"model"`
	SubModel string `json:"sub_model" db:"sub_model"`
	Engine   string `json:"engine" db:"engine"`
	Region   string `json:"region" db:"region"`
}

type FitmentFilter struct {
	ProductID string
	Year      int
	Make      string
	Model     string
	SubModel  string
	Engine    string
	Page      int
	PageSize  int
}

type FitmentPage struct {
	Items      []*Fitment `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

type UpsertFitmentRequest struct {
	ProductID  string            `json:"product_id" binding:"required"`
	Year       int               `json:"year" binding:"required,min=1900,max=2100"`
	Make       string            `json:"make" binding:"required"`
	Model      string            `json:"model" binding:"required"`
	SubModel   string            `json:"sub_model"`
	Engine     string            `json:"engine"`
	Notes      string            `json:"notes"`
	Qualifiers map[string]string `json:"qualifiers"`
}
