//debugviz:app name=annotated-app
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
	log.Println("listening")
	http.ListenAndServe(":8080", debugviz.HTTPMiddleware(http.NewServeMux(), debugviz.HTTPMiddlewareConfig{ServiceName: "annotated-app"}))
}
