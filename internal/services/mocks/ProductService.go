// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// ProductService is an autogenerated mock type for the ProductService type
type ProductService struct {
	mock.Mock
}

// CreateProduct provides a mock function with given fields: ctx, req
func (_m *ProductService) CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.Product, error) {
	ret := _m.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for CreateProduct")
	}

	var r0 *models.Product
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.CreateProductRequest) (*models.Product, error)); ok {
		return rf(ctx, req)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.CreateProductRequest) *models.Product); ok {
		r0 = rf(ctx, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Product)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.CreateProductRequest) error); ok {
		r1 = rf(ctx, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetProductByID provides a mock function with given fields: ctx, id
func (_m *ProductService) GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetProductByID")
	}

	var r0 *models.Product
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) (*models.Product, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) *models.Product); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Product)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListProducts provides a mock function with given fields: ctx, page, pageSize
func (_m *ProductService) ListProducts(ctx context.Context, page int, pageSize int) ([]*models.Product, int, error) {
	ret := _m.Called(ctx, page, pageSize)

	if len(ret) == 0 {
		panic("no return value specified for ListProducts")
	}

	var r0 []*models.Product
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int) ([]*models.Product, int, error)); ok {
		return rf(ctx, page, pageSize)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int, int) []*models.Product); ok {
		r0 = rf(ctx, page, pageSize)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*models.Product)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int, int) int); ok {
		r1 = rf(ctx, page, pageSize)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(context.Context, int, int) error); ok {
		r2 = rf(ctx, page, pageSize)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// UpdateProduct provides a mock function with given fields: ctx, id, req
func (_m *ProductService) UpdateProduct(ctx context.Context, id uuid.UUID, req *models.UpdateProductRequest) (*models.Product, error) {
	ret := _m.Called(ctx, id, req)

	if len(ret) == 0 {
		panic("no return value specified for UpdateProduct")
	}

	var r0 *models.Product
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID, *models.UpdateProductRequest) (*models.Product, error)); ok {
		return rf(ctx, id, req)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID, *models.UpdateProductRequest) *models.Product); ok {
		r0 = rf(ctx, id, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Product)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID, *models.UpdateProductRequest) error); ok {
		r1 = rf(ctx, id, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewProductService creates a new instance of ProductService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewProductService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProductService {
	mock := &ProductService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
