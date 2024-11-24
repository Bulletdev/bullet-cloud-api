// adicionar CORS para permitir requisições 

package main

import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"magalu-cloud-api/internal/handlers"
)

func main() {
	router := mux.NewRouter()
	
	
	router.HandleFunc("/products", handlers.GetAllProducts).Methods("GET")
	router.HandleFunc("/products", handlers.CreateProduct).Methods("POST")
	router.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
	router.HandleFunc("/products/{id}", handlers.UpdateProduct).Methods("PUT")
	router.HandleFunc("/products/{id}", handlers.DeleteProduct).Methods("DELETE")
	router.HandleFunc("/health", handlers.HealthCheck).Methods("GET")

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://seu-saas.com", "http://localhost:3000"}), // adicione seus domínios
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	handler := corsHandler(router)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
