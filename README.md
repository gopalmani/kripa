# KRIPA

KRIPA is a runnable, standalone Go service for tropical birth charts and location-specific astronomical Panchang, using Swiss Ephemeris directly through cgo. It is intended to power Astrel and BrahminBooking through private backend calls. Neither integration is part of this repository.

**Status:** implemented, with Panchang explicitly marked `astronomical_preview`. This is not a production-validated religious calendar. See [validation evidence](docs/VALIDATION.md), [work plan](docs/PLAN.md), and [OpenAPI](api/openapi.yaml).

## Implemented scope

- `POST /v1/charts`: tropical planets, true nodes, Chiron, retrograde flags, houses, angles and major aspects; no chart cache.
- `POST /v1/panchang`: sunrise/sunset, civil-day moonrise/moonset, sunrise vara and paksha, tithi/nakshatra/yoga/karana transitions, Rahu Kalam, Yamaganda and daytime Gulika.
- Health, metadata and Prometheus metrics; zerolog JSON logs; bounded concurrency, deadlines, strict JSON and an 8 KiB body limit.
- Bounded Panchang LRU (default 512 entries, 24-hour TTL), identical-request coalescing, no coordinate rounding, no database or Redis.

Unsupported: lunar-month naming, Amanta/Purnimanta, adhika/kshaya months, regional festivals, vrats, personalized muhurat and religious recommendations. No accounts, payments, bookings, interpretations or persisted birth records.

## Local quick start

Requires Git, Go 1.26, a C compiler, make, and `sha256sum` or `shasum`. On macOS install Xcode command-line tools; Linux needs build-essential. Setup downloads the exact Swiss source/data commit and builds a static native library. Generated files stay in ignored `.deps/` and `bin/`.

```sh
make setup
make verify
make build
SWISS_EPHEMERIS_PATH="$PWD/.deps/swisseph/ephe" ./bin/kripa
```

Default address is `127.0.0.1:8080`. For a revision-labelled build:

```sh
CGO_ENABLED=1 go build -trimpath -ldflags="-s -w -X main.version=$(git rev-parse HEAD)" -o bin/kripa ./cmd/kripa
```

Working synthetic smoke requests (default unauthenticated loopback development):

```sh
curl --fail-with-body http://127.0.0.1:8080/health/live
curl --fail-with-body http://127.0.0.1:8080/health/ready
curl --fail-with-body http://127.0.0.1:8080/v1/meta
curl --fail-with-body http://127.0.0.1:8080/metrics
curl --fail-with-body http://127.0.0.1:8080/v1/charts -H 'Content-Type: application/json' -d '{"date":"2000-01-01","time":"12:00","timezone":"UTC","time_status":"exact","latitude":51.5,"longitude":0,"profile":"western_tropical_v1"}'
curl --fail-with-body http://127.0.0.1:8080/v1/panchang -H 'Content-Type: application/json' -d '{"date":"2026-09-21","timezone":"Asia/Kolkata","latitude":12.9716,"longitude":77.5946,"profile":"lahiri_upper_limb_v1"}'
```

## Configuration and security

See [.env.example](.env.example). The binary reads environment variables; it does not load `.env` automatically.

| Variable | Default / constraint |
| --- | --- |
| `KRIPA_LISTEN_ADDR` | `127.0.0.1:8080` |
| `SWISS_EPHEMERIS_PATH` | `.deps/swisseph/ephe` |
| `KRIPA_API_TOKEN` | unset; at least 32 characters when set |
| `KRIPA_API_TOKEN_FILE` | optional; trimmed contents override token environment variable |
| `KRIPA_CACHE_ENTRIES` | 512; 0 disables storage, maximum 10000 |
| `KRIPA_MAX_INFLIGHT` | 32; range 1–256 |
| `KRIPA_REQUEST_TIMEOUT` | 2s; range 10ms–10s |
| `LOG_LEVEL` | info |

When a token is configured, charts, Panchang, readiness and metrics require `Authorization: Bearer <token>`. Liveness and metadata are public. Non-loopback binding requires a token. Use a randomly generated token and a private network; use TLS at the reverse proxy when crossing hosts. Source publication is not authorization to expose this service publicly. Server limits are 2s header read, 5s read, 15s write, 60s idle and 8 KiB headers. Overload returns 503 with `Retry-After: 1`; calculation deadline returns 504. Native C calls cannot be interrupted midway.

Logs omit request bodies, birth inputs, coordinates and tokens. Keep secrets out of Git. Size-based Compose log rotation is 3 × 10 MB; this does not guarantee a retention time. Tune log volume toward approximately 24 hours in staging or 3–7 days in production.

## Calculation conventions

Both APIs accept Gregorian dates 1900–2099, IANA timezone names, decimal degrees north/east positive. Elevation is unsupported. Unknown fields and unsupported profiles are rejected. See [conventions](docs/CONVENTIONS.md) for precise interval semantics and limitations.

Charts use `western_tropical_v1`, apparent geocentric ecliptic positions, true North Node and opposite South Node. Degrees are within the named sign, not absolute longitude. Placidus houses fall back to whole-sign at unsupported latitudes, explicitly reported in metadata. No selectable house-system field exists. `unknown` birth time uses local noon and omits houses/angles; exact/approximate require a clock time. DST gaps and overlaps are rejected. Metadata quality labels describe input completeness, not independently established astronomical precision.

Panchang uses `lahiri_upper_limb_v1`: Lahiri sidereal, upper solar limb with refraction, sea-level observer, 1013.25 hPa and 15°C. Its day is `[sunrise, next_sunrise)`. Each limb array begins with the value at sunrise and includes subsequent transitions; the last `ends_at` may extend beyond next sunrise. Timestamps include UTC offsets. Root-search tolerance is 0.1 seconds and output is rounded to seconds; these are numerical tolerances, not accuracy guarantees. No supplied-instant option exists; clients can select the segment containing an instant inside the covered day.

## Tests and performance

```sh
make verify                    # formatting, vet, race + native tests
make bench                     # domain and warm-handler benchmarks
SWISS_EPHEMERIS_PATH="$PWD/.deps/swisseph/ephe" go run ./cmd/latency -n 200 -c 4
```

The target is p95 <100 ms on intended deployment hardware under a documented workload. Laptop and CI results do not establish the VM target. Independent Panchang validation and religious review remain incomplete. [Validation](docs/VALIDATION.md) records actual commands, architectures and measurement conditions.

## Containers

```sh
mkdir -p secrets
(umask 077; openssl rand -hex 32 > secrets/kripa_api_token)
KRIPA_VERSION=$(git rev-parse HEAD) docker compose build
docker compose up -d
```

Compose publishes only `127.0.0.1:8088`, mounts a token secret, runs non-root with a read-only filesystem, drops capabilities and limits memory to 256 MB. The multi-stage image contains the native executable, timezone data embedded by Go, three ephemeris files and licence notices. Linux AMD64 and ARM64 are targets; consult validation for actual tested architectures. No VM deployment is included. Start with one CPU and monitor queue pressure, memory and rejection rate before tuning concurrency.

## Source, licensing and contribution

Source: https://github.com/gopalmani/kripa. `/v1/meta` exposes source and build revision (`development` unless supplied at build). KRIPA is AGPL-3.0-or-later; see [LICENSE](LICENSE) and [NOTICE.md](NOTICE.md) for Swiss, Astrel and Go dependency provenance. The exact source/data commit and expected data hashes are pinned in the repository. No external Panchang implementation is a runtime dependency.

Future consumer attribution: “Chart calculations powered by KRIPA · Source code” (Astrel) and “Panchang powered by KRIPA · Source code” (BrahminBooking). API separation alone does not determine an application's AGPL obligations; no legal guarantee is made.

Contribute synthetic regression cases with source, conventions, expected values, tolerance and review status. Run `make verify`; retain copyright notices. Do not submit personal birth details, credentials or production records. See the plan for deferred consumer integration and calendar work.
