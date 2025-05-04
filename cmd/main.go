package main

import (
	"bullet-cloud-api/internal/addresses"
	"bullet-cloud-api/internal/auth"
	"bullet-cloud-api/internal/cart"
	"bullet-cloud-api/internal/categories"
	"bullet-cloud-api/internal/config"
	"bullet-cloud-api/internal/database"
	"bullet-cloud-api/internal/handlers"
	"bullet-cloud-api/internal/orders"
	"bullet-cloud-api/internal/products"
	"bullet-cloud-api/internal/users"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

const defaultPort = "4444"
const defaultJWTExpiry = 24 * time.Hour

func main() {
	cfg := config.Load()

	dbPool, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}
	defer dbPool.Close()

	userRepo := users.NewPostgresUserRepository(dbPool)
	productRepo := products.NewPostgresProductRepository(dbPool)
	categoryRepo := categories.NewPostgresCategoryRepository(dbPool)
	addressRepo := addresses.NewPostgresAddressRepository(dbPool)
	cartRepo := cart.NewPostgresCartRepository(dbPool)
	orderRepo := orders.NewPostgresOrderRepository(dbPool)
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret, defaultJWTExpiry)
	userHandler := handlers.NewUserHandler(userRepo, addressRepo)
	productHandler := handlers.NewProductHandler(productRepo)
	categoryHandler := handlers.NewCategoryHandler(categoryRepo)
	cartHandler := handlers.NewCartHandler(cartRepo, productRepo)
	orderHandler := handlers.NewOrderHandler(orderRepo, cartRepo, addressRepo)
	authMiddleware := auth.NewMiddleware(cfg.JWTSecret, userRepo)

	r := setupRoutes(authHandler, userHandler, productHandler, categoryHandler, cartHandler, orderHandler, authMiddleware)

	port := os.Getenv("API_PORT")
	if port == "" {
		port = defaultPort
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-done
	log.Println("Server shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %+v", err)
	}

	log.Println("Server exited properly")
}

func setupRoutes(ah *handlers.AuthHandler, uh *handlers.UserHandler, ph *handlers.ProductHandler, ch *handlers.CategoryHandler, cartH *handlers.CartHandler, oh *handlers.OrderHandler, mw *auth.Middleware) *mux.Router {
	r := mux.NewRouter()

	apiV1 := r.PathPrefix("/api").Subrouter()

	apiV1.HandleFunc("/health", handlers.HealthCheck).Methods("GET")
	apiV1.HandleFunc("/auth/register", ah.Register).Methods("POST")
	apiV1.HandleFunc("/auth/login", ah.Login).Methods("POST")
	apiV1.HandleFunc("/products", ph.GetAllProducts).Methods("GET")
	apiV1.HandleFunc("/products/{id:[0-9a-fA-F-]+}", ph.GetProduct).Methods("GET")
	apiV1.HandleFunc("/categories", ch.GetAllCategories).Methods("GET")
	apiV1.HandleFunc("/categories/{id:[0-9a-fA-F-]+}", ch.GetCategory).Methods("GET")

	protectedUserRoutes := apiV1.PathPrefix("/users").Subrouter()
	protectedUserRoutes.Use(mw.Authenticate)
	protectedUserRoutes.HandleFunc("/me", uh.GetMe).Methods("GET")
	protectedUserRoutes.HandleFunc("/{userId:[0-9a-fA-F-]+}/addresses", uh.ListAddresses).Methods("GET")
	protectedUserRoutes.HandleFunc("/{userId:[0-9a-fA-F-]+}/addresses", uh.AddAddress).Methods("POST")
	protectedUserRoutes.HandleFunc("/{userId:[0-9a-fA-F-]+}/addresses/{addressId:[0-9a-fA-F-]+}", uh.UpdateAddress).Methods("PUT")
	protectedUserRoutes.HandleFunc("/{userId:[0-9a-fA-F-]+}/addresses/{addressId:[0-9a-fA-F-]+}", uh.DeleteAddress).Methods("DELETE")
	protectedUserRoutes.HandleFunc("/{userId:[0-9a-fA-F-]+}/addresses/{addressId:[0-9a-fA-F-]+}/default", uh.SetDefaultAddress).Methods("PATCH")

	protectedProductRoutes := apiV1.PathPrefix("/products").Subrouter()
	protectedProductRoutes.Use(mw.Authenticate)
	protectedProductRoutes.HandleFunc("", ph.CreateProduct).Methods("POST")
	protectedProductRoutes.HandleFunc("/{id:[0-9a-fA-F-]+}", ph.UpdateProduct).Methods("PUT")
	protectedProductRoutes.HandleFunc("/{id:[0-9a-fA-F-]+}", ph.DeleteProduct).Methods("DELETE")

	protectedCategoryRoutes := apiV1.PathPrefix("/categories").Subrouter()
	protectedCategoryRoutes.Use(mw.Authenticate)
	protectedCategoryRoutes.HandleFunc("", ch.CreateCategory).Methods("POST")
	protectedCategoryRoutes.HandleFunc("/{id:[0-9a-fA-F-]+}", ch.UpdateCategory).Methods("PUT")
	protectedCategoryRoutes.HandleFunc("/{id:[0-9a-fA-F-]+}", ch.DeleteCategory).Methods("DELETE")

	protectedCartRoutes := apiV1.PathPrefix("/cart").Subrouter()
	protectedCartRoutes.Use(mw.Authenticate)
	protectedCartRoutes.HandleFunc("", cartH.GetCart).Methods("GET")
	protectedCartRoutes.HandleFunc("/items", cartH.AddItem).Methods("POST")
	protectedCartRoutes.HandleFunc("/items/{productId:[0-9a-fA-F-]+}", cartH.UpdateItem).Methods("PUT")
	protectedCartRoutes.HandleFunc("/items/{productId:[0-9a-fA-F-]+}", cartH.DeleteItem).Methods("DELETE")
	protectedCartRoutes.HandleFunc("", cartH.ClearCart).Methods("DELETE")

	protectedOrderRoutes := apiV1.PathPrefix("/orders").Subrouter()
	protectedOrderRoutes.Use(mw.Authenticate)
	protectedOrderRoutes.HandleFunc("", oh.CreateOrder).Methods("POST")
	protectedOrderRoutes.HandleFunc("", oh.ListOrders).Methods("GET")
	protectedOrderRoutes.HandleFunc("/{id:[0-9a-fA-F-]+}", oh.GetOrder).Methods("GET")
	protectedOrderRoutes.HandleFunc("/{id:[0-9a-fA-F-]+}/cancel", oh.CancelOrder).Methods("PATCH")

	return r
}
