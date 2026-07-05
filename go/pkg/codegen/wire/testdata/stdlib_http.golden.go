package main

import (
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
	"log"
	"net/http"
)

func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}
	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", debugviz.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), debugviz.HTTPMiddlewareConfig{ServiceName: "my-app"})); err != nil {
		log.Fatal(err)
	}
}
