package main

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"os"

	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	"github.com/Onellan/tockrplatform/internal/platform/assertion"
	httpserver "github.com/Onellan/tockrplatform/internal/platform/http"
)

func main() {
	path := os.Getenv("PLATFORM_DB_PATH")
	if path == "" {
		path = "platform.db"
	}
	keyText := os.Getenv("PLATFORM_MFA_KEY")
	if keyText == "" {
		log.Fatal("PLATFORM_MFA_KEY must be configured as 64 hex characters")
	}
	secretKey, err := hex.DecodeString(keyText)
	if err != nil || len(secretKey) != 32 {
		log.Fatal("PLATFORM_MFA_KEY must be configured as 64 hex characters")
	}
	store, err := sqlite.OpenWithKey(context.Background(), path, secretKey)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	assertionConfig, err := assertion.ConfigFromEnvironment(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	assertionIssuer, err := assertion.New(assertionConfig)
	if err != nil {
		log.Fatal(err)
	}
	server := httpserver.NewServer(store, httpserver.Config{
		AllowInsecureCookies: os.Getenv("PLATFORM_ALLOW_INSECURE_COOKIES") == "1",
		RateLimitEnabled:     true,
		AssertionIssuer:      assertionIssuer,
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
