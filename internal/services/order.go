package service

import (
	"context"
	"time"

	apperrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	ListOrdersByCustomer(ctx context.Context, customerID uuid.UUID, page int, size int) ([]models.Order, int, error)
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) (*models.Order, error)
}

type orderService struct {
	orderRepo   repository.OrderRepository
	cartRepo    repository.CartRepository
	productRepo repository.ProductRepository
}

func NewOrderService(orderRepo repository.OrderRepository, cartRepo repository.CartRepository, productRepo repository.ProductRepository) OrderService {
	return &orderService{orderRepo: orderRepo, cartRepo: cartRepo, productRepo: productRepo}
}

func (s *orderService) validateAndGetProducts(ctx context.Context, cart *models.Cart) (map[uuid.UUID]*models.Product, error) {
	var productIDs []uuid.UUID
	for _, item := range cart.Items {
		productIDs = append(productIDs, item.ProductID)
	}

	products, err := s.productRepo.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, apperrors.DatabaseError("Failed to fetch products").WithError(err)
	}

	productMap := make(map[uuid.UUID]*models.Product)
	for _, p := range products {
		productMap[p.ID] = p
	}

	for _, item := range cart.Items {
		product, exists := productMap[item.ProductID]
		if !exists {
			return nil, apperrors.NotFoundError("Product not found: " + item.ProductID.String())
		}

		if product.StockQuantity < item.Quantity {
			return nil, apperrors.BadRequestError("Insufficient stock for product: " + item.ProductID.String())
		}
	}

	return productMap, nil
}

func (s *orderService) updateInventory(ctx context.Context, cart *models.Cart, productMap map[uuid.UUID]*models.Product) error {
	for _, item := range cart.Items {
		if product, exists := productMap[item.ProductID]; exists {
			product.StockQuantity -= item.Quantity

			if err := s.productRepo.UpdateProduct(ctx, product); err != nil {
				return apperrors.DatabaseError("Failed to update inventory").WithError(err)
			}
		}
	}

	return nil
}

func (s *orderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	cart, err := s.cartRepo.GetCartByCustomerID(ctx, req.CustomerID)
	if err != nil {
		return nil, apperrors.NotFoundError("Cart not found").WithError(err)
	}

	if len(cart.Items) == 0 {
		return nil, apperrors.BadRequestError("Cannot create order with empty cart")
	}

	productMap, err := s.validateAndGetProducts(ctx, cart)
	if err != nil {
		return nil, err
	}

	var grossTotal float64
	for _, item := range req.Items {
		grossTotal += float64(item.Quantity) * item.UnitPrice
	}

	order := &models.Order{
		ID:              uuid.New(),
		CustomerID:      req.CustomerID,
		Status:          models.OrderStatusPending,
		TotalAmount:     grossTotal,
		PaymentStatus:   models.PaymentStatusPending,
		ShippingAddress: &req.ShippingAddress,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	var items []models.OrderItem
	for _, item := range req.Items {
		orderItem := models.OrderItem{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			CreatedAt: time.Now(),
		}
		items = append(items, orderItem)
	}
	order.Items = items

	if err := s.orderRepo.CreateOrder(ctx, order); err != nil {
		return nil, apperrors.DatabaseError("Failed to create order").WithError(err)
	}

	if err := s.updateInventory(ctx, cart, productMap); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *orderService) GetOrderByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	order, err := s.orderRepo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NotFoundError("Order not found").WithError(err)
	}

	return order, nil
}

func (s *orderService) ListOrdersByCustomer(ctx context.Context, customerID uuid.UUID, page, size int) ([]models.Order, int, error) {
	if page < 1 {
		page = 1
	}

	if size < 1 || size > 10 {
		size = 10
	}

	orders, total, err := s.orderRepo.ListOrdersByCustomer(ctx, customerID, page, size)
	if err != nil {
		return nil, 0, apperrors.DatabaseError("Failed to fetch orders").WithError(err)
	}

	return orders, total, nil
}

func (s *orderService) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) (*models.Order, error) {
	// check if order exists or not
	_, err := s.orderRepo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NotFoundError("Order not found").WithError(err)
	}

	order, err := s.orderRepo.UpdateOrderStatus(ctx, id, status)
	if err != nil {
		return nil, apperrors.DatabaseError("Failed to update order status").WithError(err)
	}

	return order, nil
}
