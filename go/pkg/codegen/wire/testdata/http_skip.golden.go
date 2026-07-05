package main

import (
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
	"log"
	"net/http"
)

//debugviz:wire http skip
func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}
	log.Println("listening")
	http.ListenAndServe(":8080", http.NewServeMux())
}
