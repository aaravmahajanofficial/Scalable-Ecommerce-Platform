package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateOrder(b *testing.B) {
	mockOrderRepo := new(mocks.MockOrderRepository)
	mockCartRepo := new(mocks.MockCartRepository)
	mockProductRepo := new(mocks.MockProductRepository)

	svc := service.NewOrderService(mockOrderRepo, mockCartRepo, mockProductRepo)

	ctx := context.Background()
	customerID := uuid.New()

    // setup 100 items in cart
	cartItems := make(map[string]models.CartItem)
	var products []*models.Product
    var reqItems []models.OrderItem

	for i := 0; i < 100; i++ {
        productID := uuid.New()
		cartItems[productID.String()] = models.CartItem{
			ProductID: productID,
			Quantity:  1,
		}
        products = append(products, &models.Product{
            ID: productID,
            StockQuantity: 10,
        })
        reqItems = append(reqItems, models.OrderItem{
            ProductID: productID,
            Quantity: 1,
            UnitPrice: 10,
        })
	}

	cart := &models.Cart{
		UserID: customerID,
		Items:      cartItems,
	}

	req := &models.CreateOrderRequest{
		CustomerID: customerID,
		Items:      reqItems,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
        b.StopTimer()
        mockCartRepo.ExpectedCalls = nil
        mockProductRepo.ExpectedCalls = nil
        mockOrderRepo.ExpectedCalls = nil

		mockCartRepo.On("GetCartByCustomerID", mock.Anything, customerID).Return(cart, nil)
		mockProductRepo.On("GetProductsByIDs", mock.Anything, mock.Anything).Return(products, nil)
		mockOrderRepo.On("CreateOrder", mock.Anything, mock.AnythingOfType("*models.Order")).Return(nil)

        // Now using a single call to UpdateProducts
        mockProductRepo.On("UpdateProducts", mock.Anything, mock.AnythingOfType("[]*models.Product")).Return(nil).Once()

        // Restore product stock quantity because previous iteration reduces it
        for _, p := range products {
            p.StockQuantity = 10
        }

        b.StartTimer()

		_, err := svc.CreateOrder(ctx, req)
		if err != nil {
			b.Fatalf("expected no error, got %v", err)
		}
	}
}
