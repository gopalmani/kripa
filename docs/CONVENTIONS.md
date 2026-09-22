# Calculation contract and limitations

## Shared inputs and native source

Dates are Gregorian, 1900-01-01 through 2099-12-31 inclusive. Coordinates are finite decimal degrees, north/east positive, latitude [-90,90], longitude [-180,180]. Both coordinates are required; zero is valid. All profiles require an explicit IANA timezone (UTC allowed; Go's host-dependent `Local` rejected). No geocoding or elevation option is present. JSON is one object, no unknown/duplicate/null fields, at most 8192 bytes. Missing fields return 400; invalid values return 422.

Native source: Swiss v2.10.3a, commit `56351c57e33916651a1aa32cfc6225e05c0ad865`, runtime `2.10.03`, built with `-O2 -Wall -fPIC`. Data set `swisseph-2.10.3a-1800-2399` consists of sepl_18, semo_18 and seas_18; API range is narrower. Setup checks the committed SHA-256 manifest. Swiss calculation return flags must include SWIEPH, otherwise requests fail. No Moshier fallback is served.

UTC is converted by `swe_utc_to_jd` to UT1 for calculations; output uses `swe_jdut1_to_utc`. Historical/future delta-T and future leap seconds are model-dependent. Do not infer subsecond physical accuracy from floating-point fields or the numerical root tolerance. No independently established angular precision guarantee is published yet.

## Tropical charts

`western_tropical_v1` / calculation version `kripa-tropical-v1` uses apparent geocentric ecliptic tropical longitude with speed. Bodies: Sun, Moon, Mercury, Venus, Mars, Jupiter, Saturn, Uranus, Neptune, Pluto, true North Node and Chiron. South Node is opposite the true North Node and inherits its retrograde flag. Retrograde is negative longitudinal speed. No mean-node option exists.

`time_status` is exact, approximate or unknown. Exact/approximate accept HH:MM or HH:MM:SS. Unknown requires omitted or empty time and uses local noon for planetary positions, returns empty houses and null angles/planet houses. Noon is a convention, not an estimated birth time. Local gaps and overlaps are rejected, including non-hour DST changes. The caller must obtain an unambiguous input; no fold selector exists.

Timed charts use Placidus; if Swiss reports unavailable houses, KRIPA retries whole-sign and reports `house_system: whole_sign`. This existing v1/Astrel behavior is preserved and explicitly exposed, not a silent substitution. House system cannot be selected through v1. Degrees for planets/houses/angles are within the named sign [0,30). `high`/`standard` quality describes time-input completeness, not measurement accuracy.

Aspects use the smallest angular separation: conjunction (0°, orb ≤8°), sextile (60°, ≤6°), square (90°, ≤7°), trine (120°, ≤8°), opposition (180°, ≤8°). No interpretations or scoring are returned.

## Astronomical Panchang preview

`lahiri_upper_limb_v1` / `kripa-panchang-lahiri-v1` uses apparent geocentric Lahiri sidereal positions. Tithi is the Moon−Sun elongation in 12° divisions (1–30), karana in 6° divisions (1–60), nakshatra the Moon longitude in 360/27° divisions, yoga the normalized Moon+Sun longitude in 360/27° divisions. Phase subdivisions are numbered cyclically and mapped to names in `internal/panchang`.

Sunrise/sunset use Swiss rise_trans: upper limb, refraction enabled, observer height zero, pressure 1013.25 hPa, temperature 15°C. The Moon uses the same upper-limb/atmospheric settings. This is a standard-atmosphere model; local horizon, terrain, weather and observer elevation are not modelled. A missing usable sunrise/sunset returns 422, never a fabricated event. Coordinates across India are accepted; coverage tests are invariants, not reference certification.

The exact response coverage is `[sunrise, next_sunrise)`. Every limb array begins at sunrise (not at the actual preceding transition), includes all subsequent transitions before next sunrise, and retains the final segment's actual end even if it is outside coverage. Segments are half-open `[active_from, ends_at)`. Clip the final segment to `next_sunrise` when drawing the daily interval. At an exact boundary select the next segment. Rounded timestamps can hide subsecond differences.

Transitions are bracketed over two days, then bisected to ≤0.1 seconds; timestamps round to the nearest second and include the requested location's UTC offset. Historical offsets containing seconds are emitted in UTC (`Z`), because RFC 3339 cannot encode subminute offsets; `timezone` still identifies the requested zone and vara uses the local sunrise weekday. Multiple events and transitions past midnight are retained. There is no instant input: within coverage, select the segment containing the desired instant. `paksha_at_sunrise` refers only to the first tithi: indices 1–15 Shukla, 16–30 Krishna.

`vaar` is the weekday of the local sunrise, used for this sunrise-to-sunrise interval. The civil date changes at midnight independently. Moonrise/moonset instead cover the local civil day `[midnight, next midnight)`; null means no event in that interval. Timezones changing their midnight/date boundary may reject a nonexistent date; not all exotic historical civil-time changes are validated.

Daytime periods divide the rounded sunrise-to-sunset duration into eight equal portions. Sunday-first portion indices are:

| Period | Sun | Mon | Tue | Wed | Thu | Fri | Sat |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Rahu Kalam | 8 | 2 | 7 | 5 | 6 | 4 | 3 |
| Yamaganda | 5 | 4 | 3 | 2 | 1 | 7 | 6 |
| Gulika | 7 | 6 | 5 | 4 | 3 | 2 | 1 |

This is the implemented daytime Gulika convention; nighttime periods and religious suitability recommendations are unsupported.

## Native concurrency and cache lifecycle

The exact source defines TLS on Linux GCC builds and disables it on Apple builds (`sweodef.h`). A process-wide channel serializes the entire configuration/calculation session and honors context cancellation while waiting. `runtime.LockOSThread` keeps all calls on the same native thread. Each session sets the ephemeris path and Lahiri mode; tropical calls omit the sidereal flag. `swe_close` runs before unlocking the OS thread, avoiding native resources stranded on Go worker threads. `SE_EPHE_PATH` overrides upstream's explicit path, so startup rejects it. Paths exceeding the native buffer limit are rejected.

Readiness performs current native calculations for Sun, Moon and Chiron at J2000, covering the three required data files. Startup checks all chart bodies. Neither proves every date or every future data mutation is valid; production data should be read-only. Go's race detector cannot prove C race freedom; repeated uncached mixed calls test deterministic isolation.

Panchang keys contain every exact input (no coordinate rounding), calculation version, runtime engine version, pinned engine commit and data version. Entries expire 24 hours after completion, move to MRU on hits and evict LRU beyond capacity. Zero capacity disables storage while retaining duplicate coalescing. Flights are bounded by HTTP admission. Followers may cancel independently; the initiating request's deadline governs shared work, so a failed leader may fail its followers. Errors are never cached. Cache-hit metrics include coalesced followers. Charts have no cache or duplicate coalescing.

## Explicitly unsupported

Regional festivals, vrat rules, personalized muhurat, religious recommendations, lunar month names, Amanta/Purnimanta, adhika/kshaya rules. No claim of Drik-level accuracy, complete India-wide observance coverage or production correctness is made.

## v1 validation migration from f453ce7

Routes and successful result schemas are unchanged. The audit corrects previously
permissive decoding: omitted coordinates no longer imply 0°, null fields and
case-variant/duplicate keys no longer overwrite valid fields, and `Local` no
longer depends on the host timezone. Such clients must send both numeric
coordinates explicitly (including zero), use the documented lowercase keys once,
omit optional unknown time or send an empty string, and supply an IANA zone/UTC.
Missing/malformed fields return 400; invalid domain values return 422. This is an
explicit migration for input-correctness fixes, not a zodiac/profile change.
Environment tokens must contain at least 32 characters without ASCII whitespace;
secret-file surrounding whitespace is trimmed as before. Unset native
`SE_EPHE_PATH` and configure only `SWISS_EPHEMERIS_PATH`.

Historical timestamp migration: 1900-era subminute timezone offsets previously
lost their seconds component during JSON serialization. Those event timestamps
now use `Z` to preserve the instant; modern whole-minute offsets are unchanged.
