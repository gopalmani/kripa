FROM golang:1.26-bookworm AS build
RUN apt-get update && apt-get install -y --no-install-recommends git build-essential ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY scripts/setup-swiss.sh scripts/ephemeris.sha256 scripts/
RUN sh scripts/setup-swiss.sh
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=development
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/kripa ./cmd/kripa

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/kripa /usr/local/bin/kripa
COPY --from=build /src/.deps/swisseph/ephe/sepl_18.se1 /opt/kripa/ephe/sepl_18.se1
COPY --from=build /src/.deps/swisseph/ephe/semo_18.se1 /opt/kripa/ephe/semo_18.se1
COPY --from=build /src/.deps/swisseph/ephe/seas_18.se1 /opt/kripa/ephe/seas_18.se1
COPY --from=build /src/.deps/swisseph/LICENSE /usr/share/licenses/swisseph/LICENSE
COPY --from=build /src/.deps/ephemeris.sha256 /opt/kripa/ephe/SHA256SUMS
COPY LICENSE NOTICE.md /usr/share/licenses/kripa/
COPY docs/licenses/ /usr/share/licenses/kripa/dependencies/
ENV SWISS_EPHEMERIS_PATH=/opt/kripa/ephe KRIPA_LISTEN_ADDR=0.0.0.0:8080
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s CMD ["/usr/local/bin/kripa", "-healthcheck"]
ENTRYPOINT ["/usr/local/bin/kripa"]
