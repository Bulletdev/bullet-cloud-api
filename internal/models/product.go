package models

import (
	"errors"
	"fmt"
	"sync"
)

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
}

type ProductRepository struct {
	mu       sync.RWMutex
	products map[string]Product
	counter  int
}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{
		products: make(map[string]Product),
		counter:  0,
	}
}

func (r *ProductRepository) Create(product *Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counter++
	product.ID = fmt.Sprintf("PROD-%d", r.counter)
	r.products[product.ID] = *product
	return nil
}

func (r *ProductRepository) GetAll() []Product {
	r.mu.RLock()
	defer r.mu.RUnlock()

	products := make([]Product, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}
	return products
}

func (r *ProductRepository) GetByID(id string) (Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, exists := r.products[id]
	if !exists {
		return Product{}, errors.New("product not found")
	}
	return product, nil
}

func (r *ProductRepository) Update(id string, updatedProduct Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[id]; !exists {
		return errors.New("product not found")
	}

	updatedProduct.ID = id
	r.products[id] = updatedProduct
	return nil
}

func (r *ProductRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[id]; !exists {
		return errors.New("product not found")
	}

	delete(r.products, id)
	return nil
}
