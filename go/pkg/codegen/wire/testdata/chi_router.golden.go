package router

import (
	"net/http"

	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
	"github.com/go-chi/chi/v5"
)

func New(serviceName string) http.Handler {
	r := chi.NewRouter()
	r.Use(debugviz.ChiMiddleware(debugviz.HTTPMiddlewareConfig{ServiceName: serviceName}))
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}
