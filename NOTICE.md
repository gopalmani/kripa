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
