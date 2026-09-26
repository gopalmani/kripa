# Lunar calendar, muhurta and observances

`POST /v1/panchang` returns `calendar` (`kripa-calendar-rules-v1`,
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
