package service

import (
	"context"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
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

func (s *orderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	cart, productMap, err := s.validateCartAndGetProducts(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	err = s.checkProductAvailability(cart, productMap)
	if err != nil {
		return nil, err
	}

	order := s.assembleOrder(req)

	err = s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		return nil, errors.DatabaseError("Failed to create order").WithError(err)
	}

	err = s.updateInventory(ctx, cart, productMap)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s *orderService) validateCartAndGetProducts(ctx context.Context, customerID uuid.UUID) (*models.Cart, map[uuid.UUID]*models.Product, error) {
	cart, err := s.cartRepo.GetCartByCustomerID(ctx, customerID)
	if err != nil {
		return nil, nil, errors.NotFoundError("Cart not found").WithError(err)
	}

	if len(cart.Items) == 0 {
		return nil, nil, errors.BadRequestError("Cannot create order with empty cart")
	}

	var productIDs []uuid.UUID
	for _, item := range cart.Items {
		productIDs = append(productIDs, item.ProductID)
	}

	products, err := s.productRepo.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, nil, errors.DatabaseError("Failed to fetch products").WithError(err)
	}

	productMap := make(map[uuid.UUID]*models.Product)
	for _, p := range products {
		productMap[p.ID] = p
	}

	return cart, productMap, nil
}

func (s *orderService) checkProductAvailability(cart *models.Cart, productMap map[uuid.UUID]*models.Product) error {
	for _, item := range cart.Items {
		product, exists := productMap[item.ProductID]
		if !exists {
			return errors.NotFoundError("Product not found: " + item.ProductID.String())
		}

		if product.StockQuantity < item.Quantity {
			return errors.BadRequestError("Insufficient stock for product: " + item.ProductID.String())
		}
	}
	return nil
}

func (s *orderService) assembleOrder(req *models.CreateOrderRequest) *models.Order {
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

	return order
}

func (s *orderService) updateInventory(ctx context.Context, cart *models.Cart, productMap map[uuid.UUID]*models.Product) error {
	for _, item := range cart.Items {
		if product, exists := productMap[item.ProductID]; exists {
			product.StockQuantity -= item.Quantity

			err := s.productRepo.UpdateProduct(ctx, product)
			if err != nil {
				return errors.DatabaseError("Failed to update inventory").WithError(err)
			}
		}
	}
	return nil
}

func (s *orderService) GetOrderByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	order, err := s.orderRepo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Order not found").WithError(err)
	}

	return order, nil
}

func (s *orderService) ListOrdersByCustomer(ctx context.Context, customerID uuid.UUID, page int, size int) ([]models.Order, int, error) {
	if page < 1 {
		page = 1
	}

	if size < 1 || size > 10 {
		size = 10
	}

	orders, total, err := s.orderRepo.ListOrdersByCustomer(ctx, customerID, page, size)
	if err != nil {
		return nil, 0, errors.DatabaseError("Failed to fetch orders").WithError(err)
	}

	return orders, total, nil
}

func (s *orderService) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) (*models.Order, error) {
	// check if order exists or not
	_, err := s.orderRepo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Order not found").WithError(err)
	}

	order, err := s.orderRepo.UpdateOrderStatus(ctx, id, status)
	if err != nil {
		return nil, errors.DatabaseError("Failed to update order status").WithError(err)
	}

	return order, nil
}
