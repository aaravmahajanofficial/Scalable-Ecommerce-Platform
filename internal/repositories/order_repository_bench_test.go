package repository

import (
	"github.com/lib/pq"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/DATA-DOG/go-sqlmock"
	"context"
)

func BenchmarkListOrdersByCustomer(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewOrderRepository(db)
	ctx := context.Background()

	customerID := uuid.New()
	page, size := 1, 100
	offset := (page - 1) * size
	now := time.Now()

	// Generate 100 mock orders
	var orders []models.Order
	orderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"})
	for i := 0; i < size; i++ {
		id := uuid.New()
		orders = append(orders, models.Order{ID: id})
		orderRows.AddRow(id, models.OrderStatusDelivered, 50.0, models.PaymentStatusSucceeded, "pi_123", `{"street": "123", "city": "City", "state": "ST", "postal_code": "12345", "country": "US"}`, now, now)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mock count query
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM orders WHERE customer_id = \$1`).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(size))

		// Mock list orders query
		// Note: we can't easily reuse orderRows, we need to create a new one each time for the mock
		currentOrderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"})
		for _, o := range orders {
			currentOrderRows.AddRow(o.ID, models.OrderStatusDelivered, 50.0, models.PaymentStatusSucceeded, "pi_123", `{"street": "123", "city": "City", "state": "ST", "postal_code": "12345", "country": "US"}`, now, now)
		}
		mock.ExpectQuery(`SELECT id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at FROM orders WHERE customer_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).WithArgs(customerID, size, offset).WillReturnRows(currentOrderRows)

		// Mock items query for all orders
		orderIDs := make([]uuid.UUID, len(orders))
		itemRows := sqlmock.NewRows([]string{"id", "order_id", "product_id", "quantity", "unit_price", "created_at"})
		for i, o := range orders {
			orderIDs[i] = o.ID
			itemRows.AddRow(uuid.New(), o.ID, uuid.New(), 1, 50.0, now)
		}
		mock.ExpectQuery(`SELECT id, order_id, product_id, quantity, unit_price, created_at FROM order_items WHERE order_id = ANY\(\$1\)`).WithArgs(pq.Array(orderIDs)).WillReturnRows(itemRows)

		_, _, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)
		if err != nil {
			b.Fatalf("ListOrdersByCustomer failed: %v", err)
		}
	}
}
