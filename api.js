const API_URL = 'http://localhost:8080'; //url do app, vou add dps

export const productService = {
  // Listar produtos
  async getProducts() {
    const response = await fetch(`${API_URL}/products`);
    return response.json();
  },

  // Criar produto
  async createProduct(product) {
    const response = await fetch(`${API_URL}/products`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(product),
    });
    return response.json();
  },

  // Buscar produto por ID
  async getProduct(id) {
    const response = await fetch(`${API_URL}/products/${id}`);
    return response.json();
  },

  // Atualizar produto
  async updateProduct(id, product) {
    const response = await fetch(`${API_URL}/products/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(product),
    });
    return response.json();
  },

  // Deletar produto
  async deleteProduct(id) {
    await fetch(`${API_URL}/products/${id}`, {
      method: 'DELETE',
    });
  }, 
};

// Exemplo de componente React so pra deixar de base
import { useEffect, useState } from 'react';

function ProductList() {
  const [products, setProducts] = useState([]);

  useEffect(() => {
    async function loadProducts() {
      const data = await productService.getProducts();
      setProducts(data);
    }
    loadProducts();
  }, []);

  return (
    <div>
      {products.map(product => (
        <div key={product.id}>
          <h3>{product.name}</h3>
          <p>{product.description}</p>
          <p>R$ {product.price}</p>
        </div>
      ))}
    </div>
  );
}
