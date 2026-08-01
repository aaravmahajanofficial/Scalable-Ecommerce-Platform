package service_test

import (
	"context"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateOrder(b *testing.B) {
	mockOrderRepo := new(mocks.MockOrderRepository)
	mockCartRepo := new(mocks.MockCartRepository)
	mockProductRepo := new(mocks.MockProductRepository)

	orderService := service.NewOrderService(mockOrderRepo, mockCartRepo, mockProductRepo)

	ctx := context.Background()
	customerID := uuid.New()

	// Create a large cart to simulate N+1 problem
	numItems := 100
	cartItems := make(map[string]models.CartItem)
	orderItems := make([]models.OrderItem, numItems)

	for i := 0; i < numItems; i++ {
		productID := uuid.New()
		cartItems[productID.String()] = models.CartItem{
			ProductID: productID,
			Quantity:  1,
		}

		orderItems[i] = models.OrderItem{
			ProductID: productID,
			Quantity:  1,
			UnitPrice: 10.0,
		}

		// Setup mock for both checks and updates
		mockProductRepo.On("UpdateProduct", ctx, mock.AnythingOfType("*models.Product")).Return(nil).Times(1 * b.N)
	}

	mockProductRepo.On("GetProductsByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).Return(func(ctx context.Context, ids []uuid.UUID) []*models.Product {
		var products []*models.Product
		for _, id := range ids {
			products = append(products, &models.Product{
				ID:            id,
				StockQuantity: 100,
				Price:         10.0,
			})
		}
		return products
	}, nil).Times(1 * b.N)

	mockCart := &models.Cart{
		UserID: customerID,
		Items:  cartItems,
	}

	mockCartRepo.On("GetCartByCustomerID", ctx, customerID).Return(mockCart, nil)
	mockOrderRepo.On("CreateOrder", ctx, mock.AnythingOfType("*models.Order")).Return(nil)

	req := &models.CreateOrderRequest{
		CustomerID: customerID,
		Items:      orderItems,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := orderService.CreateOrder(ctx, req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
