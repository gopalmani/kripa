// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/gopalmani/kripa/internal/api"
	"github.com/gopalmani/kripa/internal/ephemeris"
	"github.com/rs/zerolog"
)

var version = "development"

func main() {
	health := flag.Bool("healthcheck", false, "check the local liveness endpoint")
	flag.Parse()
	addr := env("KRIPA_LISTEN_ADDR", "127.0.0.1:8080")
	if *health {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			os.Exit(1)
		}
		c := http.Client{Timeout: 2 * time.Second}
		resp, err := c.Get("http://127.0.0.1:" + port + "/health/live")
		if err != nil {
			os.Exit(1)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			os.Exit(1)
		}
		return
	}
	logger := zerolog.New(zerolog.SyncWriter(os.Stdout)).With().Timestamp().Str("service", "kripa").Str("version", version).Logger()
	level, err := zerolog.ParseLevel(env("LOG_LEVEL", "info"))
	if err != nil {
		logger.Fatal().Msg("invalid LOG_LEVEL")
	}
	logger = logger.Level(level)
	token := os.Getenv("KRIPA_API_TOKEN")
	if path := os.Getenv("KRIPA_API_TOKEN_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			logger.Fatal().Msg("cannot read API token file")
		}
		token = strings.TrimSpace(string(b))
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		logger.Fatal().Msg("invalid KRIPA_LISTEN_ADDR")
	}
	ip := net.ParseIP(host)
	if token == "" && (ip == nil || !ip.IsLoopback()) {
		logger.Fatal().Msg("API token required when binding outside loopback")
	}
	if token != "" && (len(token) < 32 || strings.ContainsAny(token, " \t\r\n")) {
		logger.Fatal().Msg("API token must contain at least 32 characters without whitespace")
	}
	cacheN, err := positiveInt("KRIPA_CACHE_ENTRIES", 512, 0, 10000)
	if err != nil {
		logger.Fatal().Err(err).Msg("invalid configuration")
	}
	inflight, err := positiveInt("KRIPA_MAX_INFLIGHT", 32, 1, 256)
	if err != nil {
		logger.Fatal().Err(err).Msg("invalid configuration")
	}
	timeout, err := time.ParseDuration(env("KRIPA_REQUEST_TIMEOUT", "2s"))
	if err != nil || timeout < 10*time.Millisecond || timeout > 10*time.Second {
		logger.Fatal().Msg("KRIPA_REQUEST_TIMEOUT must be 10ms to 10s")
	}
	engine, err := ephemeris.New(env("SWISS_EPHEMERIS_PATH", ".deps/swisseph/ephe"))
	if err != nil {
		logger.Fatal().Err(err).Msg("ephemeris startup validation failed")
	}
	service := api.New(engine, api.Config{Version: version, Token: token, Timeout: timeout, MaxInflight: inflight, CacheEntries: cacheN, CacheTTL: 24 * time.Hour, Logger: logger})
	server := &http.Server{Addr: addr, Handler: service.Handler(), ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 8192}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() {
		logger.Info().Str("address", addr).Str("ephemeris", engine.Version()).Msg("listening")
		done <- server.ListenAndServe()
	}()
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("server stopped")
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			logger.Error().Err(err).Msg("shutdown deadline exceeded")
		}
	}
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func positiveInt(key string, fallback, min, max int) (int, error) {
	v, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil || v < min || v > max {
		return 0, fmt.Errorf("%s outside allowed range", key)
	}
	return v, nil
}
