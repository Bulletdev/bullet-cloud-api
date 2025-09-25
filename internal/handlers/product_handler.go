package handlers

import (
	"bullet-cloud-api/internal/models"
	"bullet-cloud-api/internal/products" // Product Repository
	"bullet-cloud-api/internal/webutils" // JSON Helpers
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ProductHandler handles product-related requests.
type ProductHandler struct {
	ProductRepo products.ProductRepository
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(productRepo products.ProductRepository) *ProductHandler {
	return &ProductHandler{ProductRepo: productRepo}
}

// --- Request Structs (for Create/Update) ---

type CreateProductRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Price       float64    `json:"price"`
	CategoryID  *uuid.UUID `json:"category_id"` // Optional
}

type UpdateProductRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Price       float64    `json:"price"`
	CategoryID  *uuid.UUID `json:"category_id"` // Optional
}

// --- Handlers ---

// CreateProduct handles POST requests to create a new product.
// This typically requires authentication.
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	// --- Specific Validations ---
	// Validate Name (must not be empty)
	if req.Name == "" {
		webutils.ErrorJSON(w, errors.New("product name is required"), http.StatusBadRequest)
		return
	}
	// Validate Price (must be positive)
	if req.Price <= 0 {
		webutils.ErrorJSON(w, errors.New("product price must be positive"), http.StatusBadRequest)
		return
	}
	// TODO: Add more validations if needed (e.g., description length, category exists)
	// --- End Validations ---

	newProduct := &models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		CategoryID:  req.CategoryID,
	}

	createdProduct, err := h.ProductRepo.Create(r.Context(), newProduct)
	if err != nil {
		// TODO: Handle specific DB errors (e.g., FK violation)
		webutils.ErrorJSON(w, errors.New("failed to create product"), http.StatusInternalServerError)
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, createdProduct)
}

// GetAllProducts handles GET requests to list all products.
// This is often a public endpoint.
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	// TODO: Add pagination and filtering based on query parameters (r.URL.Query())
	productList, err := h.ProductRepo.FindAll(r.Context())
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to retrieve products"), http.StatusInternalServerError)
		return
	}

	webutils.WriteJSON(w, http.StatusOK, productList)
}

// SearchProducts handles GET requests to search products by query.
// This is often a public endpoint.
func (h *ProductHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		webutils.ErrorJSON(w, errors.New("search query parameter 'q' is required"), http.StatusBadRequest)
		return
	}

	productList, err := h.ProductRepo.Search(r.Context(), query)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to search products"), http.StatusInternalServerError)
		return
	}

	webutils.WriteJSON(w, http.StatusOK, productList)
}

// GetProduct handles GET requests for a specific product by ID.
// This is often a public endpoint.
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		webutils.ErrorJSON(w, errors.New("product ID is required"), http.StatusBadRequest)
		return
	}

	productID, err := uuid.Parse(idStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid product ID format"), http.StatusBadRequest)
		return
	}

	product, err := h.ProductRepo.FindByID(r.Context(), productID)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to retrieve product"), http.StatusInternalServerError)
		}
		return
	}

	webutils.WriteJSON(w, http.StatusOK, product)
}

// UpdateProduct handles PUT requests to update an existing product.
// This typically requires authentication.
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		webutils.ErrorJSON(w, errors.New("product ID is required"), http.StatusBadRequest)
		return
	}

	productID, err := uuid.Parse(idStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid product ID format"), http.StatusBadRequest)
		return
	}

	var req UpdateProductRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	// Basic Validation
	if req.Name == "" || req.Price < 0 {
		webutils.ErrorJSON(w, errors.New("product name is required and price must be non-negative"), http.StatusBadRequest)
		return
	}

	productToUpdate := &models.Product{
		// ID is set by the repository based on the URL param
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		CategoryID:  req.CategoryID,
	}

	updatedProduct, err := h.ProductRepo.Update(r.Context(), productID, productToUpdate)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			// TODO: Handle other specific errors like FK violation
			webutils.ErrorJSON(w, errors.New("failed to update product"), http.StatusInternalServerError)
		}
		return
	}

	webutils.WriteJSON(w, http.StatusOK, updatedProduct)
}

// DeleteProduct handles DELETE requests for a specific product.
// This typically requires authentication.
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		webutils.ErrorJSON(w, errors.New("product ID is required"), http.StatusBadRequest)
		return
	}

	productID, err := uuid.Parse(idStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid product ID format"), http.StatusBadRequest)
		return
	}

	err = h.ProductRepo.Delete(r.Context(), productID)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to delete product"), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content on successful deletion
}

// TODO: Implement handlers for other product endpoints:
// GetFeaturedProducts (GET /api/products/featured)
// GetProductsByCategory (GET /api/products/category/{categoryId})
