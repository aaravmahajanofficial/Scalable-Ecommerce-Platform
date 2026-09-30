package service_test

import (
	"errors"
	"testing"
	"time"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repoMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	stripeMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v86"
)

func setupPaymentServiceTest(t *testing.T) (service.PaymentService, *repoMocks.MockPaymentRepository, *stripeMocks.MockClient) {
	t.Helper()
	mockRepo := repoMocks.NewMockPaymentRepository(t)
	mockStripeClient := stripeMocks.NewMockClient(t)
	paymentService := service.NewPaymentService(mockRepo, mockStripeClient)
	return paymentService, mockRepo, mockStripeClient
}

func assertAppError(t *testing.T, err error, expectedCode string) *appErrors.AppError {
	t.Helper()
	assert.Error(t, err)
	appErr, ok := appErrors.IsAppError(err)
	assert.True(t, ok)
	assert.Equal(t, expectedCode, appErr.Code)
	return appErr
}

func TestNewPaymentService(t *testing.T) {
	svc, _, _ := setupPaymentServiceTest(t)
	assert.NotNil(t, svc)
}

func TestCreatePayment(t *testing.T) {
	ctx := t.Context()

	testUserID := uuid.New().String()
	testPaymentIntentID := "pi_123"
	testPaymentMethodID := "pm_456"
	testClientSecret := "pi_123_secret_abc"

	reqCard := &models.PaymentRequest{
		CustomerID:    testUserID,
		Amount:        1000,
		Currency:      "usd",
		Description:   "Test Card Payment",
		PaymentMethod: "card",
		Token:         "tok_visa",
	}

	reqOther := &models.PaymentRequest{
		CustomerID:    testUserID,
		Amount:        2000,
		Currency:      "eur",
		Description:   "Test Other Payment",
		PaymentMethod: "ideal",
	}

	mockPaymentIntent := &stripe.PaymentIntent{
		ID:           testPaymentIntentID,
		Amount:       reqCard.Amount,
		Currency:     stripe.Currency(reqCard.Currency),
		Description:  reqCard.Description,
		ClientSecret: testClientSecret,
		Status:       stripe.PaymentIntentStatusRequiresPaymentMethod,
	}

	mockPaymentMethod := &stripe.PaymentMethod{
		ID:   testPaymentMethodID,
		Type: stripe.PaymentMethodTypeCard,
	}

	expectedPayment := &models.Payment{
		ID:            testPaymentIntentID,
		CustomerID:    reqCard.CustomerID,
		Amount:        reqCard.Amount,
		Currency:      reqCard.Currency,
		Description:   reqCard.Description,
		Status:        models.PaymentStatusPending,
		PaymentMethod: reqCard.PaymentMethod,
		StripeID:      testPaymentIntentID,
	}

	t.Run("Success - Card Payment", func(t *testing.T) {
		paymentService, mockRepo, mockStripeClient := setupPaymentServiceTest(t)

		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(mockPaymentIntent, nil).Once()
		mockStripeClient.On("CreatePaymentMethodFromToken", reqCard.Token).Return(mockPaymentMethod, nil).Once()
		mockStripeClient.On("AttachPaymentMethodToIntent", mockPaymentMethod.ID, mockPaymentIntent.ID).Return(nil).Once()
		mockRepo.On("CreatePayment", ctx, mock.MatchedBy(func(p *models.Payment) bool {
			return p.ID == testPaymentIntentID && p.CustomerID == reqCard.CustomerID && p.Amount == reqCard.Amount
		})).Return(nil).Once()

		resp, err := paymentService.CreatePayment(ctx, reqCard)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, testClientSecret, resp.ClientSecret)
		assert.Equal(t, string(models.PaymentStatusPending), resp.PaymentStatus)
		assert.NotNil(t, resp.Payment)
		assert.Equal(t, expectedPayment.ID, resp.Payment.ID)
		assert.Equal(t, expectedPayment.CustomerID, resp.Payment.CustomerID)
		assert.Equal(t, expectedPayment.Amount, resp.Payment.Amount)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Success - Non-Card Payment", func(t *testing.T) {
		paymentService, mockRepo, mockStripeClient := setupPaymentServiceTest(t)

		mockPaymentIntentOther := &stripe.PaymentIntent{
			ID:           "pi_789",
			Amount:       reqOther.Amount,
			Currency:     stripe.Currency(reqOther.Currency),
			Description:  reqOther.Description,
			ClientSecret: "pi_789_secret_def",
			Status:       stripe.PaymentIntentStatusRequiresPaymentMethod,
		}

		mockStripeClient.On("CreatePaymentIntent", reqOther.Amount, reqOther.Currency, reqOther.Description, reqOther.CustomerID).Return(mockPaymentIntentOther, nil).Once()
		mockRepo.On("CreatePayment", ctx, mock.MatchedBy(func(p *models.Payment) bool {
			return p.ID == mockPaymentIntentOther.ID && p.CustomerID == reqOther.CustomerID && p.Amount == reqOther.Amount
		})).Return(nil).Once()

		resp, err := paymentService.CreatePayment(ctx, reqOther)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mockPaymentIntentOther.ClientSecret, resp.ClientSecret)
		assert.Equal(t, string(models.PaymentStatusPending), resp.PaymentStatus)
		assert.NotNil(t, resp.Payment)
		assert.Equal(t, mockPaymentIntentOther.ID, resp.Payment.ID)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
		mockStripeClient.AssertNotCalled(t, "CreatePaymentMethodFromToken")
		mockStripeClient.AssertNotCalled(t, "AttachPaymentMethodToIntent")
	})

	t.Run("Failure - CreatePaymentIntent Fails", func(t *testing.T) {
		paymentService, mockRepo, mockStripeClient := setupPaymentServiceTest(t)

		stripeErr := errors.New("stripe API error")
		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(nil, stripeErr).Once()

		resp, err := paymentService.CreatePayment(ctx, reqCard)

		assert.Nil(t, resp)
		assertAppError(t, err, appErrors.ErrCodeThirdPartyError)
		assert.ErrorIs(t, err, stripeErr)

		mockRepo.AssertNotCalled(t, "CreatePayment")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - CreatePaymentMethodFromToken Fails", func(t *testing.T) {
		paymentService, mockRepo, mockStripeClient := setupPaymentServiceTest(t)

		stripeErr := errors.New("stripe token error")
		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(mockPaymentIntent, nil).Once()
		mockStripeClient.On("CreatePaymentMethodFromToken", reqCard.Token).Return(nil, stripeErr).Once()

		resp, err := paymentService.CreatePayment(ctx, reqCard)

		assert.Nil(t, resp)
		assertAppError(t, err, appErrors.ErrCodeThirdPartyError)
		assert.ErrorIs(t, err, stripeErr)

		mockRepo.AssertNotCalled(t, "CreatePayment")
		mockStripeClient.AssertExpectations(t)
		mockStripeClient.AssertNotCalled(t, "AttachPaymentMethodToIntent")
	})

	t.Run("Failure - AttachPaymentMethodToIntent Fails", func(t *testing.T) {
		paymentService, mockRepo, mockStripeClient := setupPaymentServiceTest(t)

		stripeErr := errors.New("stripe attach error")
		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(mockPaymentIntent, nil).Once()
		mockStripeClient.On("CreatePaymentMethodFromToken", reqCard.Token).Return(mockPaymentMethod, nil).Once()
		mockStripeClient.On("AttachPaymentMethodToIntent", mockPaymentMethod.ID, mockPaymentIntent.ID).Return(stripeErr).Once()

		resp, err := paymentService.CreatePayment(ctx, reqCard)

		assert.Nil(t, resp)
		assertAppError(t, err, appErrors.ErrCodeThirdPartyError)
		assert.ErrorIs(t, err, stripeErr)

		mockRepo.AssertNotCalled(t, "CreatePayment")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - Repository CreatePayment Fails", func(t *testing.T) {
		paymentService, mockRepo, mockStripeClient := setupPaymentServiceTest(t)

		dbErr := errors.New("database insert error")
		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(mockPaymentIntent, nil).Once()
		mockStripeClient.On("CreatePaymentMethodFromToken", reqCard.Token).Return(mockPaymentMethod, nil).Once()
		mockStripeClient.On("AttachPaymentMethodToIntent", mockPaymentMethod.ID, mockPaymentIntent.ID).Return(nil).Once()
		mockRepo.On("CreatePayment", ctx, mock.AnythingOfType("*models.Payment")).Return(dbErr).Once()

		resp, err := paymentService.CreatePayment(ctx, reqCard)

		assert.Nil(t, resp)
		assertAppError(t, err, appErrors.ErrCodeDatabaseError)
		assert.ErrorIs(t, err, dbErr)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})
}

func TestGetPaymentByID(t *testing.T) {
	ctx := t.Context()

	testPaymentID := uuid.New().String()
	expectedPayment := &models.Payment{
		ID:         testPaymentID,
		CustomerID: uuid.New().String(),
		Amount:     500,
		Currency:   "gbp",
		Status:     models.PaymentStatusSucceeded,
		CreatedAt:  time.Now().Add(-time.Hour),
		UpdatedAt:  time.Now(),
	}

	t.Run("Success", func(t *testing.T) {
		paymentService, mockRepo, _ := setupPaymentServiceTest(t)
		mockRepo.On("GetPaymentByID", ctx, testPaymentID).Return(expectedPayment, nil).Once()

		payment, err := paymentService.GetPaymentByID(ctx, testPaymentID)

		assert.NoError(t, err)
		assert.Equal(t, expectedPayment, payment)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Repository Error", func(t *testing.T) {
		paymentService, mockRepo, _ := setupPaymentServiceTest(t)
		repoErr := errors.New("payment not found in DB")
		mockRepo.On("GetPaymentByID", ctx, testPaymentID).Return(nil, repoErr).Once()

		payment, err := paymentService.GetPaymentByID(ctx, testPaymentID)

		assert.Nil(t, payment)
		assertAppError(t, err, appErrors.ErrCodeDatabaseError)
		assert.ErrorIs(t, err, repoErr)
		mockRepo.AssertExpectations(t)
	})
}

func TestListPaymentsByCustomer(t *testing.T) {
	ctx := t.Context()

	testCustomerID := uuid.New().String()
	page, size, expectedTotal := 1, 10, 5
	expectedPayments := []*models.Payment{
		{ID: uuid.New().String(), CustomerID: testCustomerID, Amount: 100},
		{ID: uuid.New().String(), CustomerID: testCustomerID, Amount: 200},
	}

	t.Run("Success", func(t *testing.T) {
		paymentService, mockRepo, _ := setupPaymentServiceTest(t)
		mockRepo.On("ListPaymentsOfCustomer", ctx, testCustomerID, page, size).Return(expectedPayments, expectedTotal, nil).Once()

		payments, total, err := paymentService.ListPaymentsByCustomer(ctx, testCustomerID, page, size)

		assert.NoError(t, err)
		assert.Equal(t, expectedPayments, payments)
		assert.Equal(t, expectedTotal, total)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Repository Error", func(t *testing.T) {
		paymentService, mockRepo, _ := setupPaymentServiceTest(t)
		repoErr := errors.New("failed to query payments")
		mockRepo.On("ListPaymentsOfCustomer", ctx, testCustomerID, page, size).Return(nil, 0, repoErr).Once()

		payments, total, err := paymentService.ListPaymentsByCustomer(ctx, testCustomerID, page, size)

		assert.Nil(t, payments)
		assert.Equal(t, 0, total)
		assertAppError(t, err, appErrors.ErrCodeDatabaseError)
		assert.ErrorIs(t, err, repoErr)
		mockRepo.AssertExpectations(t)
	})
}

func TestProcessWebhook(t *testing.T) {
	ctx := t.Context()

	payload := []byte(`{"id": "evt_123", "type": "payment_intent.succeeded", "data": {"object": {"id": "pi_abc"}}}`)
	signature := "whsec_sig"
	stripePaymentIntentID := "pi_abc"

	eventSucceeded := stripe.Event{
		ID:   "evt_123",
		Type: "payment_intent.succeeded",
		Data: &stripe.EventData{Object: map[string]any{"id": stripePaymentIntentID}},
	}
	eventFailed := stripe.Event{
		ID:   "evt_456",
		Type: "payment_intent.payment_failed",
		Data: &stripe.EventData{Object: map[string]any{"id": stripePaymentIntentID}},
	}
	eventRefunded := stripe.Event{
		ID:   "evt_789",
		Type: "charge.refunded",
		Data: &stripe.EventData{Object: map[string]any{"id": "ch_xyz", "payment_intent": stripePaymentIntentID}},
	}

	t.Run("Success Scenarios", func(t *testing.T) {
		successTests := []struct {
			name           string
			payload        []byte
			event          stripe.Event
			expectedStatus models.PaymentStatus
			updateCalled   bool
		}{
			{
				name:           "payment_intent.succeeded",
				payload:        payload,
				event:          eventSucceeded,
				expectedStatus: models.PaymentStatusSucceeded,
				updateCalled:   true,
			},
			{
				name:           "payment_intent.payment_failed",
				payload:        []byte(`{"id": "evt_456", "type": "payment_intent.payment_failed"}`),
				event:          eventFailed,
				expectedStatus: models.PaymentStatusFailed,
				updateCalled:   true,
			},
			{
				name:           "charge.refunded",
				payload:        []byte(`{"id": "evt_789", "type": "charge.refunded"}`),
				event:          eventRefunded,
				expectedStatus: models.PaymentStatusRefunded,
				updateCalled:   true,
			},
			{
				name:    "Unhandled Event Type",
				payload: []byte(`{"id": "evt_000", "type": "customer.created"}`),
				event: stripe.Event{
					ID:   "evt_000",
					Type: "customer.created",
					Data: &stripe.EventData{Object: map[string]any{"id": "cus_123"}},
				},
				updateCalled: false,
			},
		}

		for _, tt := range successTests {
			t.Run(tt.name, func(t *testing.T) {
				paymentService, mockRepo, mockStripeClient := setupPaymentServiceTest(t)
				mockStripeClient.On("VerifyWebhookSignature", tt.payload, signature).Return(tt.event, nil).Once()
				if tt.updateCalled {
					mockRepo.On("UpdatePaymentStatus", ctx, stripePaymentIntentID, tt.expectedStatus).Return(nil).Once()
				}

				event, err := paymentService.ProcessWebhook(ctx, tt.payload, signature)

				assert.NoError(t, err)
				assert.Equal(t, tt.event.ID, event.ID)
				mockRepo.AssertExpectations(t)
				mockStripeClient.AssertExpectations(t)
			})
		}
	})

	t.Run("Failure Scenarios", func(t *testing.T) {
		verifyErr := errors.New("invalid signature")
		dbErr := errors.New("db update failed")

		failureTests := []struct {
			name         string
			payload      []byte
			event        stripe.Event
			verifyErr    error
			expectUpdate bool
			updateStatus models.PaymentStatus
			updateErr    error
			expectedCode string
			errorContains string
		}{
			{
				name:         "VerifyWebhookSignature Fails",
				payload:      payload,
				event:        stripe.Event{},
				verifyErr:    verifyErr,
				expectedCode: appErrors.ErrCodeThirdPartyError,
			},
			{
				name:    "Missing Payment Intent ID (Succeeded)",
				payload: []byte(`{"id": "evt_bad", "type": "payment_intent.succeeded"}`),
				event: stripe.Event{
					ID:   "evt_bad",
					Type: "payment_intent.succeeded",
					Data: &stripe.EventData{Object: map[string]any{"amount": 1000}},
				},
				expectedCode:  appErrors.ErrCodeInternal,
				errorContains: "Payment intent ID not found",
			},
			{
				name:    "Non-String Payment Intent ID (Succeeded)",
				payload: []byte(`{"id": "evt_non_string", "type": "payment_intent.succeeded"}`),
				event: stripe.Event{
					ID:   "evt_non_string",
					Type: "payment_intent.succeeded",
					Data: &stripe.EventData{Object: map[string]any{"id": 12345}},
				},
				expectedCode:  appErrors.ErrCodeInternal,
				errorContains: "Payment intent ID is not a string",
			},
			{
				name:    "Empty String Payment Intent ID (Succeeded)",
				payload: []byte(`{"id": "evt_empty_string", "type": "payment_intent.succeeded"}`),
				event: stripe.Event{
					ID:   "evt_empty_string",
					Type: "payment_intent.succeeded",
					Data: &stripe.EventData{Object: map[string]any{"id": ""}},
				},
				expectedCode:  appErrors.ErrCodeThirdPartyError,
				errorContains: "Missing payment intent ID in webhook",
			},
			{
				name:         "UpdatePaymentStatus Fails (Succeeded)",
				payload:      payload,
				event:        eventSucceeded,
				expectUpdate: true,
				updateStatus: models.PaymentStatusSucceeded,
				updateErr:    dbErr,
				expectedCode: appErrors.ErrCodeDatabaseError,
			},
			{
				name:    "Missing Payment Intent ID (Failed)",
				payload: []byte(`{"id": "evt_bad_fail", "type": "payment_intent.payment_failed"}`),
				event: stripe.Event{
					ID:   "evt_bad_fail",
					Type: "payment_intent.payment_failed",
					Data: &stripe.EventData{Object: map[string]any{"reason": "card_declined"}},
				},
				expectedCode:  appErrors.ErrCodeInternal,
				errorContains: "Payment intent ID not found",
			},
			{
				name:         "UpdatePaymentStatus Fails (Failed)",
				payload:      []byte(`{"id": "evt_456", "type": "payment_intent.payment_failed"}`),
				event:        eventFailed,
				expectUpdate: true,
				updateStatus: models.PaymentStatusFailed,
				updateErr:    dbErr,
				expectedCode: appErrors.ErrCodeDatabaseError,
			},
			{
				name:    "Missing Payment Intent ID (Refunded)",
				payload: []byte(`{"id": "evt_bad_refund", "type": "charge.refunded"}`),
				event: stripe.Event{
					ID:   "evt_bad_refund",
					Type: "charge.refunded",
					Data: &stripe.EventData{Object: map[string]any{"id": "ch_xyz"}},
				},
				expectedCode:  appErrors.ErrCodeThirdPartyError,
				errorContains: "Missing payment intent ID",
			},
			{
				name:         "UpdatePaymentStatus Fails (Refunded)",
				payload:      []byte(`{"id": "evt_789", "type": "charge.refunded"}`),
				event:        eventRefunded,
				expectUpdate: true,
				updateStatus: models.PaymentStatusRefunded,
				updateErr:    dbErr,
				expectedCode: appErrors.ErrCodeDatabaseError,
			},
		}

		for _, tt := range failureTests {
			t.Run(tt.name, func(t *testing.T) {
				paymentService, mockRepo, mockStripeClient := setupPaymentServiceTest(t)
				mockStripeClient.On("VerifyWebhookSignature", tt.payload, signature).Return(tt.event, tt.verifyErr).Once()
				if tt.expectUpdate {
					mockRepo.On("UpdatePaymentStatus", ctx, stripePaymentIntentID, tt.updateStatus).Return(tt.updateErr).Once()
				}

				event, err := paymentService.ProcessWebhook(ctx, tt.payload, signature)

				assert.Equal(t, tt.event.ID, event.ID)
				_ = assertAppError(t, err, tt.expectedCode)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				if tt.verifyErr != nil {
					assert.ErrorIs(t, err, tt.verifyErr)
				}
				if tt.updateErr != nil {
					assert.ErrorIs(t, err, tt.updateErr)
				}

				if !tt.expectUpdate {
					mockRepo.AssertNotCalled(t, "UpdatePaymentStatus")
				}
				mockRepo.AssertExpectations(t)
				mockStripeClient.AssertExpectations(t)
			})
		}
	})
}
