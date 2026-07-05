package main

import (
	"log"
	"net/http"
)

//debugviz:wire http skip
func main() {
	log.Println("listening")
	http.ListenAndServe(":8080", http.NewServeMux())
}
