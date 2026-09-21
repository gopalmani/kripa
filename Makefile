.PHONY: setup fmt test bench build run verify
setup:
	sh scripts/setup-swiss.sh
fmt:
	gofmt -w cmd internal
build:
	CGO_ENABLED=1 go build -trimpath -ldflags='-s -w' -o bin/kripa ./cmd/kripa
run:
	go run ./cmd/kripa
test:
	SWISS_EPHEMERIS_PATH=$(CURDIR)/.deps/swisseph/ephe go test -race ./...
bench:
	SWISS_EPHEMERIS_PATH=$(CURDIR)/.deps/swisseph/ephe go test ./internal/api ./internal/chart ./internal/panchang -run '^$$' -bench . -benchmem -count=3
verify:
	@test -z "$$(gofmt -l cmd internal)"
	go vet ./...
	$(MAKE) test
