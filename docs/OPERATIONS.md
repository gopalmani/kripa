# Private deployment

Operate KRIPA alone; do not change consumer backends, proxy, databases or collector.
Actual host inventory and credentials stay outside this public repository.

## Install and rollback

1. Select a commit with all Verify KRIPA jobs green. Download its
   `kripa-arm64-image` artifact (never a pull-request artifact).
2. On the ARM64 VM verify `sha256sum -c kripa-arm64.tar.gz.sha256`, then
   `docker load -i kripa-arm64.tar.gz`. Retain prior images for rollback.
3. Copy that revision's deploy directory to a dedicated operator-owned directory.
   Run `python3 install.py --revision FULL_COMMIT_SHA` there.
4. Run `KRIPA_API_TOKEN_FILE=/absolute/deploy/secrets/kripa_api_token python3
   scripts/smoke.py --url http://127.0.0.1:8088` from the matching source checkout.
5. Verify unauthenticated readiness/calculations/metrics return 401, authenticated
   readiness succeeds, Prometheus target is UP, and existing services are healthy.

Only loopback ports 8088 (API) and 9098 (Prometheus) are published. The dedicated
Docker bridge isolates KRIPA from other service networks. Outbound access is
possible; this is inbound-private, not an egress-denied sandbox. Docker 29 does
not publish loopback ports for internal-only networks. Later approved consumers can
join it and use `http://kripa:8080` with a token. Localhost inside a consumer
container does not refer to KRIPA. No public proxy or consumer change is included.

The installer creates a random token only if absent. The secrets directory is
0700; its file is 0444 so two non-root container UIDs can read individual bind
mounts. Other host users cannot traverse the directory; Docker administrators
remain privileged. Never print or commit the token. Rotation requires securely
replacing the file and recreating both services.

Rollback: reinstall the prior tested image SHA with its matching deployment files.
First-deployment rollback: `KRIPA_IMAGE=kripa:FULL_SHA docker compose stop` within
this deployment directory. Preserve metrics volume and secrets. Failed smoke or
readiness is a no-go for integration. No schema migrations exist.

## Monitoring

Forward `ssh -N -L 9098:127.0.0.1:9098 YOUR_DEPLOY_HOST`, then open
http://localhost:9098/query (graphs), `/targets` (scrape health), `/alerts`.
This is Prometheus's built-in graph UI, not a separate Grafana installation.

```promql
sum by (route) (increase(kripa_requests_total[24h]))
histogram_quantile(0.95, sum by (route, le) (rate(kripa_request_duration_seconds_bucket[5m])))
sum by (route) (rate(kripa_errors_total[5m]))
rate(kripa_native_queue_wait_seconds_total[5m])
kripa_inflight
increase(kripa_auth_rejections_total[1h])
```

Scrape interval 30s; persistent metrics retention seven days or 256 MB of blocks
(WAL/head overhead is additional). Counters reset at restart; use increase/rate.
Request counters exclude auth denials (separate counter), health and metrics.
Native timings include readiness probes. Histograms approximate percentiles.
Alerts cover target down, native failure, overload and auth denial. Alerts appear
in the UI; email/Slack/pager notifications are not configured.

`docker compose logs --since 1h kripa` with KRIPA_IMAGE set shows info-level JSON:
generated request ID, fixed route, method, status, duration and cache hit only.
No birth inputs, coordinates, query strings, tokens or client IPs are logged.
API logs rotate at 3 x 10 MB; Prometheus at 2 x 5 MB, not time-based retention.
No external telemetry export, tracing or user analytics is enabled.

API caps: .25 CPU, 192 MB, GOMAXPROCS=1, 8 inflight, 2s calculation deadline.
Prometheus caps: .10 CPU, 192 MB, two concurrent queries. These caps are not a
reservation or latency guarantee. Native C work serializes and cannot be killed
mid-call. Base-image tags/OS packages are mutable; rebuilds are not byte-identical.
Retain release source bundles and image hashes and regularly scan/rebuild images.

Reproduce a bounded VM-side mixed check using `KRIPA_API_TOKEN_FILE=... python3
scripts/http-load.py --seconds 60 --rps 5`. It alternates uncached charts,
varying-date Panchang and repeated Panchang, caps four inflight requests and
includes all timed attempts in percentiles. Cold means application cache miss,
not cold OS/native startup; rerunning the same dates can warm that case, so
inspect reported cache hits. Report failures/missed scheduling, not just latency.
