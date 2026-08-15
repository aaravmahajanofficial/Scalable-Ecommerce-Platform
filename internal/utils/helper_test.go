package utils_test

import (
	"net/http"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestParseInt(t *testing.T) {
	tests := []struct {
		name          string
		paramValue    string
		expectedID    int64
		expectedError bool
	}{
		{
			name:          "Valid ID",
			paramValue:    "123",
			expectedID:    123,
			expectedError: false,
		},
		{
			name:          "Invalid ID - Not a number",
			paramValue:    "abc",
			expectedID:    0,
			expectedError: true,
		},
		{
			name:          "Invalid ID - Empty string",
			paramValue:    "",
			expectedID:    0,
			expectedError: true,
		},
		{
			name:          "Invalid ID - Float value",
			paramValue:    "12.3",
			expectedID:    0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			assert.NoError(t, err)

			req.SetPathValue("id", tt.paramValue)

			id, err := utils.ParseInt(req, "id")

			if tt.expectedError {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "Invalid id ID")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}
