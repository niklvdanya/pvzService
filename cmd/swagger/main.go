package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	swaggerAddress = "localhost:8081"
)

func main() {
	mux := chi.NewMux()

	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile("./pkg/api/contract.swagger.json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("failed to read swagger.json: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(b); err != nil {
			log.Printf("failed to write swagger.json response: %v", err)
		}
	})

	mux.HandleFunc("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger.json"),
	))

	log.Printf("Listening on %s", swaggerAddress)
	if err := http.ListenAndServe(swaggerAddress, mux); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}
