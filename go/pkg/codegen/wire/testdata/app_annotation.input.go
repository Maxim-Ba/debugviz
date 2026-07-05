//debugviz:app name=annotated-app
package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("listening")
	http.ListenAndServe(":8080", http.NewServeMux())
}
