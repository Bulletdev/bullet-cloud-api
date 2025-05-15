package database

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewConnection establishes a connection pool to the PostgreSQL database.
func NewConnection(databaseURL string) (*pgxpool.Pool, error) {
	// Parse the connection string
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Printf("Unable to parse database config: %v\n", err)
		return nil, err
	}

	// Apply connection pool limits to reduce memory usage (important for low-RAM environments like Render Free)
	config.MaxConns = 4                       // Hard cap on max simultaneous DB connections
	config.MinConns = 1                       // Keep 1 connection alive to reduce cold starts
	config.MaxConnLifetime = 30 * time.Minute // Recycle connections after 30min
	config.MaxConnIdleTime = 5 * time.Minute  // Close idle connections after 5min

	// Create a new connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Printf("Unable to create connection pool: %v\n", err)
		return nil, err
	}

	// Ping the database to verify the connection
	err = pool.Ping(context.Background())
	if err != nil {
		log.Printf("Unable to ping database: %v\n", err)
		pool.Close()
		return nil, err
	}

	log.Println("Successfully connected to the database!")
	return pool, nil
}
