# Corresponding source

GitHub releases include a source bundle for the exact deployed revision, the
unmodified pinned Swiss source/data, locked Go module sources, licences, and
build/deployment scripts. No host configuration or secrets are included.

Build a source bundle with `make setup && sh scripts/source-bundle.sh` from a
clean committed checkout. To build the unpacked bundle without dependency
downloads (Go 1.26, C compiler, make and libc development headers required):

```sh
make -C .deps/swisseph CFLAGS='-O2 -Wall -fPIC' libswe.a
CGO_ENABLED=1 GOPROXY=off go build -mod=vendor -trimpath -o bin/kripa ./cmd/kripa
SWISS_EPHEMERIS_PATH="$PWD/.deps/swisseph/ephe" ./bin/kripa
```

The Dockerfile installs its OS/toolchain dependencies from upstream repositories;
this is not a bit-for-bit reproducible OS image. KRIPA remains AGPL-3.0-or-later,
not a replacement for Swiss Ephemeris or an exemption from its licence. See
NOTICE.md. An API boundary alone does not establish that future consumers have
no AGPL obligations; obtain legal advice before relying on that interpretation.

Later consumer attribution: “Chart calculations powered by KRIPA · Source code”
and “Panchang powered by KRIPA · Source code”, linking to this repository and the
deployed revision. Source and licence notices must remain available; attribution
alone is not licence compliance. Consumers are intentionally unchanged here.
