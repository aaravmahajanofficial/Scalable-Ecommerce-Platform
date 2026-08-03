package utils

import (
	"net/http"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseID(t *testing.T) {
	validUUIDStr := uuid.New().String()
	validUUID, _ := uuid.Parse(validUUIDStr)

	tests := []struct {
		name         string
		paramName    string
		pathValue    string
		expectedID   uuid.UUID
		expectedErr  bool
		errCodeCheck func(error) bool
	}{
		{
			name:        "Valid UUID",
			paramName:   "id",
			pathValue:   validUUIDStr,
			expectedID:  validUUID,
			expectedErr: false,
		},
		{
			name:        "Missing path parameter",
			paramName:   "id",
			pathValue:   "",
			expectedID:  uuid.Nil,
			expectedErr: true,
			errCodeCheck: func(err error) bool {
				appErr, ok := err.(*errors.AppError)
				return ok && appErr.Code == errors.ErrCodeBadRequest
			},
		},
		{
			name:        "Invalid UUID format",
			paramName:   "id",
			pathValue:   "invalid-uuid",
			expectedID:  uuid.Nil,
			expectedErr: true,
			errCodeCheck: func(err error) bool {
				appErr, ok := err.(*errors.AppError)
				return ok && appErr.Code == errors.ErrCodeBadRequest
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			require.NoError(t, err)

			req.SetPathValue(tt.paramName, tt.pathValue)

			id, err := ParseID(req, tt.paramName)

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.errCodeCheck != nil {
					assert.True(t, tt.errCodeCheck(err), "Error type or code mismatch")
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}
