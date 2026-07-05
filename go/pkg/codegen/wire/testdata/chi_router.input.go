package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func New(serviceName string) http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}
