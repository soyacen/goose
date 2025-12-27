package goose

import (
	"net/http"
)

func AppendHealth(router *http.ServeMux) *http.ServeMux {
	router.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {})
	return router
}
