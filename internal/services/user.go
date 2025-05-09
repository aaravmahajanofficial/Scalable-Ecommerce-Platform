package service

import (
	"context"
	"time"

	appError "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

const userTracerName = "ecommerce/userservice"

type UserService interface {
	Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error)
	Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type userService struct {
	repo      repository.UserRepository
	redisRepo repository.RateLimitRepository
	jwtKey    []byte
}

func NewUserService(repo repository.UserRepository, redisRepo repository.RateLimitRepository, jwtKey []byte) UserService {
	return &userService{
		repo:      repo,
		redisRepo: redisRepo,
		jwtKey:    jwtKey,
	}
}

func (s *userService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	existingUser, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, appError.DatabaseError("error checking existing user").WithError(err)
	}

	if existingUser != nil {
		return nil, appError.DuplicateEntryError("Email already registered")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, appError.InternalError("Failed to secure password").WithError(err)
	}

	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, appError.DatabaseError("Failed to create user").WithError(err)
	}

	return user, err
}

func (s *userService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	tracer := otel.Tracer(userTracerName)

	ctx, span := tracer.Start(ctx, "LoginUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", req.Email))

	// check rate limit
	allowed, remaining, retryAfter, err := s.redisRepo.CheckLoginRateLimit(ctx, req.Email)
	if err != nil {
		return nil, appError.ThirdPartyError("Rate limit check failed").WithError(err)
	}

	if !allowed {
		return &models.LoginResponse{
			Success:    false,
			Message:    "Too many login attempts. Please try again later.",
			RetryAfter: retryAfter,
		}, nil
	}

	// Retrieve the user from the DB and compare the passwords
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return &models.LoginResponse{
			Success:        false,
			Message:        "Invalid email or password",
			RemainingTries: remaining,
		}, nil
	}

	claims := &models.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Generate Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		return nil, appError.InternalError("Failed to generate authentication token").WithError(err)
	}

	return &models.LoginResponse{
		Success:   true,
		Token:     tokenString,
		ExpiresIn: int(time.Until(claims.ExpiresAt.Time).Seconds()),
	}, nil
}

func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, appError.NotFoundError("User not found").WithError(err)
	}

	// Note: Password is already included in repository query
	return user, nil
}
