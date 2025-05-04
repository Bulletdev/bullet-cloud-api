package models

import (
	"time"

	"github.com/google/uuid"
)

// Product represents a product in the e-commerce system.
type Product struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Price       float64    `json:"price" db:"price"`             // Use numeric/decimal type in DB for precision
	CategoryID  *uuid.UUID `json:"category_id" db:"category_id"` // Pointer to allow null category initially
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}
