package models

import (
	"time"

	"github.com/google/uuid"
)

// OrderItem represents an item within an order.
type OrderItem struct {
	ID        uuid.UUID `json:"id" db:"id"`
	OrderID   uuid.UUID `json:"order_id" db:"order_id"`     // Foreign key to orders table
	ProductID uuid.UUID `json:"product_id" db:"product_id"` // Foreign key to products table
	Quantity  int       `json:"quantity" db:"quantity"`     // Quantity of the product ordered
	Price     float64   `json:"price" db:"price"`           // Price of the product at the time of order
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Optional: Include product details in the response (requires JOIN)
	// ProductName string `json:"product_name,omitempty" db:"product_name"`
}
