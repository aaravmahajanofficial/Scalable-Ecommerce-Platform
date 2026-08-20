package service_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repoMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateCart(t *testing.T) {
	mockRepo := repoMocks.NewMockCartRepository(t)
	cartService := service.NewCartService(mockRepo)
	ctx := t.Context()
	userID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("CreateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
			return cart.UserID == userID &&
				cart.Total == 0 &&
				len(cart.Items) == 0 &&
				!cart.CreatedAt.IsZero() &&
				!cart.UpdatedAt.IsZero()
		})).Return(nil).Once()

		cart, err := cartService.CreateCart(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, cart)
		assert.Equal(t, userID, cart.UserID)
		assert.Equal(t, 0.0, cart.Total)
		assert.Empty(t, cart.Items)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Database Error", func(t *testing.T) {
		dbError := errors.New("database connection failed")
		mockRepo.On("CreateCart", ctx, mock.AnythingOfType("*models.Cart")).Return(dbError).Once()

		cart, err := cartService.CreateCart(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, cart)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.Equal(t, "Failed to create cart", appErr.Message)
		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetCart(t *testing.T) {
	mockRepo := repoMocks.NewMockCartRepository(t)
	cartService := service.NewCartService(mockRepo)
	ctx := t.Context()
	customerID := uuid.New()
	existingCart := &models.Cart{
		ID:        uuid.New(),
		UserID:    customerID,
		Items:     make(map[string]models.CartItem),
		Total:     0,
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	t.Run("Success - Cart Found", func(t *testing.T) {
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(existingCart, nil).Once()

		cart, err := cartService.GetCart(ctx, customerID)

		assert.NoError(t, err)
		assert.NotNil(t, cart)
		assert.Equal(t, existingCart.ID, cart.ID)
		assert.Equal(t, customerID, cart.UserID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Cart Not Found", func(t *testing.T) {
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, sql.ErrNoRows).Once()

		cart, err := cartService.GetCart(ctx, customerID)

		assert.Error(t, err)
		assert.Nil(t, cart)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Other Database Error", func(t *testing.T) {
		dbError := errors.New("unexpected database error")
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, dbError).Once()

		cart, err := cartService.GetCart(ctx, customerID)

		assert.Error(t, err)
		assert.Nil(t, cart)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeInternal, appErr.Code)
		assert.Equal(t, "Failed to retrieve cart", appErr.Message)
		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddItem_SuccessNewItem(t *testing.T) {
	mockRepo := repoMocks.NewMockCartRepository(t)
	cartService := service.NewCartService(mockRepo)
	ctx := t.Context()
	customerID := uuid.New()
	productID1 := uuid.New()

	existingCart := &models.Cart{
		ID:     uuid.New(),
		UserID: customerID,
		Items:  make(map[string]models.CartItem),
		Total:  0,
	}

	addItemReq := &models.AddItemRequest{
		ProductID: productID1,
		Quantity:  2,
		UnitPrice: 10.50,
	}

	mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(existingCart, nil).Once()
	mockRepo.On("UpdateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
		item, exists := cart.Items[productID1.String()]
		return exists &&
			item.ProductID == productID1 &&
			item.Quantity == 2 &&
			item.UnitPrice == 10.50 &&
			item.TotalPrice == 21.00 &&
			cart.Total == 21.00 &&
			cart.UserID == customerID
	})).Return(nil).Once()

	updatedCart, err := cartService.AddItem(ctx, customerID, addItemReq)

	assert.NoError(t, err)
	assert.NotNil(t, updatedCart)
	assert.Len(t, updatedCart.Items, 1)
	item, exists := updatedCart.Items[productID1.String()]
	assert.True(t, exists)
	assert.Equal(t, productID1, item.ProductID)
	assert.Equal(t, 2, item.Quantity)
	assert.Equal(t, 10.50, item.UnitPrice)
	assert.Equal(t, 21.00, item.TotalPrice)
	assert.Equal(t, 21.00, updatedCart.Total)
	assert.WithinDuration(t, time.Now(), updatedCart.UpdatedAt, time.Second)
	mockRepo.AssertExpectations(t)
}

func TestAddItem_SuccessUpdateCart(t *testing.T) {
	mockRepo := repoMocks.NewMockCartRepository(t)
	cartService := service.NewCartService(mockRepo)
	ctx := t.Context()
	customerID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()

	existingCart := &models.Cart{
		ID:     uuid.New(),
		UserID: customerID,
		Items: map[string]models.CartItem{
			productID1.String(): {ProductID: productID1, Quantity: 1, UnitPrice: 5.0, TotalPrice: 5.0},
		},
		Total: 5.0,
	}

	addItemReq2 := &models.AddItemRequest{ProductID: productID2, Quantity: 3, UnitPrice: 2.0}

	mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(existingCart, nil).Once()
	mockRepo.On("UpdateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
		item1, exists1 := cart.Items[productID1.String()]
		item2, exists2 := cart.Items[productID2.String()]
		expectedTotal := 5.0 + (3 * 2.0)

		return exists1 && exists2 &&
			item1.Quantity == 1 && item1.TotalPrice == 5.0 &&
			item2.ProductID == productID2 && item2.Quantity == 3 && item2.UnitPrice == 2.0 && item2.TotalPrice == 6.0 &&
			cart.Total == expectedTotal &&
			len(cart.Items) == 2
	})).Return(nil).Once()

	updatedCart, err := cartService.AddItem(ctx, customerID, addItemReq2)

	assert.NoError(t, err)
	assert.NotNil(t, updatedCart)
	assert.Len(t, updatedCart.Items, 2)
	assert.Equal(t, 11.00, updatedCart.Total)
	mockRepo.AssertExpectations(t)
}

func TestAddItem_Failures(t *testing.T) {
	mockRepo := repoMocks.NewMockCartRepository(t)
	cartService := service.NewCartService(mockRepo)
	ctx := t.Context()
	customerID := uuid.New()
	productID1 := uuid.New()

	addItemReq := &models.AddItemRequest{
		ProductID: productID1,
		Quantity:  2,
		UnitPrice: 10.50,
	}

	t.Run("Failure - Cart Not Found", func(t *testing.T) {
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, sql.ErrNoRows).Once()

		cart, err := cartService.AddItem(ctx, customerID, addItemReq)

		assert.Error(t, err)
		assert.Nil(t, cart)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "UpdateCart")
	})

	t.Run("Failure - Database Error on Update", func(t *testing.T) {
		dbError := errors.New("failed to write to db")
		existingCart := &models.Cart{
			ID:     uuid.New(),
			UserID: customerID,
			Items:  make(map[string]models.CartItem),
			Total:  0,
		}

		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(existingCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.AnythingOfType("*models.Cart")).Return(dbError).Once()

		cart, err := cartService.AddItem(ctx, customerID, addItemReq)

		assert.Error(t, err)
		assert.Nil(t, cart)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.Equal(t, "Failed to update cart", appErr.Message)
		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}

func TestCartService_UpdateQuantity(t *testing.T) {
	mockRepo := repoMocks.NewMockCartRepository(t)
	cartService := service.NewCartService(mockRepo)
	ctx := t.Context()
	customerID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()

	initialItem := models.CartItem{
		ProductID:  productID1,
		Quantity:   2,
		UnitPrice:  10.0,
		TotalPrice: 20.0,
	}
	initialCart := &models.Cart{
		ID:     uuid.New(),
		UserID: customerID,
		Items:  map[string]models.CartItem{productID1.String(): initialItem},
		Total:  20.0,
	}

	resetState := func() {
		mockRepo.ExpectedCalls = nil
		mockRepo.Calls = nil
		initialCart.Items = map[string]models.CartItem{
			productID1.String(): {
				ProductID:  productID1,
				Quantity:   2,
				UnitPrice:  10.0,
				TotalPrice: 20.0,
			},
		}
		initialCart.Total = 20.0
	}

	t.Run("Success - Update Existing Item Quantity", func(t *testing.T) {
		resetState()
		updateReq := &models.UpdateQuantityRequest{ProductID: productID1, Quantity: 5}

		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(initialCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
			item, exists := cart.Items[productID1.String()]
			return exists &&
				item.Quantity == 5 &&
				item.TotalPrice == 50.0 &&
				cart.Total == 50.0
		})).Return(nil).Once()

		updatedCart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		assert.NoError(t, err)
		assert.NotNil(t, updatedCart)
		item, exists := updatedCart.Items[productID1.String()]
		assert.True(t, exists)
		assert.Equal(t, 5, item.Quantity)
		assert.Equal(t, 50.0, item.TotalPrice)
		assert.Equal(t, 50.0, updatedCart.Total)
		assert.WithinDuration(t, time.Now(), updatedCart.UpdatedAt, time.Second)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Remove Item (Quantity 0)", func(t *testing.T) {
		resetState()
		updateReq := &models.UpdateQuantityRequest{ProductID: productID1, Quantity: 0}

		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(initialCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
			_, exists := cart.Items[productID1.String()]
			return !exists &&
				cart.Total == 0.0 &&
				len(cart.Items) == 0
		})).Return(nil).Once()

		updatedCart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		assert.NoError(t, err)
		assert.NotNil(t, updatedCart)
		assert.Empty(t, updatedCart.Items)
		assert.Equal(t, 0.0, updatedCart.Total)
		assert.WithinDuration(t, time.Now(), updatedCart.UpdatedAt, time.Second)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Cart Not Found on Get", func(t *testing.T) {
		resetState()
		updateReq := &models.UpdateQuantityRequest{ProductID: productID1, Quantity: 3}

		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, sql.ErrNoRows).Once()

		cart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		assert.Error(t, err)
		assert.Nil(t, cart)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		assert.Equal(t, "Cart not found", appErr.Message)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "UpdateCart", mock.Anything, mock.Anything)
	})

	t.Run("Failure - Item Not Found in Cart", func(t *testing.T) {
		resetState()
		updateReq := &models.UpdateQuantityRequest{ProductID: productID2, Quantity: 1}

		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(initialCart, nil).Once()

		cart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		assert.Error(t, err)
		assert.Nil(t, cart)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeBadRequest, appErr.Code)
		assert.Equal(t, "Item not found in the cart", appErr.Message)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "UpdateCart", mock.Anything, mock.Anything)
	})

	t.Run("Failure - Database Error on Update", func(t *testing.T) {
		resetState()
		updateReq := &models.UpdateQuantityRequest{ProductID: productID1, Quantity: 4}
		dbError := errors.New("db write constraint failed")

		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(initialCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.AnythingOfType("*models.Cart")).Return(dbError).Once()

		cart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		assert.Error(t, err)
		assert.Nil(t, cart)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.Equal(t, "Failed to update cart", appErr.Message)
		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}
