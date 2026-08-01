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
	// Check if the cart exists or not
	cart, err := s.cartRepo.GetCartByCustomerID(ctx, req.CustomerID)
	if err != nil {
		return nil, errors.NotFoundError("Cart not found").WithError(err)
	}

	if len(cart.Items) == 0 {
		return nil, errors.BadRequestError("Cannot create order with empty cart")
	}

	// now check the availability of the product
	if err := s.checkProductAvailability(ctx, cart.Items); err != nil {
		return nil, err
	}

	// calculate the order total
	grossTotal := s.calculateGrossTotal(req.Items)

	// assemble the order struct
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

	// now add the items
	order.Items = s.buildOrderItems(order.ID, req.Items)

	err = s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		return nil, errors.DatabaseError("Failed to create order").WithError(err)
	}

	if err := s.updateInventory(ctx, cart.Items); err != nil {
		return nil, err
	}

	return order, nil
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

func (s *orderService) checkProductAvailability(ctx context.Context, items map[string]models.CartItem) error {
	for _, item := range items {
		product, err := s.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return errors.NotFoundError("Product not found: " + item.ProductID.String()).WithError(err)
		}

		if product.StockQuantity < item.Quantity {
			return errors.BadRequestError("Insufficient stock for product: " + item.ProductID.String())
		}
	}
	return nil
}

func (s *orderService) calculateGrossTotal(items []models.OrderItem) float64 {
	var grossTotal float64
	for _, item := range items {
		grossTotal += float64(item.Quantity) * item.UnitPrice
	}
	return grossTotal
}

func (s *orderService) buildOrderItems(orderID uuid.UUID, items []models.OrderItem) []models.OrderItem {
	var orderItems []models.OrderItem
	for _, item := range items {
		orderItem := models.OrderItem{
			ID:        uuid.New(),
			OrderID:   orderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			CreatedAt: time.Now(),
		}
		orderItems = append(orderItems, orderItem)
	}
	return orderItems
}

func (s *orderService) updateInventory(ctx context.Context, items map[string]models.CartItem) error {
	for _, item := range items {
		product, err := s.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return errors.DatabaseError("Failed to get product").WithError(err)
		}
		product.StockQuantity -= item.Quantity

		err = s.productRepo.UpdateProduct(ctx, product)
		if err != nil {
			return errors.DatabaseError("Failed to update inventory").WithError(err)
		}
	}
	return nil
}
