package main

import (
	"log"
	"net/http"

	"bullet-cloud-api/internal/handlers"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()

	// essa rota deu trabalho, handlers handlers
	router.HandleFunc("/products", handlers.GetAllProducts).Methods("GET")
	router.HandleFunc("/products", handlers.CreateProduct).Methods("POST")
	router.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
	router.HandleFunc("/products/{id}", handlers.UpdateProduct).Methods("PUT")
	router.HandleFunc("/products/{id}", handlers.DeleteProduct).Methods("DELETE")
	//
	// checar a saúde do jovem
	router.HandleFunc("/health", handlers.HealthCheck).Methods("GET")

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
