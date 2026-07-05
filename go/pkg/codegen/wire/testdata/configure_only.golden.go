package main

import (
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
	"log"
)

func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}
	log.Println("ready")
}
