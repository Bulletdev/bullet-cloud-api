package models

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus defines allowed statuses for an order.
type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"    // Order received, awaiting processing/payment
	StatusProcessing OrderStatus = "processing" // Payment received, order being processed
	StatusShipped    OrderStatus = "shipped"    // Order shipped
	StatusDelivered  OrderStatus = "delivered"  // Order delivered
	StatusCancelled  OrderStatus = "cancelled"  // Order cancelled
)

// Order represents a customer order.
type Order struct {
	ID                uuid.UUID   `json:"id" db:"id"`
	UserID            uuid.UUID   `json:"user_id" db:"user_id"`                           // FK to users
	ShippingAddressID uuid.UUID   `json:"shipping_address_id" db:"shipping_address_id"`   // FK to addresses
	Status            OrderStatus `json:"status" db:"status"`                             // Current status of the order
	Total             float64     `json:"total" db:"total"`                               // Total price of the order at creation
	TrackingNumber    *string     `json:"tracking_number,omitempty" db:"tracking_number"` // Optional tracking number
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
}
