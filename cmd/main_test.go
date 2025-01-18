package main

import (
	"bytes"
	"net/http"
	"testing"
)

func Test_main(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "Test case 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main()
			jsonStr := []byte(`{"name":"Notebook Gamer","description":"Notebook para jogos","price":5999.99,"category":"Eletronicos"}`)
			req, err := http.NewRequest("POST", "http://localhost:8080/products", bytes.NewBuffer(jsonStr))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
			}
		})
	}
}
