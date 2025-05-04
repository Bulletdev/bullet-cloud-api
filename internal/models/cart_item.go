package models

import (
	"time"

	"github.com/google/uuid"
)

// CartItem represents an item within a shopping cart.
type CartItem struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CartID    uuid.UUID `json:"cart_id" db:"cart_id"`       // Foreign key to carts table
	ProductID uuid.UUID `json:"product_id" db:"product_id"` // Foreign key to products table
	Quantity  int       `json:"quantity" db:"quantity"`     // Quantity of the product
	Price     float64   `json:"price" db:"price"`           // Price of the product at the time it was added
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Optional: Include product details directly in the response (requires JOIN in repository)
	// ProductName string `json:"product_name,omitempty" db:"product_name"`
	// ProductDescription string `json:"product_description,omitempty" db:"product_description"`
}
