package stripe_test

import (
	"fmt"
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"

	mystripe "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
)

type mockBackend struct {
	mock.Mock
}

func (m *mockBackend) Call(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	args := m.Called(method, path, key, params, v)
	return args.Error(0)
}

func (m *mockBackend) CallStreaming(method, path, key string, params stripe.ParamsContainer, v stripe.StreamingLastResponseSetter) error {
	args := m.Called(method, path, key, params, v)
	return args.Error(0)
}

func (m *mockBackend) CallRaw(method, path, key string, body *form.Values, params *stripe.Params, v stripe.LastResponseSetter) error {
	args := m.Called(method, path, key, body, params, v)
	return args.Error(0)
}

func (m *mockBackend) CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *stripe.Params, v stripe.LastResponseSetter) error {
	args := m.Called(method, path, key, boundary, body, params, v)
	return args.Error(0)
}

func (m *mockBackend) SetMaxNetworkRetries(maxNetworkRetries int64) {
	m.Called(maxNetworkRetries)
}

func setupMockBackend(t *testing.T) (*mockBackend, func()) {
	t.Helper()
	oldBackend := stripe.GetBackend(stripe.APIBackend)
	mb := new(mockBackend)
	stripe.SetBackend(stripe.APIBackend, mb)
	return mb, func() {
		stripe.SetBackend(stripe.APIBackend, oldBackend)
	}
}

func TestStripeClient_CreatePaymentIntent(t *testing.T) {
	tests := []struct {
		name        string
		amount      int64
		currency    string
		description string
		customerID  string
		mockErr     error
		mockID      string
		expectErr   bool
	}{
		{
			name:        "successful creation with customer",
			amount:      1000,
			currency:    "usd",
			description: "test description",
			customerID:  "cus_123",
			mockID:      "pi_123",
			expectErr:   false,
		},
		{
			name:        "successful creation without customer",
			amount:      2000,
			currency:    "eur",
			description: "another description",
			customerID:  "",
			mockID:      "pi_456",
			expectErr:   false,
		},
		{
			name:        "backend error",
			amount:      1000,
			currency:    "usd",
			description: "test",
			customerID:  "cus_123",
			mockErr:     errors.New("stripe error"),
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb, cleanup := setupMockBackend(t)
			defer cleanup()

			mb.On("Call", "POST", "/v1/payment_intents", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockErr).
				Run(func(args mock.Arguments) {
					if tt.mockErr == nil {
						v := args.Get(4).(stripe.LastResponseSetter)
						if pi, ok := v.(*stripe.PaymentIntent); ok {
							pi.ID = tt.mockID
						}
					}
				})

			client := mystripe.NewStripeClient("sk_test_123", "whsec_123")
			pi, err := client.CreatePaymentIntent(tt.amount, tt.currency, tt.description, tt.customerID)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pi)
				assert.Equal(t, tt.mockID, pi.ID)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestStripeClient_CreatePaymentMethod(t *testing.T) {
	tests := []struct {
		name         string
		cardNumber   string
		cardExpMonth string
		cardExpYear  string
		cardCVC      string
		mockErr      error
		mockID       string
		expectErr    bool
		errContains  string
	}{
		{
			name:         "successful creation",
			cardNumber:   "4242424242424242",
			cardExpMonth: "12",
			cardExpYear:  "2030",
			cardCVC:      "123",
			mockID:       "pm_123",
			expectErr:    false,
		},
		{
			name:         "invalid month",
			cardNumber:   "4242424242424242",
			cardExpMonth: "invalid",
			cardExpYear:  "2030",
			cardCVC:      "123",
			expectErr:    true,
			errContains:  "invalid card expiration month",
		},
		{
			name:         "invalid year",
			cardNumber:   "4242424242424242",
			cardExpMonth: "12",
			cardExpYear:  "invalid",
			cardCVC:      "123",
			expectErr:    true,
			errContains:  "invalid card expiration year",
		},
		{
			name:         "backend error",
			cardNumber:   "4242424242424242",
			cardExpMonth: "12",
			cardExpYear:  "2030",
			cardCVC:      "123",
			mockErr:      errors.New("stripe error"),
			expectErr:    true,
			errContains:  "stripe error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb, cleanup := setupMockBackend(t)
			defer cleanup()

			if tt.errContains == "" || tt.errContains == "stripe error" {
				mb.On("Call", "POST", "/v1/payment_methods", mock.Anything, mock.Anything, mock.Anything).
					Return(tt.mockErr).
					Run(func(args mock.Arguments) {
						if tt.mockErr == nil {
							v := args.Get(4).(stripe.LastResponseSetter)
							if pm, ok := v.(*stripe.PaymentMethod); ok {
								pm.ID = tt.mockID
							}
						}
					}).Maybe()
			}

			client := mystripe.NewStripeClient("sk_test", "whsec_test")
			pm, err := client.CreatePaymentMethod(tt.cardNumber, tt.cardExpMonth, tt.cardExpYear, tt.cardCVC)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.ErrorContains(t, err, tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pm)
				assert.Equal(t, tt.mockID, pm.ID)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestStripeClient_CreatePaymentMethodFromToken(t *testing.T) {
	tests := []struct {
		name      string
		pmID      string
		mockErr   error
		mockID    string
		expectErr bool
	}{
		{
			name:      "successful retrieval",
			pmID:      "pm_123",
			mockID:    "pm_123",
			expectErr: false,
		},
		{
			name:      "backend error",
			pmID:      "pm_123",
			mockErr:   errors.New("stripe error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb, cleanup := setupMockBackend(t)
			defer cleanup()

			mb.On("Call", "GET", "/v1/payment_methods/"+tt.pmID, mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockErr).
				Run(func(args mock.Arguments) {
					if tt.mockErr == nil {
						v := args.Get(4).(stripe.LastResponseSetter)
						if pm, ok := v.(*stripe.PaymentMethod); ok {
							pm.ID = tt.mockID
						}
					}
				})

			client := mystripe.NewStripeClient("sk_test", "whsec_test")
			pm, err := client.CreatePaymentMethodFromToken(tt.pmID)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pm)
				assert.Equal(t, tt.mockID, pm.ID)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestStripeClient_AttachPaymentMethodToIntent(t *testing.T) {
	tests := []struct {
		name            string
		paymentMethodID string
		paymentIntentID string
		mockErr         error
		expectErr       bool
	}{
		{
			name:            "successful attachment",
			paymentMethodID: "pm_123",
			paymentIntentID: "pi_123",
			expectErr:       false,
		},
		{
			name:            "backend error",
			paymentMethodID: "pm_123",
			paymentIntentID: "pi_123",
			mockErr:         errors.New("stripe error"),
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb, cleanup := setupMockBackend(t)
			defer cleanup()

			mb.On("Call", "POST", "/v1/payment_intents/"+tt.paymentIntentID, mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockErr)

			client := mystripe.NewStripeClient("sk_test", "whsec_test")
			err := client.AttachPaymentMethodToIntent(tt.paymentMethodID, tt.paymentIntentID)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestStripeClient_ConfirmPaymentIntent(t *testing.T) {
	tests := []struct {
		name            string
		paymentIntentID string
		mockErr         error
		mockID          string
		expectErr       bool
	}{
		{
			name:            "successful confirmation",
			paymentIntentID: "pi_123",
			mockID:          "pi_123",
			expectErr:       false,
		},
		{
			name:            "backend error",
			paymentIntentID: "pi_123",
			mockErr:         errors.New("stripe error"),
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb, cleanup := setupMockBackend(t)
			defer cleanup()

			mb.On("Call", "POST", "/v1/payment_intents/"+tt.paymentIntentID+"/confirm", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockErr).
				Run(func(args mock.Arguments) {
					if tt.mockErr == nil {
						v := args.Get(4).(stripe.LastResponseSetter)
						if pi, ok := v.(*stripe.PaymentIntent); ok {
							pi.ID = tt.mockID
						}
					}
				})

			client := mystripe.NewStripeClient("sk_test", "whsec_test")
			pi, err := client.ConfirmPaymentIntent(tt.paymentIntentID)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pi)
				assert.Equal(t, tt.mockID, pi.ID)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestStripeClient_RefundPayment(t *testing.T) {
	tests := []struct {
		name            string
		paymentIntentID string
		amount          int64
		mockErr         error
		mockID          string
		expectErr       bool
	}{
		{
			name:            "successful refund",
			paymentIntentID: "pi_123",
			amount:          1000,
			mockID:          "re_123",
			expectErr:       false,
		},
		{
			name:            "backend error",
			paymentIntentID: "pi_123",
			amount:          1000,
			mockErr:         errors.New("stripe error"),
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb, cleanup := setupMockBackend(t)
			defer cleanup()

			mb.On("Call", "POST", "/v1/refunds", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockErr).
				Run(func(args mock.Arguments) {
					if tt.mockErr == nil {
						v := args.Get(4).(stripe.LastResponseSetter)
						if re, ok := v.(*stripe.Refund); ok {
							re.ID = tt.mockID
						}
					}
				})

			client := mystripe.NewStripeClient("sk_test", "whsec_test")
			re, err := client.RefundPayment(tt.paymentIntentID, tt.amount)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, re)
				assert.Equal(t, tt.mockID, re.ID)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestStripeClient_VerifyWebhookSignature(t *testing.T) {
	t.Run("empty webhook secret", func(t *testing.T) {
		client := mystripe.NewStripeClient("sk_test", "")
		_, err := client.VerifyWebhookSignature([]byte("payload"), "signature")
		assert.ErrorContains(t, err, "webhook secret not configured")
	})

	t.Run("invalid signature", func(t *testing.T) {
		client := mystripe.NewStripeClient("sk_test", "whsec_test")

		// This will fail inside webhook.ConstructEvent because the signature doesn't match the payload for the given secret
		// We're essentially testing that we are properly passing down the args to the Stripe webhook package
		// We expect a webhook.Error or similar.

		// To mock stripe's webhook.ConstructEvent is not possible because it's a standalone function,
		// but we can pass real-looking but invalid data and expect an error to prove we are calling it.
		// A signature usually has a timestamp t=... and a signature v1=...

		tNow := time.Now().Unix()
		sigHeader := "t=" + fmt.Sprintf("%d", tNow) + ",v1=invalid_sig"

		_, err := client.VerifyWebhookSignature([]byte("invalid_payload"), sigHeader)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook had no valid signature")
	})
}
