package stripe

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

type mockBackend struct {
	CallFn func(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error
}

func (m *mockBackend) Call(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	if m.CallFn != nil {
		return m.CallFn(method, path, key, params, v)
	}
	return nil
}

func (m *mockBackend) CallStreaming(method, path, key string, params stripe.ParamsContainer, v stripe.StreamingLastResponseSetter) error {
	return nil
}

func (m *mockBackend) CallRaw(method, path, key string, body *form.Values, params *stripe.Params, v stripe.LastResponseSetter) error {
	return nil
}

func (m *mockBackend) CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *stripe.Params, v stripe.LastResponseSetter) error {
	return nil
}

func (m *mockBackend) SetMaxNetworkRetries(maxNetworkRetries int64) {}

func setupMockBackend(t *testing.T, expectedMethod, expectedPath string, setterFunc func(v stripe.LastResponseSetter)) Client {
	mb := &mockBackend{
		CallFn: func(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			assert.Equal(t, expectedMethod, method)
			assert.Equal(t, expectedPath, path)

			if setterFunc != nil {
				setterFunc(v)
			}
			return nil
		},
	}
	originalBackend := stripe.GetBackend(stripe.APIBackend)
	stripe.SetBackend(stripe.APIBackend, mb)
	t.Cleanup(func() {
		stripe.SetBackend(stripe.APIBackend, originalBackend)
	})

	return NewStripeClient("sk_test_123", "whsec_123")
}

func TestCreatePaymentIntent(t *testing.T) {
	client := setupMockBackend(t, "POST", "/v1/payment_intents", func(v stripe.LastResponseSetter) {
		pi := v.(*stripe.PaymentIntent)
		pi.ID = "pi_123"
		pi.Amount = 1000
		pi.Currency = "usd"
	})

	pi, err := client.CreatePaymentIntent(1000, "usd", "test desc", "cus_123")
	require.NoError(t, err)
	assert.Equal(t, "pi_123", pi.ID)
	assert.Equal(t, int64(1000), pi.Amount)
}

func TestCreatePaymentMethod(t *testing.T) {
	client := setupMockBackend(t, "POST", "/v1/payment_methods", func(v stripe.LastResponseSetter) {
		pm := v.(*stripe.PaymentMethod)
		pm.ID = "pm_123"
	})

	pm, err := client.CreatePaymentMethod("4242", "12", "2030", "123")
	require.NoError(t, err)
	assert.Equal(t, "pm_123", pm.ID)

	// Invalid month
	_, err = client.CreatePaymentMethod("4242", "abc", "2030", "123")
	assert.Error(t, err)

	// Invalid year
	_, err = client.CreatePaymentMethod("4242", "12", "abc", "123")
	assert.Error(t, err)
}

func TestCreatePaymentMethodFromToken(t *testing.T) {
	client := setupMockBackend(t, "GET", "/v1/payment_methods/pm_123", func(v stripe.LastResponseSetter) {
		pm := v.(*stripe.PaymentMethod)
		pm.ID = "pm_123"
	})

	pm, err := client.CreatePaymentMethodFromToken("pm_123")
	require.NoError(t, err)
	assert.Equal(t, "pm_123", pm.ID)
}

func TestAttachPaymentMethodToIntent(t *testing.T) {
	client := setupMockBackend(t, "POST", "/v1/payment_intents/pi_123", func(v stripe.LastResponseSetter) {
		pi := v.(*stripe.PaymentIntent)
		pi.ID = "pi_123"
	})

	err := client.AttachPaymentMethodToIntent("pm_123", "pi_123")
	require.NoError(t, err)
}

func TestConfirmPaymentIntent(t *testing.T) {
	client := setupMockBackend(t, "POST", "/v1/payment_intents/pi_123/confirm", func(v stripe.LastResponseSetter) {
		pi := v.(*stripe.PaymentIntent)
		pi.ID = "pi_123"
	})

	pi, err := client.ConfirmPaymentIntent("pi_123")
	require.NoError(t, err)
	assert.Equal(t, "pi_123", pi.ID)
}

func TestRefundPayment(t *testing.T) {
	client := setupMockBackend(t, "POST", "/v1/refunds", func(v stripe.LastResponseSetter) {
		r := v.(*stripe.Refund)
		r.ID = "re_123"
	})

	r, err := client.RefundPayment("pi_123", 1000)
	require.NoError(t, err)
	assert.Equal(t, "re_123", r.ID)
}

func TestVerifyWebhookSignature(t *testing.T) {
	client := NewStripeClient("sk_test_123", "whsec_123")

	// test empty secret
	clientEmpty := NewStripeClient("sk_test_123", "")
	_, err := clientEmpty.VerifyWebhookSignature([]byte("payload"), "sig")
	assert.Error(t, err)
	assert.Equal(t, "webhook secret not configured", err.Error())

	// test invalid sig
	_, err = client.VerifyWebhookSignature([]byte("payload"), "sig")
	assert.Error(t, err)

	// test valid sig
	// Stripe requires the API version in the payload to match its configured version unless ignored.
	payload := []byte(fmt.Sprintf(`{"id":"evt_123","type":"payment_intent.succeeded","api_version":"%s"}`, stripe.APIVersion))
	now := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte("whsec_123"))
	mac.Write([]byte(fmt.Sprintf("%d.%s", now, payload)))
	sig := hex.EncodeToString(mac.Sum(nil))

	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	evt, err := client.VerifyWebhookSignature(payload, header)
	require.NoError(t, err)
	assert.Equal(t, "evt_123", evt.ID)
}
