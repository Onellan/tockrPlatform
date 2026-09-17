package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	"github.com/Onellan/tockrplatform/internal/platform/assertion"
	platformconfig "github.com/Onellan/tockrplatform/internal/platform/config"
	httpserver "github.com/Onellan/tockrplatform/internal/platform/http"
)

func main() {
	if err := run(context.Background(), os.Getenv); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(parent context.Context, getenv func(string) string) error {
	return runWithListener(parent, getenv, net.Listen)
}

func runWithListener(parent context.Context, getenv func(string) string, listen func(string, string) (net.Listener, error)) error {
	cfg, err := platformconfig.FromEnvironment(getenv)
	if err != nil {
		return err
	}
	assertionConfig, err := assertion.ConfigFromEnvironment(getenv)
	if err != nil {
		return err
	}
	assertionIssuer, err := assertion.New(assertionConfig)
	if err != nil {
		return err
	}
	store, err := sqlite.OpenWithKey(parent, cfg.DBPath, cfg.MFAKey)
	if err != nil {
		return err
	}
	defer store.Close()

	server := httpserver.NewServer(store, httpserver.Config{
		AllowInsecureCookies: cfg.AllowInsecureCookie,
		RateLimitEnabled:     true,
		AssertionIssuer:      assertionIssuer,
		ReadinessCheck:       store.DB().PingContext,
		ReadAuthorityKeys:    cfg.ReadAuthorityKeys,
	})
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
	listener, err := listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return err
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.Serve(listener)
	}()

	stopContext, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stopContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			return err
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
