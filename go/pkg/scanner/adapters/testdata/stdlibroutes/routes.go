package stdlibroutes

import "net/http"

func healthHandler(w http.ResponseWriter, _ *http.Request) {}

func Register() {
	http.HandleFunc("/health", healthHandler)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/items/", healthHandler)
}
