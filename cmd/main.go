package main

import (
	"bullet-cloud-api/internal/handlers"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

const defaultPort = "8080"

func main() {
	r := setupRoutes()

	port := os.Getenv("API_PORT")
	if port == "" {
		port = defaultPort // Porta padrão definida para maior clareza
	}

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func setupRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/health/{id}", handlers.HealthCheck).Methods("GET")
	r.HandleFunc("/products", handlers.GetAllProducts).Methods("GET")
	r.HandleFunc("/products", handlers.CreateProduct).Methods("POST")
	r.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
	r.HandleFunc("/products/{id}", handlers.UpdateProduct).Methods("PUT")
	r.HandleFunc("/products/{id}", handlers.DeleteProduct).Methods("DELETE")
	return r
}
