package service

import (
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateTotal(t *testing.T) {
	s := &cartService{}

	t.Run("EmptyMap", func(t *testing.T) {
		items := make(map[string]models.CartItem)
		total := s.calculateTotal(items)
		assert.Equal(t, 0.0, total)
	})

	t.Run("SingleItem", func(t *testing.T) {
		item1ID := uuid.New().String()
		items := map[string]models.CartItem{
			item1ID: {
				ProductID:  uuid.MustParse(item1ID),
				Quantity:   2,
				UnitPrice:  15.50,
				TotalPrice: 31.00,
			},
		}
		total := s.calculateTotal(items)
		assert.Equal(t, 31.00, total)
	})

	t.Run("MultipleItems", func(t *testing.T) {
		item1ID := uuid.New().String()
		item2ID := uuid.New().String()
		item3ID := uuid.New().String()

		items := map[string]models.CartItem{
			item1ID: {
				ProductID:  uuid.MustParse(item1ID),
				Quantity:   2,
				UnitPrice:  10.00,
				TotalPrice: 20.00,
			},
			item2ID: {
				ProductID:  uuid.MustParse(item2ID),
				Quantity:   1,
				UnitPrice:  15.50,
				TotalPrice: 15.50,
			},
			item3ID: {
				ProductID:  uuid.MustParse(item3ID),
				Quantity:   3,
				UnitPrice:  5.00,
				TotalPrice: 15.00,
			},
		}
		total := s.calculateTotal(items)
		assert.Equal(t, 50.50, total)
	})
}
