# Notices and source provenance

KRIPA is distributed under GNU AGPL version 3 or, at your option, any later version. See LICENSE.

## Chart extraction

The chart result structures, zodiac/house/aspect mathematics, body selection and chart assembly were adapted from the user's supplied `astro-backend-main 2.zip`, package `internal/astrology`. That source declares:

> Copyright (c) 2024-2026 AstroMatch contributors

Its license permits redistribution and modification under AGPL-3.0-or-later. KRIPA changes the native binding, input contract, validation, time conversion, concurrency handling and API boundary. No account, database, dating, payment or user-record code was copied. Existing Astrel repository licensing is unchanged.

## Swiss Ephemeris

Swiss Ephemeris is copyright Astrodienst AG. Authors: Dieter Koch and Alois Treindl. KRIPA uses the AGPL route and preserves the upstream license and notices in the fetched source and container. No endorsement is implied.

- Upstream: https://github.com/aloistr/swisseph
- Tag: v2.10.3a
- Exact source/data commit: `56351c57e33916651a1aa32cfc6225e05c0ad865`
- Data: `ephe/sepl_18.se1`, `ephe/semo_18.se1`, `ephe/seas_18.se1`
- Fetch/build: `scripts/setup-swiss.sh`
- The generated `.deps/ephemeris.sha256` records the installed data checksums.

The C interface is called directly through a small KRIPA cgo adapter. No gose or pyswisseph wrapper is included. Keep the fetched upstream license and notices with redistributions of its source, library or data.

## Panchang implementation

Core Panchang calculations and root searches in this repository were written for KRIPA. No code was copied from `webresh/drik-panchanga` or `dhoomakethu/panchanga-cli`. These remain useful evaluation references; KRIPA is not affiliated with DrikPanchang.com. Additional code reuse must include a provenance and license review.

## Go dependencies

The only direct Go module dependency is `github.com/rs/zerolog` (MIT). `go.mod` and `go.sum` identify its exact transitive dependency graph. Preserve dependency notices when redistributing their source. `go mod vendor` can produce a corresponding-source dependency snapshot for release distribution.

## Audit additions (2026-09-22)

The supplied Astrel archive's `internal/astrology/swiss.go` has SHA-256
`e4ddde649e740ffbbf3fccda2b2e8d0cb075065b0b61de3894cd2738bee5aedd`.
The extracted Downloads copy matches it byte-for-byte. Its actual tropical
`SwissEngine` was run in an isolated temporary module with `gose v0.0.1` to
produce synthetic regression fixtures; gose is **not** a KRIPA build/runtime
dependency. Fixture metadata records provenance and tolerances. The fixture
results are derived from the AGPL Astrel implementation; retain its attribution
above. No Astrel source or licence was modified.

The pinned Swiss source is unmodified. Setup now rejects modified tracked source
and verifies the three data files against `scripts/ephemeris.sha256` rather than
merely generating hashes after download. The source commit is the data provenance;
the version string is a KRIPA dataset label, not a claim of a separately released
data package. Swiss source, static library and ephemeris data are distributed
under the upstream AGPL option. The exact upstream notice is preserved in
`docs/licenses/swisseph.txt` and the container. Upstream source/license:
https://github.com/aloistr/swisseph/blob/56351c57e33916651a1aa32cfc6225e05c0ad865/LICENSE
and distribution guidance: https://www.astro.com/swisseph/swedownload_e.htm .

Exact Go dependency inventory (unmodified):

| Module | Version | Licence | Preserved notice |
| --- | --- | --- | --- |
| github.com/rs/zerolog | v1.35.1 | MIT | docs/licenses/zerolog-MIT.txt |
| github.com/mattn/go-colorable | v0.1.14 | MIT | docs/licenses/go-colorable-MIT.txt |
| github.com/mattn/go-isatty | v0.0.20 | MIT | docs/licenses/go-isatty-MIT.txt |
| golang.org/x/sys | v0.35.0 | BSD-3-Clause | docs/licenses/x-sys-BSD.txt |

Source locations are the module URLs above; versions and integrity hashes are in
`go.mod`/`go.sum`. All four notices are included in the runtime image. The Go
standard library/runtime is BSD-3-Clause; its licence is also preserved in
`docs/licenses/go-BSD.txt`. The Debian base image retains installed packages'
licences in `/usr/share/doc`. Base image tags and apt repository contents are
mutable; native source/data pins do not imply a bit-for-bit reproducible image.

Reference-only inspection (no copying or translation in this audit):
`webresh/drik-panchanga` at `325c7f37c01131d1a159d9db1cc3a2e70822ee13`
and `dhoomakethu/panchanga-cli` at `7173deebe7ed00a5cc0d4836c2cb92a823c4a4df`.
The former's source headers declare AGPL-3.0-or-later (Satish BD, 2013); the latter
includes AGPL-3.0 and inherited source notices. They are related implementations,
not independent astronomical references. See `docs/VALIDATION.md` for findings.

Release distributors must provide corresponding KRIPA source at the deployed
revision, these build/setup instructions, pinned native source/data and required
notices. `go mod vendor` can collect the locked Go sources for an offline release
bundle. Public `/v1/meta` and the HTTP source Link header identify KRIPA source;
operators must set the build revision. No claim is made that using an API alone
settles a consumer application's AGPL obligations.
