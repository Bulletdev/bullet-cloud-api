package models

import (
	"reflect"
	"testing"
)

func TestNewProductRepository(t *testing.T) {
	tests := []struct {
		name string
		want *ProductRepository
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewProductRepository(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProductRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}
