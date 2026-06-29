package main

import (
	"log"
	"net/http"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/router"
)

func main() {
	log.Println("demo/http listening on :8080")
	if err := http.ListenAndServe(":8080", router.New()); err != nil {
		log.Fatal(err)
	}
}
