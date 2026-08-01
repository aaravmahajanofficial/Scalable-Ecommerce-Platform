package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	stripeClient "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/stripe/stripe-go/v81"
)

// MockStripeClient implements stripeClient.Client for testing
type MockStripeClient struct{}

func (m *MockStripeClient) CreatePaymentIntent(amount int64, currency string, description string, customerID string) (*stripe.PaymentIntent, error) {
	return nil, nil
}
func (m *MockStripeClient) CreatePaymentMethod(cardNumber, cardExpMonth, cardExpYear, cardCVC string) (*stripe.PaymentMethod, error) {
	return nil, nil
}
func (m *MockStripeClient) CreatePaymentMethodFromToken(paymentMethodID string) (*stripe.PaymentMethod, error) {
	return nil, nil
}
func (m *MockStripeClient) AttachPaymentMethodToIntent(paymentMethodID, paymentIntentID string) error {
	return nil
}
func (m *MockStripeClient) ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	return nil, nil
}
func (m *MockStripeClient) RefundPayment(paymentIntentID string, amount int64) (*stripe.Refund, error) {
	return nil, nil
}
func (m *MockStripeClient) VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error) {
	return stripe.Event{}, nil
}

func TestNewLivenessHandler(t *testing.T) {
	handler := NewLivenessHandler()

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "Service is alive"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want it to contain %v",
			rr.Body.String(), expected)
	}
}

func TestNewReadinessHandler(t *testing.T) {
	// Provide invalid DSNs so it attempts to connect quickly or fails,
	// or provide standard ones if health-go handles it.
	cfg := &config.Config{
		Database: config.Database{
			Host: "127.0.0.1",
			Port: "5432",
			User: "user",
			Password: "password",
			Name: "testdb",
			SSLMode: "disable",
		},
		RedisConnect: config.RedisConnect{
			Host: "127.0.0.1",
			Port: "6379",
		},
		OTel: config.OTelConfig{
			ServiceName: "test-service",
		},
	}

	// We set a stripe key to bypass the 'invalid API key' error output from stripe lib
	// although balance.Get will still fail due to network/unauthorized if actually hit.
	stripe.Key = "sk_test_mock"

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	redisClient, _ := redismock.NewClientMock()

	var mockStripe stripeClient.Client = &MockStripeClient{}

	he := &HealthEndpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: &mockStripe,
	}

	handler, err := NewReadinessHandler(cfg, he)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if handler == nil {
		t.Fatalf("expected non-nil handler")
	}

	req, err := http.NewRequest("GET", "/ready", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Since we are mocking connections but health-go uses the DSN string,
	// health-go will fail to connect (returning 503). That is completely fine
	// as long as the handler executes properly and formats the JSON response.
	if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", rr.Code)
	}

	// We should be able to unmarshal the response
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("expected valid JSON response, got error: %v", err)
	}

	if status, ok := response["status"]; ok {
		if status != "OK" && status != "Partially Available" && status != "Unavailable" {
			t.Errorf("unexpected status string in response: %v", status)
		}
	} else {
		t.Errorf("response JSON missing 'status' field")
	}
}
