package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestRunGracefullyStopsOnParentCancellation(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{
		"PLATFORM_DB_PATH":               filepath.Join(t.TempDir(), "platform.db"),
		"PLATFORM_MFA_KEY":               "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"PLATFORM_HTTP_ADDR":             "127.0.0.1:8080",
		"PLATFORM_ASSERTION_ISSUER":      "https://platform.example.test",
		"PLATFORM_ASSERTION_KEY_ID":      "test",
		"PLATFORM_ASSERTION_PRIVATE_KEY": hex.EncodeToString(privateKey),
		"PLATFORM_ASSERTION_AUDIENCES":   "tockrctrl,tockrims",
	}
	getenv := func(key string) string { return values[key] }
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	listen := func(network, _ string) (net.Listener, error) {
		listener, err := net.Listen(network, "127.0.0.1:0")
		if err == nil {
			close(started)
		}
		return listener, err
	}

	result := make(chan error, 1)
	go func() { result <- runWithListener(parent, getenv, listen) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("runtime did not bind a listener")
	}
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("graceful shutdown error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runtime did not shut down within the bounded timeout")
	}
}
