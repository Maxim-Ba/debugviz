package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/handler"
	"github.com/Maxim-Ba/debugviz/demo/http/internal/middleware"
	"github.com/Maxim-Ba/debugviz/demo/http/internal/repository"
	"github.com/Maxim-Ba/debugviz/demo/http/internal/service"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

func New(serviceName string) http.Handler {
	userRepo := repository.NewUserRepository()
	itemRepo := repository.NewItemRepository()

	userSvc := service.NewUserService(userRepo)
	itemSvc := service.NewItemService(itemRepo)

	userHandler := handler.NewUserHandler(userSvc)
	itemHandler := handler.NewItemHandler(itemSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logging)
	if serviceName != "" {
		r.Use(debugviz.ChiMiddleware(debugviz.HTTPMiddlewareConfig{ServiceName: serviceName}))
	}

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/users", func(r chi.Router) {
		r.Get("/", userHandler.List)
		r.Post("/", userHandler.Create)
		r.Get("/{id}", userHandler.GetByID)
	})

	r.Route("/api/items", func(r chi.Router) {
		r.Get("/", itemHandler.List)
		r.Get("/{id}", itemHandler.GetByID)
	})

	return r
}
