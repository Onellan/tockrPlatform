// Package config owns the Platform process configuration boundary.
package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	defaultDBPath       = "platform.db"
	defaultHTTPAddr     = ":8080"
	defaultMaxHeaders   = 1 << 20
	defaultReadHeader   = 10 * time.Second
	defaultReadTimeout  = 15 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultIdleTimeout  = 120 * time.Second
	defaultShutdown     = 10 * time.Second
)

var ErrConfiguration = errors.New("invalid Platform runtime configuration")

type Config struct {
	DBPath              string
	MFAKey              []byte
	HTTPAddr            string
	AllowInsecureCookie bool
	MaxHeaderBytes      int
	ReadHeaderTimeout   time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	ShutdownTimeout     time.Duration
}

// FromEnvironment validates process configuration without echoing secret
// material. Defaults are safe for local development and the container sets an
// explicit persistent database path.
func FromEnvironment(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, fmt.Errorf("%w: environment reader", ErrConfiguration)
	}

	dbPath := strings.TrimSpace(getenv("PLATFORM_DB_PATH"))
	if dbPath == "" {
		dbPath = defaultDBPath
	}
	if strings.IndexFunc(dbPath, func(r rune) bool { return r == '\x00' }) >= 0 {
		return Config{}, fmt.Errorf("%w: database path", ErrConfiguration)
	}

	keyText := strings.TrimSpace(getenv("PLATFORM_MFA_KEY"))
	key, err := hex.DecodeString(keyText)
	if err != nil || len(key) != 32 {
		return Config{}, fmt.Errorf("%w: PLATFORM_MFA_KEY must be configured as 64 hex characters", ErrConfiguration)
	}

	addr := strings.TrimSpace(getenv("PLATFORM_HTTP_ADDR"))
	if addr == "" {
		addr = defaultHTTPAddr
	}
	if err := validateHTTPAddr(addr); err != nil {
		return Config{}, fmt.Errorf("%w: PLATFORM_HTTP_ADDR", ErrConfiguration)
	}

	allowInsecure, err := parseBoolean(getenv("PLATFORM_ALLOW_INSECURE_COOKIES"))
	if err != nil {
		return Config{}, fmt.Errorf("%w: PLATFORM_ALLOW_INSECURE_COOKIES", ErrConfiguration)
	}

	return Config{
		DBPath:              dbPath,
		MFAKey:              append([]byte(nil), key...),
		HTTPAddr:            addr,
		AllowInsecureCookie: allowInsecure,
		MaxHeaderBytes:      defaultMaxHeaders,
		ReadHeaderTimeout:   defaultReadHeader,
		ReadTimeout:         defaultReadTimeout,
		WriteTimeout:        defaultWriteTimeout,
		IdleTimeout:         defaultIdleTimeout,
		ShutdownTimeout:     defaultShutdown,
	}, nil
}

func validateHTTPAddr(addr string) error {
	if strings.IndexFunc(addr, func(r rune) bool { return r == '\x00' || r == '\r' || r == '\n' || r == ' ' || r == '\t' }) >= 0 {
		return errors.New("address contains whitespace or control characters")
	}
	_, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.New("address must contain a host and port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("address port is invalid")
	}
	return nil
}

func parseBoolean(value string) (bool, error) {
	switch strings.TrimSpace(value) {
	case "", "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, errors.New("boolean must be 0 or 1")
	}
}
