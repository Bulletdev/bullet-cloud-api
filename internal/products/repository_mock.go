package products

import (
	"bullet-cloud-api/internal/models"
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockProductRepository is a mock type for the ProductRepository interface
type MockProductRepository struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, product
func (_m *MockProductRepository) Create(ctx context.Context, product *models.Product) (*models.Product, error) {
	ret := _m.Called(ctx, product)

	var r0 *models.Product
	if rf, ok := ret.Get(0).(func(context.Context, *models.Product) *models.Product); ok {
		r0 = rf(ctx, product)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Product)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *models.Product) error); ok {
		r1 = rf(ctx, product)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindByID provides a mock function with given fields: ctx, id
func (_m *MockProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.Product
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) *models.Product); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Product)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindAll provides a mock function with given fields: ctx
func (_m *MockProductRepository) FindAll(ctx context.Context) ([]models.Product, error) {
	ret := _m.Called(ctx)

	var r0 []models.Product
	if rf, ok := ret.Get(0).(func(context.Context) []models.Product); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Product)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, id, product
func (_m *MockProductRepository) Update(ctx context.Context, id uuid.UUID, product *models.Product) (*models.Product, error) {
	ret := _m.Called(ctx, id, product)

	var r0 *models.Product
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID, *models.Product) *models.Product); ok {
		r0 = rf(ctx, id, product)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Product)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID, *models.Product) error); ok {
		r1 = rf(ctx, id, product)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ctx, id
func (_m *MockProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
