# Validation record — 2026-09-22

## Repository and environment

Starting commit `f453ce7`, branch `main`, clean working tree. Origin verified as
`gopalmani/kripa`; existing commits were not rewritten. Prior Actions run
[35713384875](https://github.com/gopalmani/kripa/actions/runs/35713384875) passed,
including its Linux AMD64 container build. This is historical baseline evidence,
not evidence for the changes in this audit.

Local environment: Apple M5 Pro, 15 logical CPUs, 24 GiB RAM, macOS 26.5.2
(25F84), Darwin ARM64, Go 1.26.5, Apple clang 21.0.0. No Docker executable is
installed locally. No application-VM access/deployment was attempted.

## Commands and outcomes

- `sh scripts/setup-swiss.sh`: passed source checkout, native static library and swetest build, and all three committed data SHA-256 checks. Upstream swetest emitted five clang format-security warnings; the service does not invoke swetest per request or distribute it in the runtime image.
- `make verify`: passed committed gofmt check, `go vet ./...`, and race-enabled unit/native integration tests with `SWISS_EPHEMERIS_PATH` set. Tests are in normal Go packages, not a separate integration build tag.
- `SWISS_EPHEMERIS_PATH="$PWD/.deps/swisseph/ephe" go test -race -count=1 ./...`: fresh uncached run passed (see final delivery/CI evidence).
- Production configuration smoke: non-loopback without token and whitespace-only token both exited 1; authenticated schema smoke passed and unauthenticated readiness returned 401.
- `make build`: passed native Darwin ARM64 binary build (`CGO_ENABLED=1`, `-trimpath`, `-s -w`).
- `python -m openapi_spec_validator api/openapi.yaml`: passed, validator 0.7.2 in an isolated temporary venv with PyYAML 6.0.3.
- `python scripts/smoke.py --url http://127.0.0.1:18080 --schema`: passed live local HTTP and response-schema validation for both chart time modes, Panchang, health, metadata, metrics and representative errors. Local server exited 0 on SIGTERM. An initial connection probe retried while startup completed.
- `make bench`: passed three repetitions of uncached chart/Panchang domain and warm HTTP-handler benchmarks.
- `go run ./cmd/latency -n 200 -c 4` and `GOMAXPROCS=1 go run ./cmd/latency -n 100 -c 1`, with the ephemeris environment set: passed; results below/in JSON.
- Local container build: unavailable (`docker: command not found`). Current workflow adds native Linux AMD64 and Linux ARM64 build/test/container/schema-smoke/SIGTERM jobs. Their results must be read from the pushed commit, not inferred from local tests.

No production correctness or VM latency claim follows from these checks.

## Accuracy evidence and gaps

Three distinct categories are retained:

1. **Mathematics/invariants:** analytic linear-orbit wraparound, multiple karana transitions, sequence mapping, interval continuity, daylight ordering, six Indian coordinates across four dates, equal-eighth period arithmetic, date and DST validation (including Lord Howe half-hour transitions). Linear orbits are test doubles only, never production fallback calculations.
2. **Upstream regressions:** J2000 Sun ~280.36892° (0.01° broad tolerance) agrees with the pinned `swetest -b1.1.2000 -utc12:00:00 -p01 -fPl -head`. This is the same engine and is not independent. Actual Astrel SwissEngine output is captured in `fixtures/astrel-charts.json` for four synthetic inputs (UTC/India, exact/approximate/unknown, polar whole-sign fallback). The source SHA, binding, coordinates, times, timezone, results and 0.02° tolerance are recorded. Tolerance allows the verified UT1-vs-UTC conversion difference; exact numerical parity is not promised. Assertions cover body/sign/retrograde/house/aspect structure and angles. No Astrel production records were used.
3. **Independent event sanity:** `fixtures/usno-phases.json` records retrieved USNO new/full moon times, UTC, coordinates/timezone used to select the KRIPA Panchang interval, expected tithi boundary and 300-second tolerance. Source: [USNO primary phases API](https://aa.usno.navy.mil/api/moon/phases/date?date=2024-04-08&nump=4). This validates only broad phase timing, not sunrise, nakshatra/yoga, festivals or religious rules. A five-minute tolerance is intentionally not an accuracy certification.

An additional independent sanity fixture, `fixtures/usno-sun.json`, checks
Bengaluru 2024-06-21 sunrise 05:55 and sunset 18:48 IST against the USNO API
with a 120-second tolerance. [USNO methodology](https://aa.usno.navy.mil/faq/RST_defs)
uses a fixed 90.8333° solar-centre zenith distance; Swiss uses its upper-limb
and atmospheric model. The fixture records this difference and minute rounding;
it is not proof of exact convention parity or two-minute physical accuracy.

Independent reference coverage remains inadequate for a production calendar.
More historical/future, near-sunrise/midnight, high-latitude and convention-aligned
sunrise/limb cases need review. Geographical invariant coverage is not equivalent
to independently validated India-wide Panchang accuracy. Numerical root tolerance
is not physical accuracy. See [CONVENTIONS.md](CONVENTIONS.md).

## Astrel and Panchang reference review

The provided `astro-backend-main 2.zip` and extracted Downloads Swiss source match
SHA-256 `e4ddde649e740ffbbf3fccda2b2e8d0cb075065b0b61de3894cd2738bee5aedd`.
Reviewed calculator interface, input/output structs, Swiss adapter, mathematics
and Swiss/engine tests. Western tropical, true nodes, body list, aspects and house
fallback are confirmed. An isolated temporary copy of only calculator files was
run with gose v0.0.1 linked to the pinned Swiss library; Astrel itself was not
modified. KRIPA intentionally has stricter validation, rejects DST ambiguity,
and uses Swiss UTC-to-UT1 conversion whereas Astrel passes clock UTC as UT1.
Full consumer compatibility remains a later integration gate.

Reference-only source inspection, no new copied/translated code:

- [webresh/drik-panchanga](https://github.com/webresh/drik-panchanga/tree/325c7f37c01131d1a159d9db1cc3a2e70822ee13): inherited AGPL-3.0-or-later source header; disc-centre rise/set convention; fixed numeric timezone offset in core; karana returns only an index despite its end-time docstring. The example block contains a few assertions and many printed comparisons; no complete boundary/reference suite was established.
- [dhoomakethu/panchanga-cli](https://github.com/dhoomakethu/panchanga-cli/tree/7173deebe7ed00a5cc0d4836c2cb92a823c4a4df): related implementation, AGPL licence/inherited notices; same disc-centre/karana behavior; timezone offset calculated at civil midnight with `is_dst=True`, potentially different from a later event across a DST transition; legacy requirements include pyswisseph 2.0.0.post2 and pytz 2017.2. No complete festival coverage or defensible numerical accuracy tolerance was established from this review.

Their agreement would not be independent validation. KRIPA's upper-limb sunrise
is different and no DrikPanchang.com scraping/runtime dependency was introduced.

## Performance interpretation

Raw reports: `performance-darwin-arm64.json`, `performance-darwin-arm64-single.json`.
Both use Swiss 2.10.03, source commit/data version from metadata, native flags
`-O2 -Wall -fPIC`, Go run default build flags and CGO enabled. No race instrumentation
or request logging. Server admission 64, 10s calculation deadline, HTTP timeout 15s.

Each case has one first request separately measured, then 200 requests/c=4 or
100 requests/c=1. Uncached inputs vary civil date from 2000-01-01 over the request
count (wrap after 3650); location is Bengaluru 12.9716N, 77.5946E, Asia/Kolkata.
Charts use local noon exact; warm Panchang repeats 2026-09-21 with 512-entry cache;
uncached/mixed cases have zero-entry cache. Mixed alternates charts and Panchang.
The first request is cold only with respect to that service's application cache;
OS filesystem/native startup state is already warm, not a cold boot.

Reported p50/p95/p99 include every timed HTTP attempt, including failures if any;
first request is excluded. Throughput is requests divided by measured wall time.
Raw server metrics include the first request and separate handler histogram/sum,
native session count/duration, gate wait, cache and rejection counts. Native-session
timing excludes path configuration/cleanup; domain benchmarks include it. Short
microbenchmarks are diagnostic, not sustained-load or saturation certification.
GOMAXPROCS=1 does not emulate a VM CPU quota or memory budget.

Domain benchmarks (three repetitions, one fixed synthetic input): chart
0.151–0.152 ms, uncached Panchang 0.810–0.816 ms; warm handler 0.0474–0.0477 ms.
Chart input: J2000 noon UTC at 51.5N/0E. Panchang: 2026-09-21 Bengaluru. The
benchmark named `PanchangCold` means no application cache, not cold OS/native startup.

| Workload (c=4, n=200) | p50 ms | p95 ms | p99 ms | req/s | duration s | errors / rejections |
| --- | --- | --- | --- | --- | --- | --- |
| charts_uncached | 0.993 | 1.468 | 1.767 | 3765 | 0.053 | 0 / 0 |
| panchang_uncached | 3.236 | 3.690 | 3.945 | 1217 | 0.164 | 0 / 0 |
| panchang_warm | 0.166 | 0.334 | 0.750 | 20529 | 0.010 | 0 / 0 |
| mixed_uncached | 2.094 | 2.296 | 2.491 | 1881 | 0.106 | 0 / 0 |

## Reproduction and later acceptance commands

Local start and smoke from repository root:

```sh
make setup && make build
SWISS_EPHEMERIS_PATH="$PWD/.deps/swisseph/ephe" ./bin/kripa
# In another terminal:
python3 scripts/smoke.py
curl --fail-with-body http://127.0.0.1:8080/health/live
curl --fail-with-body http://127.0.0.1:8080/v1/meta
```

For optional schema checks, create a venv, install
`openapi-spec-validator==0.7.2 PyYAML==6.0.3`, then run
`python -m openapi_spec_validator api/openapi.yaml` and
`python scripts/smoke.py --schema`. Protected deployment smoke reads
`KRIPA_API_TOKEN` or `KRIPA_API_TOKEN_FILE`; it does not print the token.

**Next ARM64 VM validation step (not executed in this task):** after authorization,
use an isolated checkout of the delivered revision on the intended machine, not
an existing production service. Record `uname -a`, `lscpu`, `free -h`, `go version`,
compiler version and CPU quota. Run:

```sh
make setup && make verify && make build
SWISS_EPHEMERIS_PATH="$PWD/.deps/swisseph/ephe" GOMAXPROCS=1 go run ./cmd/latency -n 10000 -c 1 > latency-vm-c1.json
SWISS_EPHEMERIS_PATH="$PWD/.deps/swisseph/ephe" GOMAXPROCS=1 go run ./cmd/latency -n 10000 -c 4 > latency-vm-c4.json
```

Repeat with the chosen sustained workload and resource limits; preserve all
errors/rejections, queue time, input diversity, cache state, duration and p95.
Then build/smoke the ARM64 container there, with private binding and a temporary
service token, before any deployment decision. A CI ARM64 runner is not the VM.

**Later Astrel integration:** in a separate task, implement its `AstrologyEngine`
client adapter mapping `BirthInput` to `/v1/charts` (profile western_tropical_v1),
configure private URL/service token, map errors and version metadata, and run the
full Astrel regression/product suite, including DST migration behavior. Broaden
the four regression fixtures before switching real traffic. Preserve Astrel's
licence and add “Chart calculations powered by KRIPA · Source code”.

**Later BrahminBooking integration:** in a separate task, add a backend client for
`/v1/panchang` with the user's local date, IANA timezone, exact coordinates and
lahiri_upper_limb_v1 profile. Display sunrise-day intervals and preview status;
do not infer festivals/vrats or religious recommendations. Test midnight/next-day
transitions and add “Panchang powered by KRIPA · Source code”. No Firebase,
Razorpay, booking or provider workflows belong in KRIPA.
