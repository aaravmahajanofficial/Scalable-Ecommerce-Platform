// Package apputils provides shared application utilities.
package apputils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	apperrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func DecodeJSONBody(r *http.Request, dest any) error {
	logger := middleware.LoggerFromContext(r.Context())
	defer func() {
		if err := r.Body.Close(); err != nil {
			logger.Error("Failed to close request body", slog.Any("error", err))
		}
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body", slog.Any("error", err))

		return apperrors.BadRequestError("Failed to read request body").WithError(err)
	}

	if len(body) == 0 {
		logger.Warn("Empty request body received")

		return apperrors.BadRequestError("Request body cannot be empty").WithError(err)
	}

	if err := json.Unmarshal(body, dest); err != nil {
		logger.Error("Failed to parse request JSON", slog.Any("error", err))

		return apperrors.BadRequestError("Invalid JSON format").WithError(err)
	}

	return nil
}

func ValidateStruct(ctx context.Context, validate *validator.Validate, data any) error {
	logger := middleware.LoggerFromContext(ctx)

	if err := validate.Struct(data); err != nil {
		if validationErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
			logger.Warn("User input validation failed", slog.String("error", validationErrs.Error()))

			var details []string
			for _, verr := range validationErrs {
				details = append(details, formatValidationError(verr))
			}

			return apperrors.ValidationError("Validation Failed").WithDetail(fmt.Sprintf("%v", details))
		}

		logger.Error("Unexpected validation error", slog.String("error", err.Error()))

		return apperrors.InternalError("Unexpected validation error").WithError(err)
	}

	return nil
}

func ParseAndValidate(r *http.Request, w http.ResponseWriter, dest any, validate *validator.Validate) bool {
	if err := DecodeJSONBody(r, dest); err != nil {
		response.Error(w, err)

		return false
	}

	if err := ValidateStruct(r.Context(), validate, dest); err != nil {
		response.Error(w, err)

		return false
	}

	return true
}

func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("Field %s is required", err.Field())
	case "email":
		return fmt.Sprintf("Field %s must be a valid email address", err.Field())
	case "min":
		return fmt.Sprintf("Field %s must be at least %s characters", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("Field %s must be at most %s characters", err.Field(), err.Param())
	case "gt":
		return fmt.Sprintf("Field %s must be greater than %s", err.Field(), err.Param())
	case "lt":
		return fmt.Sprintf("Field %s must be less than %s", err.Field(), err.Param())
	default:
		return fmt.Sprintf("Field %s is invalid: %s=%s", err.Field(), err.Tag(), err.Param())
	}
}

func ParseInt(r *http.Request, paramName string) (int64, error) {
	idStr := r.PathValue(paramName)

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, apperrors.BadRequestError(fmt.Sprintf("Invalid %s ID", paramName))
	}

	return id, nil
}

func ParseID(r *http.Request, paramName string) (uuid.UUID, error) {
	idStr := r.PathValue(paramName)

	if idStr == "" {
		return uuid.Nil, apperrors.BadRequestError("Missing path parameter: " + paramName)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, apperrors.BadRequestError(fmt.Sprintf("Invalid %s ID format: must be a UUID", paramName)).WithError(err)
	}

	return id, nil
}
