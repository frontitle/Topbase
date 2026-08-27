package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/topbase/topbase/internal/platform/httpapi"
)

func main() {
	address := os.Getenv("TOPBASE_ADDR")
	if address == "" {
		address = ":8080"
	}
	server := &http.Server{Addr: address, Handler: httpapi.NewServer(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("Topbase listening on http://localhost%s", address)
	log.Fatal(server.ListenAndServe())
}
