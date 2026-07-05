package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})); err != nil {
		log.Fatal(err)
	}
}
