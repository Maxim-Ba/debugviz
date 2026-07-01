package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/router"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}

	handler := debugviz.HTTPMiddleware(router.New(), debugviz.HTTPMiddlewareConfig{
		ServiceName: envOr("DEBUGVIZ_SERVICE_NAME", "demo-http"),
	})

	log.Println("demo/http listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
