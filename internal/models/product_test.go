package models_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProductRequest_Validation(t *testing.T) {
	validate := validator.New()
	validCategoryID := uuid.New()

	tests := []struct {
		name    string
		req     models.CreateProductRequest
		wantErr bool
	}{
		{
			name: "valid create product request",
			req: models.CreateProductRequest{
				CategoryID:    validCategoryID,
				Name:          "Valid Product Name",
				Description:   "Optional description",
				Price:         29.99,
				StockQuantity: 100,
				SKU:           "SKU-12345",
			},
			wantErr: false,
		},
		{
			name: "missing category_id",
			req: models.CreateProductRequest{
				CategoryID:    uuid.Nil,
				Name:          "Valid Product Name",
				Price:         29.99,
				StockQuantity: 10,
				SKU:           "SKU-12345",
			},
			wantErr: true,
		},
		{
			name: "name too short",
			req: models.CreateProductRequest{
				CategoryID:    validCategoryID,
				Name:          "AB",
				Price:         29.99,
				StockQuantity: 10,
				SKU:           "SKU-12345",
			},
			wantErr: true,
		},
		{
			name: "name too long",
			req: models.CreateProductRequest{
				CategoryID:    validCategoryID,
				Name:          strings.Repeat("a", 201),
				Price:         29.99,
				StockQuantity: 10,
				SKU:           "SKU-12345",
			},
			wantErr: true,
		},
		{
			name: "price zero or negative",
			req: models.CreateProductRequest{
				CategoryID:    validCategoryID,
				Name:          "Valid Product",
				Price:         0,
				StockQuantity: 10,
				SKU:           "SKU-12345",
			},
			wantErr: true,
		},
		{
			name: "stock quantity negative",
			req: models.CreateProductRequest{
				CategoryID:    validCategoryID,
				Name:          "Valid Product",
				Price:         10.0,
				StockQuantity: -1,
				SKU:           "SKU-12345",
			},
			wantErr: true,
		},
		{
			name: "sku too short",
			req: models.CreateProductRequest{
				CategoryID:    validCategoryID,
				Name:          "Valid Product",
				Price:         10.0,
				StockQuantity: 5,
				SKU:           "AB",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateProductRequest_Validation(t *testing.T) {
	validate := validator.New()

	validCategoryID := uuid.New()
	validName := "Updated Product Name"
	shortName := "AB"
	validPrice := 49.99
	invalidPrice := 0.0
	validStock := 50
	invalidStock := -5
	validStatus := "active"
	invalidStatus := "unknown"

	tests := []struct {
		name    string
		req     models.UpdateProductRequest
		wantErr bool
	}{
		{
			name:    "empty update request (all fields optional)",
			req:     models.UpdateProductRequest{},
			wantErr: false,
		},
		{
			name: "valid full update request",
			req: models.UpdateProductRequest{
				CategoryID:    &validCategoryID,
				Name:          &validName,
				Price:         &validPrice,
				StockQuantity: &validStock,
				Status:        &validStatus,
			},
			wantErr: false,
		},
		{
			name: "invalid name",
			req: models.UpdateProductRequest{
				Name: &shortName,
			},
			wantErr: true,
		},
		{
			name: "invalid price",
			req: models.UpdateProductRequest{
				Price: &invalidPrice,
			},
			wantErr: true,
		},
		{
			name: "invalid stock quantity",
			req: models.UpdateProductRequest{
				StockQuantity: &invalidStock,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			req: models.UpdateProductRequest{
				Status: &invalidStatus,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProductModels_JSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	categoryID := uuid.New()
	productID := uuid.New()

	category := models.Category{
		ID:          categoryID,
		Name:        "Electronics",
		Description: "Gadgets and devices",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	product := models.Product{
		ID:            productID,
		CategoryID:    categoryID,
		Name:          "Smartphone",
		Description:   "Latest model",
		Price:         999.99,
		StockQuantity: 50,
		SKU:           "PHONE-999",
		Status:        "active",
		CreatedAt:     now,
		UpdatedAt:     now,
		Category:      &category,
	}

	t.Run("Category JSON serialization and deserialization", func(t *testing.T) {
		data, err := json.Marshal(category)
		require.NoError(t, err)

		var unmarshaled models.Category
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, category.ID, unmarshaled.ID)
		assert.Equal(t, category.Name, unmarshaled.Name)
		assert.Equal(t, category.Description, unmarshaled.Description)
	})

	t.Run("Product JSON serialization and deserialization", func(t *testing.T) {
		data, err := json.Marshal(product)
		require.NoError(t, err)

		var unmarshaled models.Product
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, product.ID, unmarshaled.ID)
		assert.Equal(t, product.CategoryID, unmarshaled.CategoryID)
		assert.Equal(t, product.Name, unmarshaled.Name)
		assert.Equal(t, product.Price, unmarshaled.Price)
		assert.NotNil(t, unmarshaled.Category)
		assert.Equal(t, product.Category.Name, unmarshaled.Category.Name)
	})
}
