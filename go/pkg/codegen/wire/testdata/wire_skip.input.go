//debugviz:wire skip
package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("listening")
	http.ListenAndServe(":8080", http.NewServeMux())
}
