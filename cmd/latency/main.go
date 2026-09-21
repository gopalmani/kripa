// SPDX-License-Identifier: AGPL-3.0-or-later
// Runs a finite local HTTP latency measurement. Never targets a remote service.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gopalmani/kripa/internal/api"
	"github.com/gopalmani/kripa/internal/ephemeris"
	"github.com/rs/zerolog"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

type result struct {
	Case        string  `json:"case"`
	Requests    int     `json:"requests"`
	Concurrency int     `json:"concurrency"`
	Errors      int     `json:"errors"`
	P50         float64 `json:"p50_ms"`
	P95         float64 `json:"p95_ms"`
	P99         float64 `json:"p99_ms"`
	TargetMet   bool    `json:"p95_under_100ms"`
}

func main() {
	n := flag.Int("n", 200, "requests per case")
	concurrency := flag.Int("c", 4, "bounded concurrent clients")
	enforce := flag.Bool("enforce", false, "exit nonzero on errors or p95 >= 100ms")
	flag.Parse()
	if *n < 1 || *n > 10000 || *concurrency < 1 || *concurrency > 64 {
		panic("n must be 1-10000 and c 1-64")
	}
	engine, err := ephemeris.New(os.Getenv("SWISS_EPHEMERIS_PATH"))
	if err != nil {
		panic(err)
	}
	var results []result
	for _, tc := range []struct {
		name, path string
		cache      int
		vary       bool
	}{{"charts_uncached", "/v1/charts", 0, true}, {"panchang_uncached", "/v1/panchang", 0, true}, {"panchang_warm", "/v1/panchang", 512, false}} {
		h := api.New(engine, api.Config{MaxInflight: 64, CacheEntries: tc.cache, Timeout: 10 * time.Second, Logger: zerolog.Nop()}).Handler()
		server := httptest.NewServer(h)
		client := &http.Client{Transport: &http.Transport{MaxIdleConns: 64, MaxIdleConnsPerHost: 64}, Timeout: 15 * time.Second}
		payload := func(i int) []byte {
			date := "2026-09-21"
			if tc.vary {
				date = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i%3650).Format("2006-01-02")
			}
			v := map[string]any{"date": date, "timezone": "Asia/Kolkata", "latitude": 12.9716, "longitude": 77.5946, "profile": "lahiri_upper_limb_v1"}
			if tc.path == "/v1/charts" {
				v["profile"] = "western_tropical_v1"
				v["time"] = "12:00"
				v["time_status"] = "exact"
			}
			b, _ := json.Marshal(v)
			return b
		}
		do := func(i int) (time.Duration, error) {
			body := payload(i)
			start := time.Now()
			resp, err := client.Post(server.URL+tc.path, "application/json", bytes.NewReader(body))
			if err != nil {
				return time.Since(start), err
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 {
				return time.Since(start), fmt.Errorf("HTTP %d", resp.StatusCode)
			}
			return time.Since(start), nil
		}
		if _, err := do(0); err != nil {
			panic(err)
		}
		jobs := make(chan int)
		times := make([]float64, *n)
		errors := make([]bool, *n)
		var wg sync.WaitGroup
		for j := 0; j < *concurrency; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range jobs {
					d, err := do(i)
					times[i] = float64(d) / float64(time.Millisecond)
					errors[i] = err != nil
				}
			}()
		}
		for i := 0; i < *n; i++ {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		client.CloseIdleConnections()
		server.Close()
		sort.Float64s(times)
		percentile := func(p float64) float64 { idx := int(math.Ceil(p*float64(len(times)))) - 1; return times[idx] }
		count := 0
		for _, e := range errors {
			if e {
				count++
			}
		}
		results = append(results, result{tc.name, *n, *concurrency, count, percentile(.5), percentile(.95), percentile(.99), count == 0 && percentile(.95) < 100})
	}
	out := map[string]any{"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "cpus": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "ephemeris": engine.Version(), "scope": "local HTTP loopback; no public network or reverse proxy; logging disabled", "results": results}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
	if *enforce {
		for _, r := range results {
			if !r.TargetMet {
				os.Exit(1)
			}
		}
	}
}
