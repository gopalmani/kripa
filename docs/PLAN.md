# Standalone implementation plan

Approved direction: Go backend; zerolog; shared birth-chart and Panchang engine; public source; private deployments supported. Astrel and BrahminBooking integration is explicitly deferred.

## Choices

- Go standard `net/http` server and JSON codec. No framework is required for five routes; avoid replacing standard interfaces without profiling evidence.
- Direct cgo calls into pinned Swiss Ephemeris C code. In-process native calls avoid subprocess and Python service hops.
- Serialize native sessions process-wide and pin their OS thread. Set path and Lahiri mode for every session. Scale CPU-heavy misses with separate processes only after measuring queue pressure.
- Bounded LRU for daily Panchang JSON, with duplicate-request coalescing. No chart cache or persistent birth data. No SQL database, Redis, queue broker, or Kubernetes needed by KRIPA.
- Private container network or loopback; service-token authentication outside loopback. Application login and public request abuse protection belong in consuming products later.
- Immutable profile names, explicit unsupported fields, timestamped transitions, and reference tests.

## Checkpoints

1. Extract chart math and create a narrow, versioned request contract.
2. Build native adapter and startup checks; reject missing Swiss data instead of accepting Moshier fallback.
3. Implement core Panchang and event root searches.
4. Add bounded HTTP admission, deadlines, cache, metrics, logging and shutdown.
5. Verify with native tests, concurrent mixed traffic, race detector and benchmarks.
6. Document source, build, deployment, capabilities and measured limitations; commit everything.

## Latency acceptance

Target p95 <100 ms for successful chart and daily Panchang requests on the intended VM at an explicitly reported load. Report p50/p95/p99, concurrency, CPU allocation, cache mode, input diversity and error counts. A cached result is not evidence of uncached speed. Local loopback excludes Internet and reverse-proxy latency. Measure again on the ARM app VM before claiming the production SLO.

## Non-goals for this checkpoint

No consumer repository changes, live deployment, Firebase migration, payment integration, database migration, unreviewed festival calendar, lunar month naming, personalized muhurta, or public hosted calculation endpoint.
