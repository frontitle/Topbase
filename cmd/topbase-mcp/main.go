package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/topbase/topbase/internal/aimcp"
	"github.com/topbase/topbase/internal/topbaseapi"
)

func main() {
	baseURL := flag.String("url", envOr("TOPBASE_URL", "http://localhost"), "Topbase base URL")
	apiKey := flag.String("api-key", os.Getenv("TOPBASE_API_KEY"), "Topbase API key (prefer TOPBASE_API_KEY)")
	maxRows := flag.Int("max-rows", 200, "maximum rows returned to the AI per query")
	flag.Parse()

	client, err := topbaseapi.New(*baseURL, *apiKey, topbaseapi.WithMaxRows(*maxRows))
	if err != nil {
		fmt.Fprintln(os.Stderr, "topbase-mcp:", err)
		os.Exit(2)
	}
	server := aimcp.New(client)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
