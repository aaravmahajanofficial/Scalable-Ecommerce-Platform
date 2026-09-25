package stripe

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

type mockStripeBackend struct {
	callFunc func(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error
}

func (m *mockStripeBackend) Call(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	if m.callFunc != nil {
		return m.callFunc(method, path, key, params, v)
	}
	return nil
}

func (m *mockStripeBackend) CallStreaming(_, _, _ string, _ stripe.ParamsContainer, _ stripe.StreamingLastResponseSetter) error {
	return nil
}

func (m *mockStripeBackend) CallRaw(_, _, _ string, _ []byte, _ *stripe.Params, _ stripe.LastResponseSetter) error {
	return nil
}

func (m *mockStripeBackend) CallMultipart(_, _, _, _ string, _ *bytes.Buffer, _ *stripe.Params, _ stripe.LastResponseSetter) error {
	return nil
}

func (m *mockStripeBackend) SetMaxNetworkRetries(_ int64) {}

func setupTestBackend(t *testing.T, callFn func(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error) {
	t.Helper()
	origBackend := stripe.GetBackend(stripe.APIBackend)
	origKey := stripe.Key
	t.Cleanup(func() {
		stripe.SetBackend(stripe.APIBackend, origBackend)
		stripe.Key = origKey
	})

	mock := &mockStripeBackend{callFunc: callFn}
	stripe.SetBackend(stripe.APIBackend, mock)
}

func TestNewStripeClient(t *testing.T) {
	origKey := stripe.Key
	t.Cleanup(func() {
		stripe.Key = origKey
	})

	client := NewStripeClient("test_api_key", "whsec_test")
	assert.NotNil(t, client)
	assert.Equal(t, "test_api_key", stripe.Key)
}

func TestCreatePaymentIntent(t *testing.T) {
	t.Run("success with customer ID", func(t *testing.T) {
		setupTestBackend(t, func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			assert.Equal(t, "POST", method)
			assert.Equal(t, "/v1/payment_intents", path)
			if pi, ok := v.(*stripe.PaymentIntent); ok {
				pi.ID = "pi_123"
				pi.Amount = 1000
				pi.Currency = stripe.CurrencyUSD
			}
			return nil
		})

		client := NewStripeClient("sk_test", "whsec_test")
		pi, err := client.CreatePaymentIntent(1000, "usd", "Test payment", "cus_123")
		assert.NoError(t, err)
		assert.NotNil(t, pi)
		assert.Equal(t, "pi_123", pi.ID)
	})

	t.Run("success without customer ID", func(t *testing.T) {
		setupTestBackend(t, func(_, _, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			if pi, ok := v.(*stripe.PaymentIntent); ok {
				pi.ID = "pi_456"
			}
			return nil
		})

		client := NewStripeClient("sk_test", "whsec_test")
		pi, err := client.CreatePaymentIntent(500, "usd", "Test payment no cus", "")
		assert.NoError(t, err)
		assert.NotNil(t, pi)
		assert.Equal(t, "pi_456", pi.ID)
	})

	t.Run("backend error", func(t *testing.T) {
		setupTestBackend(t, func(_, _, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			return errors.New("stripe API error")
		})

		client := NewStripeClient("sk_test", "whsec_test")
		_, err := client.CreatePaymentIntent(1000, "usd", "Test", "cus_123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stripe API error")
	})
}

func TestCreatePaymentMethod(t *testing.T) {
	t.Run("invalid month", func(t *testing.T) {
		client := NewStripeClient("sk_test", "whsec_test")
		pm, err := client.CreatePaymentMethod("4242424242424242", "invalid", "2025", "123")
		assert.Error(t, err)
		assert.Nil(t, pm)
		assert.Contains(t, err.Error(), "invalid card expiration month")
	})

	t.Run("invalid year", func(t *testing.T) {
		client := NewStripeClient("sk_test", "whsec_test")
		pm, err := client.CreatePaymentMethod("4242424242424242", "12", "invalid", "123")
		assert.Error(t, err)
		assert.Nil(t, pm)
		assert.Contains(t, err.Error(), "invalid card expiration year")
	})

	t.Run("success", func(t *testing.T) {
		setupTestBackend(t, func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			assert.Equal(t, "POST", method)
			assert.Equal(t, "/v1/payment_methods", path)
			if pm, ok := v.(*stripe.PaymentMethod); ok {
				pm.ID = "pm_123"
			}
			return nil
		})

		client := NewStripeClient("sk_test", "whsec_test")
		pm, err := client.CreatePaymentMethod("4242424242424242", "12", "2025", "123")
		assert.NoError(t, err)
		assert.NotNil(t, pm)
		assert.Equal(t, "pm_123", pm.ID)
	})

	t.Run("backend error", func(t *testing.T) {
		setupTestBackend(t, func(_, _, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			return errors.New("backend error")
		})

		client := NewStripeClient("sk_test", "whsec_test")
		_, err := client.CreatePaymentMethod("4242424242424242", "12", "2025", "123")
		assert.Error(t, err)
	})
}

func TestCreatePaymentMethodFromToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setupTestBackend(t, func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			assert.Equal(t, "GET", method)
			assert.Equal(t, "/v1/payment_methods/pm_token_123", path)
			if pm, ok := v.(*stripe.PaymentMethod); ok {
				pm.ID = "pm_token_123"
			}
			return nil
		})

		client := NewStripeClient("sk_test", "whsec_test")
		pm, err := client.CreatePaymentMethodFromToken("pm_token_123")
		assert.NoError(t, err)
		assert.NotNil(t, pm)
		assert.Equal(t, "pm_token_123", pm.ID)
	})

	t.Run("backend error", func(t *testing.T) {
		setupTestBackend(t, func(_, _, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			return errors.New("backend error")
		})

		client := NewStripeClient("sk_test", "whsec_test")
		_, err := client.CreatePaymentMethodFromToken("pm_token_123")
		assert.Error(t, err)
	})
}

func TestAttachPaymentMethodToIntent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setupTestBackend(t, func(method, path, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			assert.Equal(t, "POST", method)
			assert.Equal(t, "/v1/payment_intents/pi_123", path)
			return nil
		})

		client := NewStripeClient("sk_test", "whsec_test")
		err := client.AttachPaymentMethodToIntent("pm_123", "pi_123")
		assert.NoError(t, err)
	})

	t.Run("backend error", func(t *testing.T) {
		setupTestBackend(t, func(_, _, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			return errors.New("backend error")
		})

		client := NewStripeClient("sk_test", "whsec_test")
		err := client.AttachPaymentMethodToIntent("pm_123", "pi_123")
		assert.Error(t, err)
	})
}

func TestConfirmPaymentIntent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setupTestBackend(t, func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			assert.Equal(t, "POST", method)
			assert.Equal(t, "/v1/payment_intents/pi_123/confirm", path)
			if pi, ok := v.(*stripe.PaymentIntent); ok {
				pi.ID = "pi_123"
				pi.Status = stripe.PaymentIntentStatusSucceeded
			}
			return nil
		})

		client := NewStripeClient("sk_test", "whsec_test")
		pi, err := client.ConfirmPaymentIntent("pi_123")
		assert.NoError(t, err)
		assert.NotNil(t, pi)
		assert.Equal(t, stripe.PaymentIntentStatusSucceeded, pi.Status)
	})

	t.Run("backend error", func(t *testing.T) {
		setupTestBackend(t, func(_, _, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			return errors.New("backend error")
		})

		client := NewStripeClient("sk_test", "whsec_test")
		_, err := client.ConfirmPaymentIntent("pi_123")
		assert.Error(t, err)
	})
}

func TestRefundPayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setupTestBackend(t, func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			assert.Equal(t, "POST", method)
			assert.Equal(t, "/v1/refunds", path)
			if re, ok := v.(*stripe.Refund); ok {
				re.ID = "re_123"
				re.Amount = 500
			}
			return nil
		})

		client := NewStripeClient("sk_test", "whsec_test")
		re, err := client.RefundPayment("pi_123", 500)
		assert.NoError(t, err)
		assert.NotNil(t, re)
		assert.Equal(t, "re_123", re.ID)
	})

	t.Run("backend error", func(t *testing.T) {
		setupTestBackend(t, func(_, _, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			return errors.New("backend error")
		})

		client := NewStripeClient("sk_test", "whsec_test")
		_, err := client.RefundPayment("pi_123", 500)
		assert.Error(t, err)
	})
}

func TestVerifyWebhookSignature(t *testing.T) {
	t.Run("missing secret", func(t *testing.T) {
		client := NewStripeClient("sk_test", "")
		event, err := client.VerifyWebhookSignature([]byte("{}"), "t=1,v1=sig")
		assert.Error(t, err)
		assert.Equal(t, "webhook secret not configured", err.Error())
		assert.Empty(t, event.ID)
	})

	t.Run("invalid signature", func(t *testing.T) {
		client := NewStripeClient("sk_test", "whsec_test")
		event, err := client.VerifyWebhookSignature([]byte("{}"), "invalid_sig")
		assert.Error(t, err)
		assert.Empty(t, event.ID)
	})

	t.Run("valid payload and signature", func(t *testing.T) {
		secret := "whsec_test_secret"
		payload := []byte(fmt.Sprintf(`{"id": "evt_123", "object": "event", "type": "payment_intent.succeeded", "api_version": %q}`, stripe.APIVersion))
		signedPayload := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
			Payload:   payload,
			Secret:    secret,
			Timestamp: time.Now(),
		})

		client := NewStripeClient("sk_test", secret)
		event, err := client.VerifyWebhookSignature(signedPayload.Payload, signedPayload.Header)
		assert.NoError(t, err)
		assert.Equal(t, "evt_123", event.ID)
		assert.Equal(t, stripe.EventType("payment_intent.succeeded"), event.Type)
	})
}
