package cart

import (
	"bullet-cloud-api/internal/models"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCartNotFound     = errors.New("cart not found")
	ErrCartItemNotFound = errors.New("cart item not found")
	ErrProductNotInCart = errors.New("product not found in cart")
)

// CartRepository defines the interface for cart data operations.
type CartRepository interface {
	// GetOrCreateCartByUserID finds the cart for a user or creates one if it doesn't exist.
	GetOrCreateCartByUserID(ctx context.Context, userID uuid.UUID) (*models.Cart, error)
	// GetCartItems retrieves all items currently in the specified cart.
	GetCartItems(ctx context.Context, cartID uuid.UUID) ([]models.CartItem, error)
	// AddItem adds a product to the cart or updates its quantity if it already exists.
	AddItem(ctx context.Context, cartID, productID uuid.UUID, quantity int, price float64) (*models.CartItem, error)
	// UpdateItemQuantity changes the quantity of an existing item in the cart.
	UpdateItemQuantity(ctx context.Context, cartID, productID uuid.UUID, quantity int) (*models.CartItem, error)
	// RemoveItem removes a specific product from the cart.
	RemoveItem(ctx context.Context, cartID, productID uuid.UUID) error
	// ClearCart removes all items from a specific cart.
	ClearCart(ctx context.Context, cartID uuid.UUID) error
	// FindCartItem retrieves a specific item from a cart.
	FindCartItem(ctx context.Context, cartID, productID uuid.UUID) (*models.CartItem, error)
}

// postgresCartRepository implements CartRepository using PostgreSQL.
type postgresCartRepository struct {
	db *pgxpool.Pool
}

// NewPostgresCartRepository creates a new instance of postgresCartRepository.
func NewPostgresCartRepository(db *pgxpool.Pool) CartRepository {
	return &postgresCartRepository{db: db}
}

// GetOrCreateCartByUserID finds or creates a cart for the user.
func (r *postgresCartRepository) GetOrCreateCartByUserID(ctx context.Context, userID uuid.UUID) (*models.Cart, error) {
	// Try to find existing cart
	queryFind := `SELECT id, user_id, created_at, updated_at FROM carts WHERE user_id = $1`
	cart := &models.Cart{}
	err := r.db.QueryRow(ctx, queryFind, userID).Scan(
		&cart.ID, &cart.UserID, &cart.CreatedAt, &cart.UpdatedAt,
	)

	if err == nil {
		return cart, nil // Cart found
	}

	// If no cart found, create one
	if errors.Is(err, pgx.ErrNoRows) {
		queryCreate := `
			INSERT INTO carts (user_id)
			VALUES ($1)
			RETURNING id, user_id, created_at, updated_at
		`
		errCreate := r.db.QueryRow(ctx, queryCreate, userID).Scan(
			&cart.ID, &cart.UserID, &cart.CreatedAt, &cart.UpdatedAt,
		)
		if errCreate != nil {
			// Handle potential unique constraint violation if called concurrently (unlikely with user_id unique)
			return nil, errCreate
		}
		return cart, nil // Cart created
	}

	// Other unexpected error during find
	return nil, err
}

// GetCartItems retrieves all items for a given cart ID.
func (r *postgresCartRepository) GetCartItems(ctx context.Context, cartID uuid.UUID) ([]models.CartItem, error) {
	// Query includes JOIN to get product details (optional, adjust fields as needed)
	// query := `
	// 	SELECT ci.id, ci.cart_id, ci.product_id, ci.quantity, ci.price, ci.created_at, ci.updated_at,
	// 	       p.name as product_name, p.description as product_description -- Example JOIN
	// 	FROM cart_items ci
	// 	JOIN products p ON ci.product_id = p.id
	// 	WHERE ci.cart_id = $1
	// 	ORDER BY ci.created_at ASC
	// `
	query := `
		SELECT id, cart_id, product_id, quantity, price, created_at, updated_at
		FROM cart_items
		WHERE cart_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.CartItem])
	if err != nil {
		return nil, err
	}
	return items, nil
}

// AddItem adds or updates a product in the cart.
func (r *postgresCartRepository) AddItem(ctx context.Context, cartID, productID uuid.UUID, quantity int, price float64) (*models.CartItem, error) {
	query := `
		INSERT INTO cart_items (cart_id, product_id, quantity, price)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (cart_id, product_id) DO UPDATE SET
			quantity = cart_items.quantity + EXCLUDED.quantity,
			price = EXCLUDED.price, -- Update price in case it changed
			updated_at = NOW()
		RETURNING id, cart_id, product_id, quantity, price, created_at, updated_at
	`
	item := &models.CartItem{}
	err := r.db.QueryRow(ctx, query, cartID, productID, quantity, price).Scan(
		&item.ID,
		&item.CartID,
		&item.ProductID,
		&item.Quantity,
		&item.Price,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		// Handle potential FK violations (cart_id or product_id invalid)
		return nil, err
	}
	return item, nil
}

// UpdateItemQuantity updates the quantity of a specific item.
func (r *postgresCartRepository) UpdateItemQuantity(ctx context.Context, cartID, productID uuid.UUID, quantity int) (*models.CartItem, error) {
	if quantity <= 0 {
		// If quantity is zero or less, remove the item instead
		return nil, r.RemoveItem(ctx, cartID, productID)
	}

	query := `
		UPDATE cart_items
		SET quantity = $1, updated_at = NOW()
		WHERE cart_id = $2 AND product_id = $3
		RETURNING id, cart_id, product_id, quantity, price, created_at, updated_at
	`
	item := &models.CartItem{}
	err := r.db.QueryRow(ctx, query, quantity, cartID, productID).Scan(
		&item.ID,
		&item.CartID,
		&item.ProductID,
		&item.Quantity,
		&item.Price,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotInCart
		}
		return nil, err
	}
	return item, nil
}

// RemoveItem deletes an item from the cart.
func (r *postgresCartRepository) RemoveItem(ctx context.Context, cartID, productID uuid.UUID) error {
	query := `DELETE FROM cart_items WHERE cart_id = $1 AND product_id = $2`
	result, err := r.db.Exec(ctx, query, cartID, productID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProductNotInCart // Item wasn't in the cart
	}
	return nil
}

// ClearCart removes all items from a given cart.
func (r *postgresCartRepository) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	query := `DELETE FROM cart_items WHERE cart_id = $1`
	_, err := r.db.Exec(ctx, query, cartID)
	// We don't necessarily return an error if cart was already empty
	return err
}

// FindCartItem retrieves a specific item from a cart.
func (r *postgresCartRepository) FindCartItem(ctx context.Context, cartID, productID uuid.UUID) (*models.CartItem, error) {
	query := `
		SELECT id, cart_id, product_id, quantity, price, created_at, updated_at
		FROM cart_items
		WHERE cart_id = $1 AND product_id = $2
	`
	item := &models.CartItem{}
	err := r.db.QueryRow(ctx, query, cartID, productID).Scan(
		&item.ID,
		&item.CartID,
		&item.ProductID,
		&item.Quantity,
		&item.Price,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotInCart
		}
		return nil, err
	}
	return item, nil
}
