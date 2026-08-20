// Package middleware provides HTTP middleware.
package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey uuid.UUID

var UserContextKey = contextKey(uuid.New())

type AuthMiddleware struct {
	jwtKey []byte
}

func NewAuthMiddleware(jwtKey []byte) *AuthMiddleware {
	return &AuthMiddleware{jwtKey: jwtKey}
}

func (m *AuthMiddleware) extractBearerToken(authHeader string, logger *slog.Logger) (string, *appErrors.AppError) {
	if authHeader == "" {
		logger.Warn("Missing authorization header")
		return "", appErrors.UnauthorizedError("Authorization header is required")
	}

	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		logger.Warn("Invalid authorization header format")
		return "", appErrors.UnauthorizedError("Invalid authorization format")
	}

	return tokenParts[1], nil
}

func (m *AuthMiddleware) validateJWTClaims(tokenString string, logger *slog.Logger) (*models.Claims, *appErrors.AppError) {
	claims := &models.Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Header["alg"] != jwt.SigningMethodHS256.Alg() {
			logger.Error("Unexpected signing method used in JWT", slog.Any("alg", t.Header["alg"]))
			return nil, appErrors.BadRequestError("unexpected signing method")
		}
		return m.jwtKey, nil
	})
	if err != nil {
		logger.Warn("JWT parsing failed", slog.String("error", err.Error()))

		var appErr *appErrors.AppError
		if errors.As(err, &appErr) && appErr.Code == appErrors.ErrCodeBadRequest {
			return nil, appErr
		}
		return nil, appErrors.UnauthorizedError("Invalid or expired token")
	}

	if !token.Valid {
		logger.Warn("Invalid token")
		return nil, appErrors.UnauthorizedError("Invalid token")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		logger.Warn("Expired token", slog.String("userId", claims.UserID.String()))
		return nil, appErrors.UnauthorizedError("Token expired")
	}

	return claims, nil
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := LoggerFromContext(r.Context())

		tokenString, extractErr := m.extractBearerToken(r.Header.Get("Authorization"), logger)
		if extractErr != nil {
			response.Error(w, extractErr)
			return
		}

		claims, validateErr := m.validateJWTClaims(tokenString, logger)
		if validateErr != nil {
			response.Error(w, validateErr)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		requestScopedLogger := logger.With(slog.String("userId", claims.UserID.String()))
		ctx = context.WithValue(ctx, LoggerKey, requestScopedLogger)

		requestScopedLogger.Info("User authenticated")
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
