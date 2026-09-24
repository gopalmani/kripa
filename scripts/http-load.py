#!/usr/bin/env python3
"""Bounded synthetic private-API load. No birth records, token or bodies printed."""
import argparse
from concurrent.futures import ThreadPoolExecutor
from datetime import date, timedelta
import json
import math
import os
from pathlib import Path
import threading
import time
import urllib.request
import urllib.error

parser = argparse.ArgumentParser()
parser.add_argument("--url", default="http://127.0.0.1:8088")
parser.add_argument("--seconds", type=int, default=60, choices=range(1, 301))
parser.add_argument("--rps", type=int, default=5, choices=range(1, 21))
args = parser.parse_args()
token = os.environ.get("KRIPA_API_TOKEN", "")
if os.environ.get("KRIPA_API_TOKEN_FILE"):
    token = Path(os.environ["KRIPA_API_TOKEN_FILE"]).read_text().strip()
slots = threading.BoundedSemaphore(4)

def call(index):
    try:
        kind = ["charts", "panchang_uncached", "panchang_warm"][index % 3]
        payload = dict(date="2026-09-21", timezone="Asia/Kolkata", latitude=12.9716,
                       longitude=77.5946, profile="lahiri_upper_limb_v1")
        route = "/v1/panchang"
        if kind == "charts":
            route = "/v1/charts"
            payload.update(profile="western_tropical_v1", time="12:00", time_status="exact")
        elif kind == "panchang_uncached":
            payload["date"] = (date(2000, 1, 1) + timedelta(days=index)).isoformat()
        headers = {"Content-Type": "application/json"}
        if token:
            headers["Authorization"] = "Bearer " + token
        req = urllib.request.Request(args.url.rstrip("/") + route,
                                     data=json.dumps(payload).encode(), headers=headers)
        start = time.monotonic()
        status, cached = 0, None
        try:
            with urllib.request.urlopen(req, timeout=5) as response:
                response.read()
                status = response.status
                cached = response.headers.get("X-Kripa-Cache")
        except urllib.error.HTTPError as exc:
            status = exc.code
            exc.close()
        except (OSError, TimeoutError):
            pass
        return dict(kind=kind, status=status, cache=cached, ms=(time.monotonic()-start)*1000)
    finally:
        slots.release()

started = time.monotonic()
futures, missed = [], 0
with ThreadPoolExecutor(max_workers=4) as pool:
    for index in range(args.seconds * args.rps):
        time.sleep(max(0, started + index / args.rps - time.monotonic()))
        if not slots.acquire(blocking=False):
            missed += 1
            continue
        futures.append(pool.submit(call, index))
results = [future.result() for future in futures]
report = dict(seconds=args.seconds, rps=args.rps, maximum_inflight=4, requests=len(results),
              missed=missed, failures=sum(row["status"] != 200 for row in results), workloads={})
for kind in ["charts", "panchang_uncached", "panchang_warm"]:
    rows = [row for row in results if row["kind"] == kind]
    durations = sorted(row["ms"] for row in rows)
    report["workloads"][kind] = dict(requests=len(rows),
        cache_hits=sum(row["cache"] == "HIT" for row in rows),
        failures=sum(row["status"] != 200 for row in rows),
        **{f"p{percent}_ms": round(durations[math.ceil(len(durations)*percent/100)-1], 3)
           for percent in [50, 95, 99] if durations})
print(json.dumps(report, indent=2))
raise SystemExit(bool(report["failures"] or missed))
