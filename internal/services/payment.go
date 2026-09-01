package service

import (
	"context"
	"time"

	apperrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, req *models.PaymentRequest) (*models.PaymentResponse, error)
	GetPaymentByID(ctx context.Context, id string) (*models.Payment, error)
	ListPaymentsByCustomer(ctx context.Context, customerID string, page, size int) ([]*models.Payment, int, error)
	ProcessWebhook(ctx context.Context, payload []byte, signature string) (stripe.Event, error)
}

type paymentService struct {
	repo         repository.PaymentRepository
	stripeClient stripe.Client
}

func NewPaymentService(repo repository.PaymentRepository, stripeClient stripe.Client) PaymentService {
	return &paymentService{repo: repo, stripeClient: stripeClient}
}

// CreatePayment implements PaymentService.
func (s *paymentService) CreatePayment(ctx context.Context, req *models.PaymentRequest) (*models.PaymentResponse, error) {
	// new request for payment
	paymentIntent, err := s.stripeClient.CreatePaymentIntent(
		req.Amount, req.Currency, req.Description, req.CustomerID)
	if err != nil {
		return nil, apperrors.ThirdPartyError("Failed to create payment intent").WithError(err)
	}

	// create a payment method & attach it to paymentIntent
	if req.PaymentMethod == "card" {
		paymentMethod, err := s.stripeClient.CreatePaymentMethodFromToken(req.Token)
		if err != nil {
			return nil, apperrors.ThirdPartyError("Failed to create payment method").WithError(err)
		}

		err = s.stripeClient.AttachPaymentMethodToIntent(paymentMethod.ID, paymentIntent.ID)
		if err != nil {
			return nil, apperrors.ThirdPartyError("Failed to attach payment method").WithError(err)
		}
	}

	// store the payment in the database
	payment := &models.Payment{
		ID:            paymentIntent.ID,
		CustomerID:    req.CustomerID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Description:   req.Description,
		Status:        models.PaymentStatusPending,
		PaymentMethod: req.PaymentMethod,
		StripeID:      paymentIntent.ID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, apperrors.DatabaseError("Failed to record payment").WithError(err)
	}

	return &models.PaymentResponse{
		Payment:       payment,
		ClientSecret:  paymentIntent.ClientSecret,
		PaymentStatus: string(payment.Status),
		Message:       "Payment initiated successfully.",
	}, nil
}

// GetPaymentByID implements PaymentService.
func (s *paymentService) GetPaymentByID(ctx context.Context, id string) (*models.Payment, error) {
	payment, err := s.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, apperrors.DatabaseError("Payment not found").WithError(err)
	}

	return payment, nil
}

// ListPaymentsByCustomer implements PaymentService.
func (s *paymentService) ListPaymentsByCustomer(ctx context.Context, customerID string, page, size int) ([]*models.Payment, int, error) {
	payments, total, err := s.repo.ListPaymentsOfCustomer(ctx, customerID, page, size)
	if err != nil {
		return nil, 0, apperrors.DatabaseError("Failed to fetch payments").WithError(err)
	}

	return payments, total, nil
}

func extractPaymentIntentID(data map[string]any, key string) (string, error) {
	val, ok := data[key]
	if !ok {
		return "", apperrors.InternalError("Payment intent ID not found in Stripe response")
	}

	strVal, ok := val.(string)
	if !ok {
		return "", apperrors.InternalError("Payment intent ID is not a string in Stripe response")
	}

	if strVal == "" {
		return "", apperrors.ThirdPartyError("Missing payment intent ID in webhook")
	}

	return strVal, nil
}

func (s *paymentService) handlePaymentIntentEvent(ctx context.Context, obj map[string]any, status models.PaymentStatus) error {
	stripeID, err := extractPaymentIntentID(obj, "id")
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePaymentStatus(ctx, stripeID, status); err != nil {
		return apperrors.DatabaseError("Failed to update payment status").WithError(err)
	}

	return nil
}

func (s *paymentService) handleChargeRefunded(ctx context.Context, obj map[string]any) error {
	paymentIntentID, ok := obj["payment_intent"].(string)
	if !ok || paymentIntentID == "" {
		return apperrors.ThirdPartyError("Missing payment intent ID in webhook")
	}

	if err := s.repo.UpdatePaymentStatus(ctx, paymentIntentID, models.PaymentStatusRefunded); err != nil {
		return apperrors.DatabaseError("Failed to update payment status").WithError(err)
	}

	return nil
}

// ProcessWebhook implements PaymentService.
func (s *paymentService) ProcessWebhook(ctx context.Context, payload []byte, signature string) (stripe.Event, error) {
	event, err := s.stripeClient.VerifyWebhookSignature(payload, signature)
	if err != nil {
		return stripe.Event{}, apperrors.ThirdPartyError("Webhook signature verification failed").WithError(err)
	}

	var processErr error
	switch event.Type {
	case "payment_intent.succeeded":
		processErr = s.handlePaymentIntentEvent(ctx, event.Data.Object, models.PaymentStatusSucceeded)
	case "payment_intent.payment_failed":
		processErr = s.handlePaymentIntentEvent(ctx, event.Data.Object, models.PaymentStatusFailed)
	case "charge.refunded":
		processErr = s.handleChargeRefunded(ctx, event.Data.Object)
	}

	if processErr != nil {
		return event, processErr
	}

	return event, nil
}
