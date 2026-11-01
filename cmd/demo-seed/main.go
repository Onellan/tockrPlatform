package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	"github.com/Onellan/tockrplatform/internal/demoseed"
	platformconfig "github.com/Onellan/tockrplatform/internal/platform/config"
)

func main() {
	manifestPath := flag.String("manifest", "", "write the local demo fixture manifest to this path")
	flag.Parse()
	if *manifestPath == "" {
		fatal("--manifest is required")
	}

	cfg, err := platformconfig.FromEnvironment(os.Getenv)
	if err != nil {
		fatal("load Platform configuration: %v", err)
	}
	ctx := context.Background()
	persistence, err := sqlite.OpenWithKey(ctx, cfg.DBPath, cfg.MFAKey)
	if err != nil {
		fatal("open Platform database: %v", err)
	}
	defer func() { _ = persistence.Close() }()

	fixture, err := demoseed.Seed(ctx, persistence)
	if err != nil {
		fatal("seed local demo fixture: %v", err)
	}
	encoded, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		fatal("encode local demo manifest: %v", err)
	}
	if *manifestPath == "-" {
		if _, err := os.Stdout.Write(append(encoded, '\n')); err != nil {
			fatal("write local demo manifest: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Platform demo fixture %s is ready (%d users, %d products).\n", fixture.Fixture, len(fixture.Users), len(fixture.Products))
		return
	}
	path := filepath.Clean(*manifestPath)
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		fatal("write local demo manifest: %v", err)
	}
	fmt.Printf("Platform demo fixture %s is ready (%d users, %d products).\n", fixture.Fixture, len(fixture.Users), len(fixture.Products))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
