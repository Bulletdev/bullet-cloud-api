package orders

import (
	"bullet-cloud-api/internal/models"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOrderNotFound          = errors.New("order not found")
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in its current status")
)

// OrderRepository defines the interface for order data operations.
type OrderRepository interface {
	// CreateOrderFromCart creates a new order based on the items in a user's cart.
	// It requires the cart items and the chosen shipping address ID.
	// Returns the newly created order.
	CreateOrderFromCart(ctx context.Context, userID, cartID, shippingAddressID uuid.UUID, cartItems []models.CartItem) (*models.Order, error)

	// FindUserOrders retrieves all orders for a specific user, ordered by creation date.
	FindUserOrders(ctx context.Context, userID uuid.UUID) ([]models.Order, error)

	// FindOrderByID retrieves a specific order by its ID, including its items.
	FindOrderByID(ctx context.Context, orderID uuid.UUID) (*models.Order, []models.OrderItem, error)

	// UpdateOrderStatus changes the status of an existing order.
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status models.OrderStatus) error

	// UpdateOrderTracking updates the tracking number for an order.
	UpdateOrderTracking(ctx context.Context, orderID uuid.UUID, trackingNumber string) error
}

// postgresOrderRepository implements OrderRepository using PostgreSQL.
type postgresOrderRepository struct {
	db *pgxpool.Pool
}

// NewPostgresOrderRepository creates a new instance of postgresOrderRepository.
func NewPostgresOrderRepository(db *pgxpool.Pool) OrderRepository {
	return &postgresOrderRepository{db: db}
}

// CreateOrderFromCart handles the creation of an order within a transaction.
func (r *postgresOrderRepository) CreateOrderFromCart(ctx context.Context, userID, cartID, shippingAddressID uuid.UUID, cartItems []models.CartItem) (*models.Order, error) {
	if len(cartItems) == 0 {
		return nil, errors.New("cannot create order from empty cart")
	}

	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) // Ensure rollback on error

	// 1. Calculate total price
	var total float64
	for _, item := range cartItems {
		total += item.Price * float64(item.Quantity)
	}

	// 2. Create the order record
	orderQuery := `
		INSERT INTO orders (user_id, shipping_address_id, status, total)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, created_at, updated_at
	`
	order := &models.Order{
		UserID:            userID,
		ShippingAddressID: shippingAddressID,
		Total:             total,
	}
	err = tx.QueryRow(ctx, orderQuery,
		userID,
		shippingAddressID,
		models.StatusPending, // Initial status
		total,
	).Scan(&order.ID, &order.Status, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		// Handle potential FK violations (user_id, shipping_address_id invalid)
		return nil, err
	}

	// 3. Create order items from cart items
	orderItemQuery := `
		INSERT INTO order_items (order_id, product_id, quantity, price)
		VALUES ($1, $2, $3, $4)
	`
	batch := &pgx.Batch{}
	for _, item := range cartItems {
		batch.Queue(orderItemQuery, order.ID, item.ProductID, item.Quantity, item.Price)
	}

	results := tx.SendBatch(ctx, batch)
	// Check results for errors
	for i := 0; i < len(cartItems); i++ {
		_, errItem := results.Exec()
		if errItem != nil {
			results.Close() // Important to close batch results
			// Handle potential errors like product_id FK violation
			return nil, errItem
		}
	}
	errClose := results.Close() // Close the batch results
	if errClose != nil {
		return nil, errClose
	}

	// 4. Clear the cart (important: use the original cartID)
	clearCartQuery := `DELETE FROM cart_items WHERE cart_id = $1`
	_, errClear := tx.Exec(ctx, clearCartQuery, cartID)
	if errClear != nil {
		// Don't necessarily fail the order if clearing cart fails, but log it
		// log.Printf("Warning: failed to clear cart %s after creating order %s: %v", cartID, order.ID, errClear)
		// Consider if this should be a fatal error for the transaction
		return nil, errClear // For now, treat as failure
	}

	// 5. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return order, nil
}

// FindUserOrders retrieves orders for a user.
func (r *postgresOrderRepository) FindUserOrders(ctx context.Context, userID uuid.UUID) ([]models.Order, error) {
	query := `
		SELECT id, user_id, shipping_address_id, status, total, tracking_number, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Order])
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// FindOrderByID retrieves a single order and its items.
func (r *postgresOrderRepository) FindOrderByID(ctx context.Context, orderID uuid.UUID) (*models.Order, []models.OrderItem, error) {
	// Use a transaction to ensure consistency
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	// Get order details
	orderQuery := `
		SELECT id, user_id, shipping_address_id, status, total, tracking_number, created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	order := &models.Order{}
	err = tx.QueryRow(ctx, orderQuery, orderID).Scan(
		&order.ID, &order.UserID, &order.ShippingAddressID, &order.Status, &order.Total, &order.TrackingNumber, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrOrderNotFound
		}
		return nil, nil, err
	}

	// Get order items
	itemsQuery := `
		SELECT id, order_id, product_id, quantity, price, created_at, updated_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC
	`
	rows, err := tx.Query(ctx, itemsQuery, orderID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.OrderItem])
	if err != nil {
		return nil, nil, err
	}

	// Commit (read-only transaction, could use Query instead of Begin/Commit)
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return order, items, nil
}

// UpdateOrderStatus changes the status.
func (r *postgresOrderRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status models.OrderStatus) error {
	query := `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	// Optional: Add checks to prevent invalid status transitions (e.g., cannot cancel if shipped)
	// WHERE id = $2 AND status NOT IN ('shipped', 'delivered', 'cancelled')

	result, err := r.db.Exec(ctx, query, status, orderID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		// Could be not found, or status prevented update
		// Check if order exists to differentiate
		_, _, findErr := r.FindOrderByID(ctx, orderID)
		if findErr == ErrOrderNotFound {
			return ErrOrderNotFound
		} else if findErr != nil {
			return findErr // Error during the check
		}
		// If order exists but wasn't updated, assume status prevented it
		return ErrOrderCannotBeCancelled // Or a more generic "status update failed"
	}
	return nil
}

// UpdateOrderTracking updates the tracking number.
func (r *postgresOrderRepository) UpdateOrderTracking(ctx context.Context, orderID uuid.UUID, trackingNumber string) error {
	query := `
        UPDATE orders
        SET tracking_number = $1, updated_at = NOW()
        WHERE id = $2
    `
	// Optional: Only allow setting tracking if status is appropriate (e.g., 'processing' or 'shipped')
	// WHERE id = $2 AND status IN ('processing', 'shipped')

	result, err := r.db.Exec(ctx, query, trackingNumber, orderID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		// Check if order exists to return correct error
		_, _, findErr := r.FindOrderByID(ctx, orderID)
		if findErr == ErrOrderNotFound {
			return ErrOrderNotFound
		} else if findErr != nil {
			return findErr
		}
		// If exists but not updated, maybe status prevented it (if check added)
		return errors.New("failed to update tracking number, order might not exist or status inappropriate")
	}
	return nil
}
