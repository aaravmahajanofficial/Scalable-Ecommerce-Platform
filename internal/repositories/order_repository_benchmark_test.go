package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
)

func BenchmarkCreateOrder_100Items(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			b.Logf("close err: %v", err)
		}
	}()

	repo := repository.NewOrderRepository(db)
	ctx := context.Background()

	var items []models.OrderItem
	for range 100 {
		items = append(items, models.OrderItem{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			Quantity:  1,
			UnitPrice: 10.0,
		})
	}

	order := &models.Order{
		ID:              uuid.New(),
		CustomerID:      uuid.New(),
		Status:          "PENDING",
		TotalAmount:     1000.0,
		PaymentStatus:   "UNPAID",
		PaymentIntentID: "pi_123",
		ShippingAddress: &models.Address{City: "Test"},
		Items:           items,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	b.ResetTimer()
	for range b.N {
		b.StopTimer()
		mock.ExpectExec("INSERT INTO orders").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO order_items").WillReturnResult(sqlmock.NewResult(100, 100))
		b.StartTimer()

		if err := repo.CreateOrder(ctx, order); err != nil {
			b.Fatal(err)
		}
	}
}
