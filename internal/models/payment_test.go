package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayment_JSONMarshal(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	payment := models.Payment{
		ID:            "pay_123",
		CustomerID:    "cust_456",
		Amount:        1000,
		Currency:      "USD",
		Description:   "Test Payment",
		Status:        models.PaymentStatusSucceeded,
		PaymentMethod: "card",
		StripeID:      "ch_789",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	data, err := json.Marshal(payment)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "pay_123", result["id"])
	assert.Equal(t, "cust_456", result["customer_id"])
	assert.Equal(t, float64(1000), result["amount"])
	assert.Equal(t, "USD", result["currency"])
	assert.Equal(t, "Test Payment", result["description"])
	assert.Equal(t, "succeeded", result["payment_status"])
	assert.Equal(t, "card", result["payment_method"])
	assert.Equal(t, "ch_789", result["stripe_id"])
	assert.NotEmpty(t, result["created_at"])
	assert.NotEmpty(t, result["updated_at"])
}

func TestPaymentIntent_JSONMarshal(t *testing.T) {
	t.Parallel()

	intent := models.PaymentIntent{
		ID:     "pi_123",
		Amount: 29.99,
		Status: "requires_payment_method",
	}

	data, err := json.Marshal(intent)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "pi_123", result["id"])
	assert.Equal(t, 29.99, result["amount"])
	assert.Equal(t, "requires_payment_method", result["status"])
}

func TestPaymentRequest_JSONMarshal(t *testing.T) {
	t.Parallel()

	req := models.PaymentRequest{
		CustomerID:    "cust_123",
		Amount:        5000,
		Currency:      "USD",
		Description:   "Order payment",
		PaymentMethod: "card",
		Token:         "tok_visa",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "cust_123", result["customer_id"])
	assert.Equal(t, float64(5000), result["amount"])
	assert.Equal(t, "USD", result["currency"])
	assert.Equal(t, "Order payment", result["description"])
	assert.Equal(t, "card", result["payment_method"])
	assert.Equal(t, "tok_visa", result["token"])
}

func TestPaymentResponse_JSONMarshal(t *testing.T) {
	t.Parallel()

	t.Run("full response", func(t *testing.T) {
		t.Parallel()
		payment := &models.Payment{
			ID: "pay_100",
		}
		resp := models.PaymentResponse{
			Payment:       payment,
			ClientSecret:  "secret_123",
			PaymentStatus: "succeeded",
			Message:       "Payment successful",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.NotNil(t, result["payment"])
		assert.Equal(t, "secret_123", result["client_secret"])
		assert.Equal(t, "succeeded", result["payment_status"])
		assert.Equal(t, "Payment successful", result["message"])
	})

	t.Run("omitempty fields", func(t *testing.T) {
		t.Parallel()
		resp := models.PaymentResponse{
			PaymentStatus: "pending",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		_, hasSecret := result["client_secret"]
		_, hasMessage := result["message"]
		assert.False(t, hasSecret)
		assert.False(t, hasMessage)
	})
}

func TestPayment_JSONUnmarshal(t *testing.T) {
	t.Parallel()

	jsonStr := `{
		"id": "pay_999",
		"customer_id": "cust_888",
		"amount": 2500,
		"currency": "EUR",
		"description": "Subscription",
		"payment_status": "pending",
		"payment_method": "paypal",
		"stripe_id": "ch_555",
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:00Z"
	}`

	var payment models.Payment
	err := json.Unmarshal([]byte(jsonStr), &payment)
	require.NoError(t, err)

	assert.Equal(t, "pay_999", payment.ID)
	assert.Equal(t, "cust_888", payment.CustomerID)
	assert.Equal(t, int64(2500), payment.Amount)
	assert.Equal(t, "EUR", payment.Currency)
	assert.Equal(t, "Subscription", payment.Description)
	assert.Equal(t, models.PaymentStatusPending, payment.Status)
	assert.Equal(t, "paypal", payment.PaymentMethod)
	assert.Equal(t, "ch_555", payment.StripeID)
	assert.False(t, payment.CreatedAt.IsZero())
	assert.False(t, payment.UpdatedAt.IsZero())
}

func TestPaymentIntent_JSONUnmarshal(t *testing.T) {
	t.Parallel()

	jsonStr := `{
		"id": "pi_777",
		"amount": 49.95,
		"status": "succeeded"
	}`

	var intent models.PaymentIntent
	err := json.Unmarshal([]byte(jsonStr), &intent)
	require.NoError(t, err)

	assert.Equal(t, "pi_777", intent.ID)
	assert.Equal(t, 49.95, intent.Amount)
	assert.Equal(t, "succeeded", intent.Status)
}

func TestPaymentRequest_JSONUnmarshal(t *testing.T) {
	t.Parallel()

	jsonStr := `{
		"customer_id": "cust_111",
		"amount": 1500,
		"currency": "GBP",
		"description": "Item purchase",
		"payment_method": "card",
		"token": "tok_123"
	}`

	var req models.PaymentRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	require.NoError(t, err)

	assert.Equal(t, "cust_111", req.CustomerID)
	assert.Equal(t, int64(1500), req.Amount)
	assert.Equal(t, "GBP", req.Currency)
	assert.Equal(t, "Item purchase", req.Description)
	assert.Equal(t, "card", req.PaymentMethod)
	assert.Equal(t, "tok_123", req.Token)
}

func TestPaymentResponse_JSONUnmarshal(t *testing.T) {
	t.Parallel()

	jsonStr := `{
		"payment": {
			"id": "pay_333"
		},
		"client_secret": "sec_abc",
		"payment_status": "succeeded",
		"message": "Done"
	}`

	var resp models.PaymentResponse
	err := json.Unmarshal([]byte(jsonStr), &resp)
	require.NoError(t, err)

	require.NotNil(t, resp.Payment)
	assert.Equal(t, "pay_333", resp.Payment.ID)
	assert.Equal(t, "sec_abc", resp.ClientSecret)
	assert.Equal(t, "succeeded", resp.PaymentStatus)
	assert.Equal(t, "Done", resp.Message)
}

func TestPaymentRequest_Validation(t *testing.T) {
	t.Parallel()

	validate := validator.New()

	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		req := models.PaymentRequest{
			CustomerID:    "cust_123",
			Amount:        100,
			Currency:      "USD",
			Description:   "Valid payment",
			PaymentMethod: "card",
			Token:         "tok_valid",
		}
		err := validate.Struct(req)
		assert.NoError(t, err)
	})

	type testCase struct {
		name      string
		field     string
		modifyReq func(r *models.PaymentRequest)
	}

	testCases := []testCase{
		{
			name:  "missing customer_id",
			field: "CustomerID",
			modifyReq: func(r *models.PaymentRequest) {
				r.CustomerID = ""
			},
		},
		{
			name:  "invalid amount zero",
			field: "Amount",
			modifyReq: func(r *models.PaymentRequest) {
				r.Amount = 0
			},
		},
		{
			name:  "invalid amount negative",
			field: "Amount",
			modifyReq: func(r *models.PaymentRequest) {
				r.Amount = -10
			},
		},
		{
			name:  "missing currency",
			field: "Currency",
			modifyReq: func(r *models.PaymentRequest) {
				r.Currency = ""
			},
		},
		{
			name:  "currency longer than 3 characters",
			field: "Currency",
			modifyReq: func(r *models.PaymentRequest) {
				r.Currency = "USDD"
			},
		},
		{
			name:  "missing description",
			field: "Description",
			modifyReq: func(r *models.PaymentRequest) {
				r.Description = ""
			},
		},
		{
			name:  "missing payment_method",
			field: "PaymentMethod",
			modifyReq: func(r *models.PaymentRequest) {
				r.PaymentMethod = ""
			},
		},
		{
			name:  "missing token",
			field: "Token",
			modifyReq: func(r *models.PaymentRequest) {
				r.Token = ""
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := models.PaymentRequest{
				CustomerID:    "cust_123",
				Amount:        100,
				Currency:      "USD",
				Description:   "Valid payment",
				PaymentMethod: "card",
				Token:         "tok_valid",
			}
			tc.modifyReq(&req)

			err := validate.Struct(req)
			require.Error(t, err)

			var validationErrors validator.ValidationErrors
			require.ErrorAs(t, err, &validationErrors)

			found := false
			for _, fieldErr := range validationErrors {
				if fieldErr.Field() == tc.field {
					found = true
					break
				}
			}
			assert.True(t, found, "expected validation error for field %s", tc.field)
		})
	}
}
