package stripe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	stripego "github.com/stripe/stripe-go/v81"
)

func TestNewStripeClient(t *testing.T) {
	// Save the original stripe.Key and restore it after the test
	originalKey := stripego.Key
	defer func() {
		stripego.Key = originalKey
	}()

	apiKey := "sk_test_12345"
	webhookSecret := "whsec_12345"

	client := NewStripeClient(apiKey, webhookSecret)

	// Verify stripe.Key was set
	assert.Equal(t, apiKey, stripego.Key)

	// Verify the returned client type and its fields
	stripeClientImpl, ok := client.(*stripeClient)
	assert.True(t, ok, "Expected client to be of type *stripeClient")
	if ok {
		assert.Equal(t, webhookSecret, stripeClientImpl.webhookSecret)
	}
}
