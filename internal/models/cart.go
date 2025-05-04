package models

import (
	"time"

	"github.com/google/uuid"
)

// Cart represents a user's shopping cart.
type Cart struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"` // Foreign key to users table
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
