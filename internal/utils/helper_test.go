package utils

import (
	"context"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestValidateStruct(t *testing.T) {
	validate := validator.New()
	ctx := context.Background()

	type TestStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
		Age   int    `validate:"gte=18"`
		Min   string `validate:"min=3"`
		Max   string `validate:"max=5"`
		Gt    int    `validate:"gt=10"`
		Lt    int    `validate:"lt=20"`
	}

	t.Run("Valid Struct", func(t *testing.T) {
		validData := TestStruct{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   25,
			Min:   "abc",
			Max:   "abcd",
			Gt:    15,
			Lt:    15,
		}

		err := ValidateStruct(ctx, validate, validData)
		assert.NoError(t, err)
	})

	t.Run("Validation Failure", func(t *testing.T) {
		invalidData := TestStruct{
			Name:  "",              // missing required
			Email: "invalid-email", // invalid email
			Age:   10,              // less than 18 (gte)
			Min:   "a",             // less than 3
			Max:   "abcdef",        // more than 5
			Gt:    5,               // less than 10
			Lt:    25,              // more than 20
		}

		err := ValidateStruct(ctx, validate, invalidData)
		assert.Error(t, err)

		appErr, ok := errors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, errors.ErrCodeValidation, appErr.Code)

		assert.Contains(t, appErr.Detail, "Field Name is required")
		assert.Contains(t, appErr.Detail, "Field Email must be a valid email address")
		assert.Contains(t, appErr.Detail, "Field Age is invalid: gte=18")
		assert.Contains(t, appErr.Detail, "Field Min must be at least 3 characters")
		assert.Contains(t, appErr.Detail, "Field Max must be at most 5 characters")
		assert.Contains(t, appErr.Detail, "Field Gt must be greater than 10")
		assert.Contains(t, appErr.Detail, "Field Lt must be less than 20")
	})

	t.Run("Unexpected Validation Error", func(t *testing.T) {
		// Passing nil to struct validation causes an InvalidValidationError
		err := ValidateStruct(ctx, validate, nil)
		assert.Error(t, err)

		appErr, ok := errors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, errors.ErrCodeInternal, appErr.Code)
		assert.Equal(t, "Unexpected validation error", appErr.Message)
	})
}
