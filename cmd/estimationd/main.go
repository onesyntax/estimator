// Command estimationd serves the estimation service over HTTP.
//
// Usage:
//
//	estimationd --ai-provider=mock --addr=:8080
//
// With --ai-provider=mock the deterministic QA priming affordance
// (POST /qa/ai/next-wbs) is enabled and no network LLM is used.
package main

import (
	"flag"
	"log"
	"net/http"

	"estimation/internal/httpapi"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	aiProvider := flag.String("ai-provider", "", "AI provider; use \"mock\" for deterministic QA mode")
	flag.Parse()

	mock := *aiProvider == "mock"
	srv := httpapi.NewServer(mock)

	log.Printf("estimationd listening on %s (ai-provider=%q, mock=%v)", *addr, *aiProvider, mock)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatalf("estimationd: %v", err)
	}
}
