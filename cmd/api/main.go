// Command api serves Anarchy openly: agents fetch trained bots without keys.
// Requires DATABASE_URL + ANARCHY_CONTACT_KEY at boot (server-side only).
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"anarchy/internal/api"
)

// loadDotEnv mirrors cmd/server: read .env once, never override real env.
func loadDotEnv() {
	if os.Getenv("DATABASE_URL") != "" && os.Getenv("ANARCHY_CONTACT_KEY") != "" {
		return
	}
	b, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		k, v := strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func main() {
	loadDotEnv()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	h, err := api.Handler()
	if err != nil {
		log.Fatal("api: ", err)
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Println("anarchy api listening on :" + port + " (open, no keys)")
	log.Fatal(srv.ListenAndServe())
}
