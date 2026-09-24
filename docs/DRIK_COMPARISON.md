# Drik Panchang comparison — 24 September 2026

KRIPA uses Swiss Ephemeris, not an independent replacement. Native tests, four
Astrel chart regressions and USNO event sanity checks pass. That does not prove
DrikPanchang.com parity or independent angular accuracy. A percentage score would
be misleading.

Drik's current [sunrise documentation](https://www.drikpanchang.com/panchang/sunrise/panchang-sunrise.html)
describes upper-limb sunrise with refraction, elevation disabled by default.
KRIPA matches that broad convention but fixes sea-level, pressure and temperature.
Drik offers other settings; its older FAQ describes different conventions.
Capture effective settings rather than treating any default as universal.

Date/location-specific daily-page retrieval failed through the browsing tool and
direct HTTP returned 403. A search result for a Bengaluru Bengali-calendar page
showed 21 September, but opening the same URL returned Washington on 23 September.
That mismatch disqualifies the snippet as a golden fixture. No access restriction
was bypassed or unverified time promoted into passing tests. Measured Drik
agreement remains unestablished. Drik is not a runtime dependency or data feed.

| Capability | KRIPA now | Gap |
| --- | --- | --- |
| Tropical charts | Planets, true node, Chiron, houses, major aspects | Broader independent angle/longitude fixtures; full Astrel integration tests |
| Vedic kundali | Absent | Sidereal charts, divisions, dashas, matching require separate scope |
| Core Panchang | Sunrise-day limbs, vara/paksha | Convention-aligned independent boundary fixtures and specialist review |
| Rise/set | Upper limb, fixed atmosphere, sea level | Optional elevation/conventions; terrain not modeled |
| Moonset | Requested civil day | Drik may display following-day event; compare the same event |
| Daytime periods | Rahu Kalam, Yamaganda, Gulika | Other muhurta systems absent |
| Religious calendar | Explicitly unsupported | Lunar months, Amanta/Purnimanta, adhika/kshaya, festivals, vrat/parana |

Before removing preview, collect permitted reference observations with explicit
date, coordinates, timezone, ayanamsha, rise convention, elevation and precision.
Cover six Indian regions across seasons, historical/future dates, sunrise/midnight
boundaries and skipped/repeated tithis. Tropical chart output is not directly
comparable to a Vedic sidereal chart without aligning conventions.

Measure signed event differences and limb identity matches. Start investigation
thresholds at 60 seconds for limb transitions and 120 seconds for rise/set,
accounting for minute-rounded references. These are review gates, not accuracy
claims. Do not widen tolerances to hide mismatches; investigate UTC/UT1/TT,
ephemeris flags, ayanamsha and rise settings. Record unresolved differences.
The reviewed reference collection remains pending; retain astronomical_preview.
