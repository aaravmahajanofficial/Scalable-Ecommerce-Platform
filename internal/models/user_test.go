package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_JSON(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)

	user := models.User{
		ID:        userID,
		Name:      "John Doe",
		Username:  "johndoe",
		Email:     "john@example.com",
		Password:  "hashedsecretpassword",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test JSON Serialization (Password must be omitted due to json:"-")
	data, err := json.Marshal(user)
	require.NoError(t, err)

	dataStr := string(data)
	assert.Contains(t, dataStr, userID.String())
	assert.Contains(t, dataStr, "John Doe")
	assert.Contains(t, dataStr, "johndoe")
	assert.Contains(t, dataStr, "john@example.com")
	assert.NotContains(t, dataStr, "hashedsecretpassword")

	// Test JSON Deserialization
	var unmarshaledUser models.User
	err = json.Unmarshal(data, &unmarshaledUser)
	require.NoError(t, err)

	assert.Equal(t, user.ID, unmarshaledUser.ID)
	assert.Equal(t, user.Name, unmarshaledUser.Name)
	assert.Equal(t, user.Username, unmarshaledUser.Username)
	assert.Equal(t, user.Email, unmarshaledUser.Email)
	assert.Empty(t, unmarshaledUser.Password)
}

func TestRegisterRequest_Validation(t *testing.T) {
	t.Parallel()

	validate := validator.New()

	tests := []struct {
		name    string
		req     models.RegisterRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			req: models.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			wantErr: false,
		},
		{
			name: "Missing email",
			req: models.RegisterRequest{
				Password: "password123",
				Name:     "Test User",
			},
			wantErr: true,
		},
		{
			name: "Invalid email format",
			req: models.RegisterRequest{
				Email:    "invalid-email",
				Password: "password123",
				Name:     "Test User",
			},
			wantErr: true,
		},
		{
			name: "Password too short",
			req: models.RegisterRequest{
				Email:    "test@example.com",
				Password: "pass",
				Name:     "Test User",
			},
			wantErr: true,
		},
		{
			name: "Missing name",
			req: models.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validate.Struct(tc.req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	t.Parallel()

	validate := validator.New()

	tests := []struct {
		name    string
		req     models.LoginRequest
		wantErr bool
	}{
		{
			name: "Valid login request",
			req: models.LoginRequest{
				Email:    "login@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "Invalid email format",
			req: models.LoginRequest{
				Email:    "invalid-email",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "Missing email",
			req: models.LoginRequest{
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "Missing password",
			req: models.LoginRequest{
				Email: "login@example.com",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validate.Struct(tc.req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoginResponse_JSON(t *testing.T) {
	t.Parallel()

	resp := models.LoginResponse{
		Success:        true,
		Token:          "jwt-token-string",
		ExpiresIn:      3600,
		RemainingTries: 5,
		RetryAfter:     0,
		Message:        "Success",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var unmarshaled models.LoginResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, resp, unmarshaled)
}

func TestClaims_JWT(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	secretKey := []byte("secret-key")

	claims := models.Claims{
		UserID: userID,
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "test-issuer",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	parsedToken, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	parsedClaims, ok := parsedToken.Claims.(*models.Claims)
	require.True(t, ok)
	assert.Equal(t, claims.UserID, parsedClaims.UserID)
	assert.Equal(t, claims.Email, parsedClaims.Email)
	assert.Equal(t, claims.Issuer, parsedClaims.Issuer)
}
