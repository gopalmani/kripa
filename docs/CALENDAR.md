# Lunar calendar, muhurta and observances

`POST /v1/panchang` returns `calendar` (`kripa-calendar-rules-v2`,
`review_status: rule_preview`) beside the astronomical sunrise day. The rules
below are implemented, tested against widely published 2026 dates for Delhi, and
not yet specialist-reviewed for every tradition or region.

## Calculated values

| Field | Rule |
| --- | --- |
| `lunar_month.amanta` | Sidereal (Lahiri) solar sign at the new moon that begins the month; Mesha → Vaishakha, Meena → Chaitra |
| `lunar_month.adhika` | No Sankranti between the bounding new moons |
| `lunar_month.purnimanta` | Amanta month in Shukla paksha; following month in Krishna paksha |
| `samvat` | Vikram = Gregorian year of the governing Chaitra + 57; Shaka = year − 78 (Chaitradi) |
| `sun_rashi`, `moon_rashi` | Sidereal sign at sunrise; Moon as sunrise-day segments |
| `sankranti` | Solar ingress instant between this sunrise and the next |
| `muhurta.abhijit` | 8th of 15 daytime muhurtas; `null` on Wednesday |
| `muhurta.brahma` | 14th of 15 night muhurtas before sunrise (previous sunset to sunrise) |

## Observances

Each observance carries `category` (`festival`, `vrat`, `jayanti`, `regional`,
`panchang_event`), `scope`, `tradition`, `regions` and the human-readable `rule`.
Rules match an Amanta month and a tithi prevailing at a decisive time: sunrise,
midday, pradosh (sunset to two night muhurtas) or nishita (8th night muhurta).
A night-window tithi that touches neither evening is observed on the day it
prevails at sunrise. Named Ekadashis follow the Purnimanta month, including
Padmini/Parama in an Adhika month; festivals are not reported in Adhika months.

## Known gaps

- Bhadra (Vishti) avoidance is not applied. Holika Dahan 2026 is reported on
  2 March (pradosh-vyapini Purnima); many calendars shift it to 3 March.
- Kshaya/vriddhi tithi tie-breaking between two qualifying days, Vaishnava
  Ekadashi, parana windows, Kshaya masa, Kartikadi samvat, regional solar
  calendars, Dur Muhurta, Amrit Kalam and Varjyam are not calculated.
- Regional applicability is descriptive; it is not a ruling for any sampradaya.

Do not tune constants to imitate a single website; record the convention first.


## Calendar rules v2: festival catalogue and years

`internal/panchang/catalog.go` holds about 90 festival rules, each with display
regions, ISO 3166-2 `region_codes` (`IN` for all of India) and `communities`.
Rule kinds: Amanta month and tithi at a decisive time (sunrise, purvahna end,
midday, aparahna, pradosh or nishita), the first day of a regional solar month,
a nakshatra or tithi within a regional solar month, the Friday before Shravana
Purnima (Varalakshmi Vratam) and the Bhadra rule for Holika Dahan.

Regional solar months begin by the Tamil rule (ingress before sunset: that
civil day, else the next), the Malayalam rule (before 3/5 of daytime), the
Bengali/Assamese rule (the civil day after the ingress) or the Sankranti-day
rule (as Tamil; Odia, Punjabi and pan-Indian Sankranti).

Day selection: when a tithi touches the decisive window on two consecutive
days, the day on which it covers more of the window wins, and on a tie the
first (rules marked `later` take the later day). A kshaya tithi that misses
both windows is kept on the day it begins (later-day rules: the next day); a
kshaya Pratipada belongs to the new month. Festivals are skipped in an Adhika
month except those marked `adhika` (Ganga Dussehra), which are then not
repeated in the Nija month. Holi is the day after Holika Dahan.

`POST /v1/festivals` (`{year, timezone, latitude, longitude, profile}`) scans
the year in monthly ephemeris sessions (about 0.2 s per year) and is cached.
`GET /v1/festivals/catalogue` lists the rules.

### Reference comparison

`docs/fixtures/festival-reference-delhi.json` holds 159 dates for 53 festivals
from published New Delhi calendars for 2025-2027 (dates only). The test
`TestFestivalReferenceDelhi` matches 156; the three documented differences are
Holika Dahan and Holi 2026 (Bhadra over the whole Purnima night on an eclipse
day; the classical rule gives 2 March, published calendars 3 March) and
Krishna Janmashtami 2027 (Rohini/udaya-Ashtami preference is not modelled).
Regional solar festivals are pinned by `TestRegionalSolarCalendars2026` but
have no independent regional reference yet.
