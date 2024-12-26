package main

import (
    "bullet-cloud-api/internal/handlers"
    "log"
    "net/http"
    "os"

    "github.com/gorilla/mux"
) 

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health", handlers.HealthCheck).Methods("GET")
    r.HandleFunc("/products", handlers.GetAllProducts).Methods("GET")
    r.HandleFunc("/products", handlers.CreateProduct).Methods("POST")
    r.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
    r.HandleFunc("/products/{id}", handlers.UpdateProduct).Methods("PUT")
    r.HandleFunc("/products/{id}", handlers.DeleteProduct).Methods("DELETE")

    port := os.Getenv("API_PORT")
    if port == "" {
        port = "4444" // Porta padrão, caso não esteja definida no ambiente
    }

    log.Printf("Server starting on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}
