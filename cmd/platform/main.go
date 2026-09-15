package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	httpserver "github.com/Onellan/tockrplatform/internal/platform/http"
)

func main() {
	path := os.Getenv("PLATFORM_DB_PATH")
	if path == "" {
		path = "platform.db"
	}
	store, err := sqlite.Open(context.Background(), path)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	server := httpserver.NewServer(store, httpserver.Config{
		AllowInsecureCookies: os.Getenv("PLATFORM_ALLOW_INSECURE_COOKIES") == "1",
		RateLimitEnabled:     true,
	})
	addr := os.Getenv("PLATFORM_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("tockr Platform listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
