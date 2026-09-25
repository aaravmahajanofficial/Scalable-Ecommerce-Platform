package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
)

func TestCartItem_JSONSerialization(t *testing.T) {
	prodID := uuid.New()
	item := models.CartItem{
		ProductID:  prodID,
		Quantity:   2,
		UnitPrice:  15.50,
		TotalPrice: 31.00,
	}

	data, err := json.Marshal(item)
	require.NoError(t, err)

	var unmarshaled models.CartItem
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, item, unmarshaled)
}

func TestCart_JSONSerialization(t *testing.T) {
	cartID := uuid.New()
	userID := uuid.New()
	prodID := uuid.New()
	now := time.Now().Truncate(time.Millisecond)

	cart := models.Cart{
		ID:     cartID,
		UserID: userID,
		Items: map[string]models.CartItem{
			prodID.String(): {
				ProductID:  prodID,
				Quantity:   3,
				UnitPrice:  10.0,
				TotalPrice: 30.0,
			},
		},
		Total:     30.0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(cart)
	require.NoError(t, err)

	var unmarshaled models.Cart
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, cart.ID, unmarshaled.ID)
	assert.Equal(t, cart.UserID, unmarshaled.UserID)
	assert.Equal(t, cart.Total, unmarshaled.Total)
	assert.Len(t, unmarshaled.Items, 1)
	assert.Equal(t, cart.Items[prodID.String()], unmarshaled.Items[prodID.String()])
	assert.True(t, cart.CreatedAt.Equal(unmarshaled.CreatedAt))
	assert.True(t, cart.UpdatedAt.Equal(unmarshaled.UpdatedAt))
}

func TestAddItemRequest_JSONSerialization(t *testing.T) {
	prodID := uuid.New()
	req := models.AddItemRequest{
		ProductID: prodID,
		Quantity:  5,
		UnitPrice: 25.0,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var unmarshaled models.AddItemRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, req, unmarshaled)
}

func TestAddItemRequest_Validation(t *testing.T) {
	validate := validator.New()
	validProdID := uuid.New()

	tests := []struct {
		name    string
		req     models.AddItemRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: models.AddItemRequest{
				ProductID: validProdID,
				Quantity:  1,
				UnitPrice: 10.0,
			},
			wantErr: false,
		},
		{
			name: "missing product_id (zero UUID)",
			req: models.AddItemRequest{
				ProductID: uuid.Nil,
				Quantity:  1,
				UnitPrice: 10.0,
			},
			wantErr: true,
		},
		{
			name: "invalid quantity (zero)",
			req: models.AddItemRequest{
				ProductID: validProdID,
				Quantity:  0,
				UnitPrice: 10.0,
			},
			wantErr: true,
		},
		{
			name: "invalid quantity (negative)",
			req: models.AddItemRequest{
				ProductID: validProdID,
				Quantity:  -1,
				UnitPrice: 10.0,
			},
			wantErr: true,
		},
		{
			name: "invalid unit price (negative)",
			req: models.AddItemRequest{
				ProductID: validProdID,
				Quantity:  1,
				UnitPrice: -0.01,
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

func TestUpdateQuantityRequest_JSONSerialization(t *testing.T) {
	prodID := uuid.New()
	req := models.UpdateQuantityRequest{
		ProductID: prodID,
		Quantity:  2,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var unmarshaled models.UpdateQuantityRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, req, unmarshaled)
}

func TestUpdateQuantityRequest_Validation(t *testing.T) {
	validate := validator.New()
	validProdID := uuid.New()

	tests := []struct {
		name    string
		req     models.UpdateQuantityRequest
		wantErr bool
	}{
		{
			name: "valid request with positive quantity",
			req: models.UpdateQuantityRequest{
				ProductID: validProdID,
				Quantity:  5,
			},
			wantErr: false,
		},
		{
			name: "missing product_id (zero UUID)",
			req: models.UpdateQuantityRequest{
				ProductID: uuid.Nil,
				Quantity:  2,
			},
			wantErr: true,
		},
		{
			name: "invalid quantity (negative)",
			req: models.UpdateQuantityRequest{
				ProductID: validProdID,
				Quantity:  -1,
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
