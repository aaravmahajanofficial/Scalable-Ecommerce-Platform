package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Product struct {
	ID            uuid.UUID `json:"id"`
	CategoryID    uuid.UUID `json:"category_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	StockQuantity int       `json:"stock_quantity"`
	SKU           string    `json:"sku"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Category      *Category `json:"category,omitempty"`
}

type CreateProductRequest struct {
	CategoryID    uuid.UUID `json:"category_id"           validate:"required"`
	Name          string    `json:"name"                  validate:"required,min=3,max=200"`
	Description   string    `json:"description,omitempty"`
	Price         float64   `json:"price"                 validate:"required,gt=0"`
	StockQuantity int       `json:"stock_quantity"        validate:"required,gte=0"`
	SKU           string    `json:"sku"                   validate:"required,min=3,max=50"`
}

type UpdateProductRequest struct {
	CategoryID    *uuid.UUID `json:"category_id,omitempty"`
	Name          *string    `json:"name,omitempty"           validate:"omitempty,min=3,max=200"`
	Description   *string    `json:"description,omitempty"`
	Price         *float64   `json:"price,omitempty"          validate:"omitempty,gt=0"`
	StockQuantity *int       `json:"stock_quantity,omitempty" validate:"omitempty,gte=0"`
	Status        *string    `json:"status,omitempty"         validate:"omitempty,oneof=active inactive discontinued"`
}
