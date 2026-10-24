package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderStatusConstants(t *testing.T) {
	assert.Equal(t, models.OrderStatus("pending"), models.OrderStatusPending)
	assert.Equal(t, models.OrderStatus("confirmed"), models.OrderStatusConfirmed)
	assert.Equal(t, models.OrderStatus("shipping"), models.OrderStatusShipping)
	assert.Equal(t, models.OrderStatus("delivered"), models.OrderStatusDelivered)
	assert.Equal(t, models.OrderStatus("cancelled"), models.OrderStatusCancelled)
}

func TestOrderJSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	orderID := uuid.New()
	customerID := uuid.New()
	itemID := uuid.New()
	productID := uuid.New()

	order := models.Order{
		ID:              orderID,
		CustomerID:      customerID,
		Status:          models.OrderStatusPending,
		TotalAmount:     99.99,
		PaymentStatus:   models.PaymentStatusPending,
		PaymentIntentID: "pi_123456",
		ShippingAddress: &models.Address{
			Street:     "123 Main St",
			City:       "Springfield",
			State:      "IL",
			PostalCode: "62701",
			Country:    "US",
		},
		Items: []models.OrderItem{
			{
				ID:        itemID,
				OrderID:   orderID,
				ProductID: productID,
				Quantity:  2,
				UnitPrice: 49.995,
				CreatedAt: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(order)
	require.NoError(t, err)

	var unmarshaled models.Order
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, order.ID, unmarshaled.ID)
	assert.Equal(t, order.CustomerID, unmarshaled.CustomerID)
	assert.Equal(t, order.Status, unmarshaled.Status)
	assert.Equal(t, order.TotalAmount, unmarshaled.TotalAmount)
	assert.Equal(t, order.PaymentStatus, unmarshaled.PaymentStatus)
	assert.Equal(t, order.PaymentIntentID, unmarshaled.PaymentIntentID)
	assert.Equal(t, order.ShippingAddress, unmarshaled.ShippingAddress)
	assert.Len(t, unmarshaled.Items, 1)
	assert.Equal(t, order.Items[0].ProductID, unmarshaled.Items[0].ProductID)
}

func TestOrderResponseAndHistoryResponseJSON(t *testing.T) {
	order := models.Order{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		Status:     models.OrderStatusConfirmed,
	}

	resp := models.OrderResponse{
		Order: &order,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var unmarshaledResp models.OrderResponse
	err = json.Unmarshal(data, &unmarshaledResp)
	require.NoError(t, err)

	assert.Equal(t, resp.Order.ID, unmarshaledResp.Order.ID)
	assert.Equal(t, resp.Order.Status, unmarshaledResp.Order.Status)

	historyResp := models.OrderHistoryResponse{
		Orders: []models.Order{order},
		Total:  1,
		Page:   1,
		Size:   10,
	}

	historyData, err := json.Marshal(historyResp)
	require.NoError(t, err)

	var unmarshaledHistory models.OrderHistoryResponse
	err = json.Unmarshal(historyData, &unmarshaledHistory)
	require.NoError(t, err)

	assert.Equal(t, historyResp.Total, unmarshaledHistory.Total)
	assert.Equal(t, historyResp.Page, unmarshaledHistory.Page)
	assert.Equal(t, historyResp.Size, unmarshaledHistory.Size)
	assert.Len(t, unmarshaledHistory.Orders, 1)
	assert.Equal(t, historyResp.Orders[0].ID, unmarshaledHistory.Orders[0].ID)
}

func TestAddressValidation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		address models.Address
		wantErr bool
	}{
		{
			name: "valid address",
			address: models.Address{
				Street:     "123 Main St",
				City:       "Springfield",
				State:      "IL",
				PostalCode: "62701",
				Country:    "US",
			},
			wantErr: false,
		},
		{
			name: "missing street",
			address: models.Address{
				City:       "Springfield",
				State:      "IL",
				PostalCode: "62701",
				Country:    "US",
			},
			wantErr: true,
		},
		{
			name: "invalid country code format",
			address: models.Address{
				Street:     "123 Main St",
				City:       "Springfield",
				State:      "IL",
				PostalCode: "62701",
				Country:    "USA", // iso3166_1_alpha2 expects 2-letter code
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.address)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderItemValidation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		item    models.OrderItem
		wantErr bool
	}{
		{
			name: "valid item",
			item: models.OrderItem{
				ProductID: uuid.New(),
				Quantity:  1,
				UnitPrice: 15.50,
			},
			wantErr: false,
		},
		{
			name: "missing product_id",
			item: models.OrderItem{
				Quantity:  1,
				UnitPrice: 15.50,
			},
			wantErr: true,
		},
		{
			name: "invalid quantity less than min",
			item: models.OrderItem{
				ProductID: uuid.New(),
				Quantity:  0,
				UnitPrice: 15.50,
			},
			wantErr: true,
		},
		{
			name: "negative unit price",
			item: models.OrderItem{
				ProductID: uuid.New(),
				Quantity:  1,
				UnitPrice: -1.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.item)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderValidation(t *testing.T) {
	validate := validator.New()

	validAddress := models.Address{
		Street:     "123 Main St",
		City:       "Springfield",
		State:      "IL",
		PostalCode: "62701",
		Country:    "US",
	}

	validItem := models.OrderItem{
		ProductID: uuid.New(),
		Quantity:  1,
		UnitPrice: 10.0,
	}

	tests := []struct {
		name    string
		order   models.Order
		wantErr bool
	}{
		{
			name: "valid order",
			order: models.Order{
				CustomerID:      uuid.New(),
				ShippingAddress: &validAddress,
				Items:           []models.OrderItem{validItem},
			},
			wantErr: false,
		},
		{
			name: "missing customer_id",
			order: models.Order{
				ShippingAddress: &validAddress,
				Items:           []models.OrderItem{validItem},
			},
			wantErr: true,
		},
		{
			name: "missing shipping_address",
			order: models.Order{
				CustomerID: uuid.New(),
				Items:      []models.OrderItem{validItem},
			},
			wantErr: true,
		},
		{
			name: "empty items slice",
			order: models.Order{
				CustomerID:      uuid.New(),
				ShippingAddress: &validAddress,
				Items:           []models.OrderItem{},
			},
			wantErr: true,
		},
		{
			name: "invalid item nested in items",
			order: models.Order{
				CustomerID:      uuid.New(),
				ShippingAddress: &validAddress,
				Items: []models.OrderItem{
					{
						ProductID: uuid.Nil, // required product_id missing
						Quantity:  1,
						UnitPrice: 10.0,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.order)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateOrderRequestValidation(t *testing.T) {
	validate := validator.New()

	validAddress := models.Address{
		Street:     "123 Main St",
		City:       "Springfield",
		State:      "IL",
		PostalCode: "62701",
		Country:    "US",
	}

	validItem := models.OrderItem{
		ProductID: uuid.New(),
		Quantity:  1,
		UnitPrice: 10.0,
	}

	tests := []struct {
		name    string
		req     models.CreateOrderRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: models.CreateOrderRequest{
				CustomerID:      uuid.New(),
				ShippingAddress: validAddress,
				Items:           []models.OrderItem{validItem},
			},
			wantErr: false,
		},
		{
			name: "missing customer_id",
			req: models.CreateOrderRequest{
				ShippingAddress: validAddress,
				Items:           []models.OrderItem{validItem},
			},
			wantErr: true,
		},
		{
			name: "invalid shipping address nested",
			req: models.CreateOrderRequest{
				CustomerID: uuid.New(),
				ShippingAddress: models.Address{
					Street: "", // required street missing
				},
				Items: []models.OrderItem{validItem},
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

func TestUpdateOrderStatusRequestValidation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		req     models.UpdateOrderStatusRequest
		wantErr bool
	}{
		{
			name: "valid status pending",
			req: models.UpdateOrderStatusRequest{
				Status: models.OrderStatusPending,
			},
			wantErr: false,
		},
		{
			name: "valid status confirmed",
			req: models.UpdateOrderStatusRequest{
				Status: models.OrderStatusConfirmed,
			},
			wantErr: false,
		},
		{
			name: "valid status shipping",
			req: models.UpdateOrderStatusRequest{
				Status: models.OrderStatusShipping,
			},
			wantErr: false,
		},
		{
			name: "valid status delivered",
			req: models.UpdateOrderStatusRequest{
				Status: models.OrderStatusDelivered,
			},
			wantErr: false,
		},
		{
			name: "valid status cancelled",
			req: models.UpdateOrderStatusRequest{
				Status: models.OrderStatusCancelled,
			},
			wantErr: false,
		},
		{
			name: "invalid status value",
			req: models.UpdateOrderStatusRequest{
				Status: models.OrderStatus("invalid_status"),
			},
			wantErr: true,
		},
		{
			name: "empty status",
			req: models.UpdateOrderStatusRequest{
				Status: "",
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
