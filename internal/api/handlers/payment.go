package handlers

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type PaymentHandler struct {
	paymentService service.PaymentService
	validator      *validator.Validate
}

func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService, validator: validator.New()}
}

// CreatePayment godoc
//
//	@Summary		Initiate a payment for an order
//	@Description	Creates a payment intent (e.g., with Stripe) for a specified order and returns the client secret needed for frontend processing. Requires authentication.
//	@Tags			Payments
//	@Accept			json
//	@Produce		json
//	@Param			payment	body		models.PaymentRequest	true	"Payment Request Details (Order ID, Amount, Currency, Customer ID)"
//	@Success		200		{object}	models.PaymentResponse	"Successfully initiated payment, includes client secret"
//	@Failure		400		{object}	response.ErrorResponse	"Validation error or invalid input"
//	@Failure		401		{object}	response.ErrorResponse	"Authentication required"
//	@Failure		403		{object}	response.ErrorResponse	"Forbidden - Attempting to pay for another user's order"
//	@Failure		404		{object}	response.ErrorResponse	"Order not found or already paid"
//	@Failure		500		{object}	response.ErrorResponse	"Internal server error or payment provider error"
//	@Security		BearerAuth
//	@Router			/payments [post]
func (h *PaymentHandler) CreatePayment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized payment creation attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))

			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))

		// Decode the request body
		var req models.PaymentRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			logger.Warn("Invalid create payment input")

			return
		}

		logger = logger.With(slog.String("customerID", req.CustomerID), slog.Int64("amount", req.Amount))

		if req.CustomerID != claims.UserID.String() {
			logger.Warn("User attempted to create payment for another customer ID",
				slog.String("requesterId", claims.UserID.String()),
				slog.String("requestedCustomerID", req.CustomerID))
			response.Error(w, errors.ForbiddenError("You can only make payments for your own orders"))

			return
		}

		// Call the payment service
		payment, err := h.paymentService.CreatePayment(r.Context(), &req)
		if err != nil {
			logger.Error("Failed to initiate payment", slog.Any("error", err))
			response.Error(w, err)

			return
		}

		logger.Info("Payment initiated successfully",
			slog.String("paymentIntentId", payment.ClientSecret),
			slog.String("paymentDBId", payment.Payment.ID))
		response.Success(w, http.StatusOK, payment)
	}
}

// GetPayment godoc
//
//	@Summary		Get payment details by ID
//	@Description	Retrieves details for a specific payment record. Requires authentication. (Authorization might be needed to restrict access).
//	@Tags			Payments
//	@Produce		json
//	@Param			id	path		string					true	"Payment ID (Internal DB ID)"
//	@Success		200	{object}	models.Payment			"Successfully retrieved payment details"
//	@Failure		400	{object}	response.ErrorResponse	"Invalid or missing payment ID"
//	@Failure		401	{object}	response.ErrorResponse	"Authentication required"
//	@Failure		403	{object}	response.ErrorResponse	"Forbidden - User cannot access this payment"	//	Add	if	authz	implemented
//	@Failure		404	{object}	response.ErrorResponse	"Payment not found"
//	@Failure		500	{object}	response.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/payments/{id} [get]
func (h *PaymentHandler) GetPayment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized payment get attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))

			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))

		idStr := r.PathValue("id")
		if idStr == "" {
			logger.Warn("Missing payment ID in path")
			response.Error(w, errors.BadRequestError("Payment ID is required"))

			return
		}

		logger = logger.With(slog.String("paymentId", idStr))

		// Call the service
		payment, err := h.paymentService.GetPaymentByID(r.Context(), idStr)
		if err != nil {
			logger.Error("Failed to get payment details", slog.Any("error", err))
			response.Error(w, err)

			return
		}

		logger.Info("Payment details retrieved successfully")
		response.Success(w, http.StatusOK, payment)
	}
}

// ListPayments godoc
//
//	@Summary		List user's payments with pagination
//	@Description	Retrieves a paginated list of payment records for the authenticated user. Requires authentication.
//	@Tags			Payments
//	@Produce		json
//	@Param			page		query		int												false	"Page number for pagination (default: 1)"			minimum(1)
//	@Param			pageSize	query		int												false	"Number of items per page (default: 10, max: 100)"	minimum(1)	maximum(100)
//	@Success		200			{object}	models.PaginatedResponse{Data=[]models.Payment}	"Successfully retrieved list of payments"
//	@Failure		401			{object}	response.ErrorResponse							"Authentication required"
//	@Failure		500			{object}	response.ErrorResponse							"Internal server error"
//	@Security		BearerAuth
//	@Router			/payments [get]
func (h *PaymentHandler) ListPayments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized payment list attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))

			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}

		pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}

		logger = logger.With(slog.Int("page", page), slog.Int("pageSize", pageSize))

		// Call the service
		payments, total, err := h.paymentService.ListPaymentsByCustomer(r.Context(), claims.UserID.String(), page, pageSize)
		if err != nil {
			logger.Error("Failed to list user payments", slog.Any("error", err))
			response.Error(w, err)

			return
		}

		logger.Info("Payments listed successfully", slog.Int("count", len(payments)), slog.Int("total", total))
		response.Success(w, http.StatusOK, models.PaginatedResponse{
			Data:     payments,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		})
	}
}

// HandleStripeWebhook godoc
//
//	@Summary		Handle incoming Stripe webhooks
//	@Description	Receives and processes webhook events from Stripe (e.g., payment success, failure) to update internal payment and order statuses. This endpoint should not require application-level authentication but relies on Stripe's signature verification.
//	@Tags			Payments (Internal)
//	@Accept			json
//	@Produce		json
//	@Param			Stripe-Signature	header		string					true				"Stripe webhook signature for verification"
//	@Param			payload				body		object					true				"Raw Stripe event payload (JSON)"
//	@Success		200					{object}	map[string]bool			`{"success": true}`	"Webhook received and processed successfully"
//	@Failure		400					{object}	response.ErrorResponse	"Bad request (e.g., missing signature, failed reading body, invalid payload)"
//	@Failure		401					{object}	response.ErrorResponse	"Webhook signature verification failed"
//	@Failure		500					{object}	response.ErrorResponse	"Internal server error during webhook processing"
//	@Router			/payments/webhook [post]
func (h *PaymentHandler) HandleStripeWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.LoggerFromContext(r.Context())

		// read the payload/body

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Error reading webhook body", slog.Any("error", err))
			response.Error(w, errors.BadRequestError("Failed to read request body"))

			return
		}

		signature := r.Header.Get("Stripe-Signature")
		if signature == "" {
			logger.Error("Missing Stripe signature in webhook request")
			response.Error(w, errors.BadRequestError("Stripe Signature is required"))

			return
		}

		// Call the service
		event, err := h.paymentService.ProcessWebhook(r.Context(), payload, signature)
		if err != nil {
			logger.Error("Failed to process payment webhook", slog.Any("error", err))
			response.Error(w, err)

			return
		}

		logger = logger.With(slog.String("stripeEventId", event.ID), slog.Any("stripeEventType", event.Type))
		logger.Info("Payment webhook processed successfully")
		response.Success(w, http.StatusOK, map[string]bool{"success": true})
	}
}
