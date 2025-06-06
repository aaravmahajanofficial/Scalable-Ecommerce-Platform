package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	appError "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
)

type CartService interface {
	CreateCart(ctx context.Context, userID uuid.UUID) (*models.Cart, error)
	GetCart(ctx context.Context, customerID uuid.UUID) (*models.Cart, error)
	AddItem(ctx context.Context, customerID uuid.UUID, req *models.AddItemRequest) (*models.Cart, error)
	UpdateQuantity(ctx context.Context, customerID uuid.UUID, req *models.UpdateQuantityRequest) (*models.Cart, error)
}

type cartService struct {
	repo repository.CartRepository
}

func NewCartService(repo repository.CartRepository) CartService {
	return &cartService{repo: repo}
}

func (s *cartService) CreateCart(ctx context.Context, userID uuid.UUID) (*models.Cart, error) {
	cart := &models.Cart{
		ID:        uuid.New(),
		UserID:    userID,
		Items:     make(map[string]models.CartItem),
		Total:     0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.repo.CreateCart(ctx, cart)
	if err != nil {
		return nil, appError.DatabaseError("Failed to create cart").WithError(err)
	}

	return cart, nil
}

func (s *cartService) GetCart(ctx context.Context, customerID uuid.UUID) (*models.Cart, error) {
	cart, err := s.repo.GetCartByCustomerID(ctx, customerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NotFoundError("Cart not found")
		}

		return nil, appError.InternalError("Failed to retrieve cart").WithError(err)
	}

	return cart, err
}

func (s *cartService) AddItem(ctx context.Context, customerID uuid.UUID, req *models.AddItemRequest) (*models.Cart, error) {
	cart, err := s.repo.GetCartByCustomerID(ctx, customerID)
	if err != nil {
		return nil, appError.NotFoundError("Cart not found").WithError(err)
	}

	item := models.CartItem{
		ProductID:  req.ProductID,
		Quantity:   req.Quantity,
		UnitPrice:  req.UnitPrice,
		TotalPrice: float64(req.Quantity) * req.UnitPrice,
	}

	cart.Items[req.ProductID.String()] = item
	cart.UpdatedAt = time.Now()
	cart.Total = s.calculateTotal(cart.Items)

	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		return nil, appError.DatabaseError("Failed to update cart").WithError(err)
	}

	return cart, nil
}

func (s *cartService) UpdateQuantity(ctx context.Context, customerID uuid.UUID, req *models.UpdateQuantityRequest) (*models.Cart, error) {
	cart, err := s.repo.GetCartByCustomerID(ctx, customerID)
	if err != nil {
		return nil, appError.NotFoundError("Cart not found").WithError(err)
	}

	item, exists := cart.Items[req.ProductID.String()]
	if !exists {
		return nil, appError.BadRequestError("Item not found in the cart")
	}

	if req.Quantity == 0 {
		delete(cart.Items, req.ProductID.String())
	} else {
		item.Quantity = req.Quantity
		item.TotalPrice = item.UnitPrice * float64(item.Quantity)
		cart.Items[req.ProductID.String()] = item
	}

	// update the cart
	cart.UpdatedAt = time.Now()
	cart.Total = s.calculateTotal(cart.Items)

	err = s.repo.UpdateCart(ctx, cart)
	if err != nil {
		return nil, appError.DatabaseError("Failed to update cart").WithError(err)
	}

	return cart, nil
}

func (s *cartService) calculateTotal(items map[string]models.CartItem) float64 {
	var totalPrice float64

	for _, item := range items {
		totalPrice += item.TotalPrice
	}

	return totalPrice
}
