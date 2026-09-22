# Standalone KRIPA audit plan

Baseline: `main` at `f453ce7` (2026-09-22), clean tree, origin
`https://github.com/gopalmani/kripa.git`. Existing history is retained. This audit
changes KRIPA only; no VM deployment or consumer/infrastructure changes.

## Implementation and validation states

| Checkpoint | Implemented | Tested locally | Independent validation | CI | Container / architecture |
| --- | --- | --- | --- | --- | --- |
| Tropical chart / Astrel extraction | Yes | Native tests; four actual-Astrel regression cases | Pending broader angular references | Passed on both Linux architectures (c7019f2) | Darwin ARM64 local; Linux AMD64/ARM64 CI containers passed |
| Swiss source/data pins and session isolation | Yes; checksum enforcement, override rejection, cleanup | Missing data/fallback; repeated uncached mixed requests; race tests | Source inspected, C race freedom not proved | Passed on both Linux architectures (c7019f2) | Darwin ARM64 tested |
| Core Panchang and numerical transitions | Yes, astronomical_preview | Six Indian locations; wraparound/multiple-transition invariants; period math | USNO new/full moon and one sunrise/sunset sanity case | Passed on both Linux architectures (c7019f2) | Darwin ARM64 tested |
| HTTP auth, deadlines, overload, strict decoding | Yes; malformed coordinate inputs now rejected | API tests, schema smoke, local SIGTERM | Not applicable | Passed on both Linux architectures (c7019f2) | Linux AMD64/ARM64 container smoke passed |
| Bounded cache and duplicate suppression | Yes; all version inputs in key | LRU/TTL/error/coalescing/cancellation/panic tests | Not applicable | Passed on both Linux architectures (c7019f2) | No chart cache |
| Observability | Request IDs; handler histogram; native duration/queue/failures; cache and admission counters | Metrics HTTP smoke and latency output | Not applicable | Passed on both Linux architectures (c7019f2) | No remote service required |
| README, conventions, OpenAPI, notices | Yes | OpenAPI validator and actual response schema checks | Licence/source inspection; no legal guarantee | Passed in both CI jobs | Notices copied into image |
| Performance measurement | Expanded harness | Darwin ARM64 c=4 and GOMAXPROCS=1/c=1 | Intended VM acceptance pending | Both Linux architectures passed | No VM deployment |

## Completed audit checkpoints

- [x] Fetch and verify repository/history/authentication; read implementation and all baseline tests/build files/docs.
- [x] Establish fresh green baseline (native build, vet, race-enabled tests).
- [x] Rewrite stale README before implementation edits.
- [x] Audit actual supplied Astrel source, reproduce synthetic upstream fixtures without modifying Astrel.
- [x] Inspect both Panchang references and licences; record differences; copy no reference code.
- [x] Harden native lifecycle, readiness, HTTP request shape, request IDs and metrics.
- [x] Pin expected data hashes; preserve exact third-party notices in image.
- [x] Add OpenAPI and executable smoke checks with optional response-schema validation.
- [x] Run local tests, native build, HTTP smoke, graceful SIGTERM and latency harness.
- [x] Commit and push normal successors to main; verify resulting Actions runs.
- [x] Record Linux AMD64 and ARM64 container outcomes: both passed for c7019f2, run 35718596027.

## Remaining release gates (planned)

- [ ] Independent position/angle and sunrise references across historical/future dates and boundary-heavy India-wide cases; explain all convention differences.
- [ ] Calendar-specialist review of core limbs and daytime conventions; independently sourced near-sunrise and near-midnight transition cases.
- [ ] Intended ARM64 VM acceptance at documented CPU/RAM, sustained duration, realistic input diversity and target concurrency. A short laptop run is not acceptance.
- [ ] Bit-for-bit container reproducibility: pin base-image digests/apt snapshots and toolchain artifacts for a release. Current native source/data and Go modules are pinned; OS tags remain mutable.
- [ ] Later Astrel adapter integration with full product-level regression suite; separate authorization/task.
- [ ] Later BrahminBooking daily-Panchang client integration retaining preview semantics; separate task.

## Unsupported (not completion blockers for this astronomical preview)

Regional festivals, vrats, personalized muhurat/recommendations, lunar-month
naming, Amanta/Purnimanta and adhika/kshaya rules. They require dedicated reviewed
rule sets; no placeholders are emitted. No database, customer authentication,
payments, bookings, Python service or per-request subprocess is planned.

Historical timestamp follow-up: a regression test reproduced subminute IANA
offset loss in RFC 3339 JSON. UTC fallback preserves those instants; local
formatting/vet/race checks pass. Every follow-up push runs the same CI matrix.

See [VALIDATION.md](VALIDATION.md) for exact evidence, references, measurements,
and next-step commands. Checkmarks record work performed, not a certification of
production or religious-calendar correctness.
