package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateProduct(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Valid request",
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("PUT", "/products/1", strings.NewReader(`{"name":"Updated Product"}`)),
			},
		},
		{
			name: "Invalid request",
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("PUT", "/products/1", strings.NewReader(`{"invalid":"data"}`)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateProduct(tt.args.w, tt.args.r)
		})
	}
}
