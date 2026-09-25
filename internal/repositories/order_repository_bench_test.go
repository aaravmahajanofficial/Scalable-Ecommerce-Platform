package repository_test

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func BenchmarkListOrdersByCustomer(b *testing.B) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		b.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			b.Logf("failed to close mock db: %v", closeErr)
		}
	}()

	repo := repository.NewOrderRepository(db)
	ctx := context.Background()
	customerID := uuid.New()
	now := time.Now()

	numOrders := 50
	orders := make([]models.Order, numOrders)
	orderIDs := make([]uuid.UUID, numOrders)

	addr := &models.Address{Street: "List St", City: "Listville", State: "LS", PostalCode: "11111", Country: "US"}
	addrJSON, err := json.Marshal(addr)
	if err != nil {
		b.Fatalf("failed to marshal addr: %v", err)
	}

	for i := range numOrders {
		orderIDs[i] = uuid.New()
		orders[i] = models.Order{
			ID:              orderIDs[i],
			CustomerID:      customerID,
			Status:          models.OrderStatusDelivered,
			TotalAmount:     50.0,
			PaymentStatus:   models.PaymentStatusSucceeded,
			ShippingAddress: addr,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}

	expectedCountSQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM orders WHERE customer_id = $1`)
	expectedListOrdersSQL := regexp.QuoteMeta(`
        SELECT id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at
        FROM orders
        WHERE customer_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `)
	expectedListItemsSQL := regexp.QuoteMeta(`
        SELECT id, order_id, product_id, quantity, unit_price, created_at
        FROM order_items
        WHERE order_id = ANY($1)
    `)

	b.ResetTimer()
	for range b.N {
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(numOrders))

		orderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"})
		for j := range numOrders {
			orderRows.AddRow(orders[j].ID, orders[j].Status, orders[j].TotalAmount, orders[j].PaymentStatus, orders[j].PaymentIntentID, addrJSON, orders[j].CreatedAt, orders[j].UpdatedAt)
		}
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, 50, 0).WillReturnRows(orderRows)

		itemRows := sqlmock.NewRows([]string{"id", "order_id", "product_id", "quantity", "unit_price", "created_at"})
		for j := range numOrders {
			itemRows.AddRow(uuid.New(), orders[j].ID, uuid.New(), 1, 50.0, now)
		}
		mock.ExpectQuery(expectedListItemsSQL).WithArgs(pq.Array(orderIDs)).WillReturnRows(itemRows)

		if _, _, err := repo.ListOrdersByCustomer(ctx, customerID, 1, 50); err != nil {
			b.Fatalf("ListOrdersByCustomer failed: %v", err)
		}
	}
}
