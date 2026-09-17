package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func BenchmarkUpdateStock_LoopVsBatch(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := repository.NewProductRepo(db)
	ctx := context.Background()

	itemCounts := []int{5, 10, 50, 100}

	for _, count := range itemCounts {
		products := make([]*models.Product, count)
		ids := make([]uuid.UUID, count)
		quantities := make([]int, count)

		for i := range count {
			id := uuid.New()
			ids[i] = id
			quantities[i] = 100 - i
			products[i] = &models.Product{
				ID:            id,
				StockQuantity: quantities[i],
			}
		}

		b.Run(fmt.Sprintf("Loop_Items_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				for _, p := range products {
					mock.ExpectQuery("UPDATE products SET").
						WithArgs(p.CategoryID, p.Name, p.Description, p.Price, p.StockQuantity, p.Status, p.ID).
						WillReturnRows(sqlmock.NewRows([]string{"updated_at"}))
					if updateErr := repo.UpdateProduct(ctx, p); updateErr != nil {
						b.Fatal(updateErr)
					}
				}
			}
		})

		b.Run(fmt.Sprintf("Batch_Items_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				mock.ExpectExec("UPDATE products AS p").
					WithArgs(pq.Array(ids), pq.Array(quantities)).
					WillReturnResult(sqlmock.NewResult(0, int64(count)))
				if batchErr := repo.UpdateProductStockBatch(ctx, products); batchErr != nil {
					b.Fatal(batchErr)
				}
			}
		})
	}
}
