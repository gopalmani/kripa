# KRIPA

**One calculation engine. Birth charts, Panchang, and the astronomy beneath them.**

KRIPA is a planned open-source calculation library and HTTP service for birth charts and location-aware Panchang. It is designed to power Astrel and BrahminBooking through a shared, independently deployable engine built around Swiss Ephemeris.

> **Status: project specification.** This README describes the intended implementation. No runnable engine, validated Panchang release, or deployment is included yet. API paths below are proposals, not available endpoints.

## Why KRIPA

Astronomical calculations should be reproducible, explicit about their conventions, and easy to improve without changing the products that use them. KRIPA will keep calculation logic in one place while applications retain their own accounts, booking flows, payments, and user experiences.

- **Shared foundation:** reuse astronomy across chart and Panchang calculations.
- **Explicit conventions:** distinguish tropical charts from sidereal Panchang calculations.
- **Reproducible results:** identify the engine, data, settings, and rules used for each result.
- **Independent deployment:** run your own instance; a public source repository does not imply a public hosted API.
- **Privacy by design:** calculate from supplied inputs without requiring customer accounts or persistent birth records.

## Intended capabilities

| Area | Initial scope | Status |
| --- | --- | --- |
| Birth charts | Extract and preserve Astrel's existing chart behavior after source review | Planned |
| Core Panchang | Tithi, nakshatra, yoga, karana, weekday, and transition times | Planned |
| Local timings | Sunrise, sunset, and derived periods such as Rahu Kalam | Planned |
| Calendar | Paksha, lunar months, Amanta/Purnimanta conventions, intercalary months | Planned |
| Observances | Reviewed festival and vrat rules, with regional distinctions | Later milestone |
| Operations | Versioned API, health checks, reproducible builds, source metadata | Planned |

India-wide location support is a goal. Comprehensive regional observance coverage will be added and documented incrementally. KRIPA does not currently claim parity with DrikPanchang.com or any other reference provider.

## Architecture

Astrel and BrahminBooking will call KRIPA from their own backends. KRIPA will accept calculation inputs and return structured results. It will not handle dating profiles, provider approval, bookings, payment settlements, or application authentication.

| Component | Responsibility |
| --- | --- |
| HTTP layer | Validate requests, enforce limits, return versioned responses |
| Chart module | Chart-specific calculations and conventions |
| Panchang module | Daily calendar calculations and transition searches |
| Astronomy adapter | Swiss Ephemeris calls, data selection, configuration isolation |
| Versioned rule profiles | Calendar conventions and reviewed observance rules |
| Reference suite | Regression cases, tolerances, provenance, and known discrepancies |

The implementation language and bindings will be chosen after reviewing the existing Astrel code and candidate Panchang libraries. Shared mutable Swiss Ephemeris configuration must be isolated or serialized so simultaneous requests cannot change each other's calculation settings.

## Proposed API

These routes are a starting contract and may change before the first release.

| Method | Route | Purpose |
| --- | --- | --- |
| `POST` | `/v1/charts` | Calculate a chart from time, location, and an explicit profile |
| `POST` | `/v1/panchang` | Calculate Panchang for a local date, location, and tradition profile |
| `GET` | `/health/live` | Check process availability |
| `GET` | `/health/ready` | Check engine and required data readiness |
| `GET` | `/v1/meta` | Report version, supported profiles, data coverage, and source URL |

The first implementation should publish an OpenAPI contract with executable examples. A successful response should include:

- Requested location and IANA timezone.
- Resolved date/time and UTC offset.
- Calculation profile and convention details.
- Engine, ephemeris data, and rule-set versions.
- Timestamped transitions, including events on the following local date.
- Warnings or unsupported conditions rather than guessed results.

## Calculation conventions

Every supported profile must document:

1. Tropical or sidereal zodiac; ayanamsa where applicable.
2. Sunrise/sunset definition, refraction, and elevation treatment.
3. Timezone conversion and handling of ambiguous or nonexistent local times.
4. House system and node selection where relevant to charts.
5. Lunar calendar tradition and observance rules where relevant to Panchang.
6. Supported date range and required ephemeris files.

The Panchang response must distinguish the value at sunrise from the value at another requested instant. It must support multiple transitions instead of assuming one tithi or nakshatra per civil day.

## Accuracy and validation

Accuracy is a release requirement, not a marketing claim. The validation suite will cover geographically varied Indian locations and difficult boundaries, including transitions near sunrise and midnight, skipped or repeated tithis, lunar-month boundaries, and leap lunar months.

Reference cases must record their source, input coordinates, timezone, conventions, expected results, and agreed tolerances. Differences must be explained before being accepted. Related forks are not independent validation sources. Festival and vrat rules require review by a knowledgeable calendar specialist in addition to astronomical testing.

Astrel extraction will require regression comparisons against its existing calculator before switching consumers to KRIPA.

## Running your own instance

**Installation is not available yet.** The first runnable release must include:

- Complete implementation source and dependency lockfiles or equivalent version pins.
- Build files and a container deployment example.
- Instructions for obtaining the exact required ephemeris data, with checksums and applicable notices.
- Example configuration containing placeholders only.
- Local startup commands and a working calculation example.
- Test commands, supported platforms, and upgrade instructions.

Until those artifacts exist and are tested, this README deliberately does not present speculative commands as working setup instructions.

## Deployment and privacy

The intended production setup uses private backend-to-backend access. Operators choose where to host KRIPA and whether to expose any endpoint publicly.

- Authenticate service callers when requests cross a trust boundary.
- Enforce input validation, request size limits, timeouts, and rate limits.
- Avoid logging request bodies, birth details, precise coordinates, or credentials.
- Keep caches bounded and account for the sensitivity of their inputs.
- Include all calculation settings and version identifiers in cache keys.
- Return an explicit error if required data is absent; do not silently switch calculation methods.

Source publication must never include production secrets or personal records.

## Roadmap

- [ ] Review Astrel's calculator, upstream dependencies, and code ownership.
- [ ] Select bindings and pin astronomy dependencies and data.
- [ ] Extract the chart module and verify existing behavior.
- [ ] Evaluate reusable Panchang code and retain required attribution.
- [ ] Implement core Panchang and document supported conventions.
- [ ] Add API contracts, concurrency safeguards, and reference tests.
- [ ] Publish reproducible builds and self-hosting instructions.
- [ ] Integrate the service into Astrel and BrahminBooking.
- [ ] Add reviewed regional festival and vrat rules.

## Licensing and attribution

KRIPA is licensed under **GNU AGPL v3 or later** (`AGPL-3.0-or-later`); see [LICENSE](LICENSE). Preserve upstream copyright and license notices, and inventory the exact dependencies and data before incorporating or distributing them.

The public source must track released and deployed versions, including modifications and the material required to build and run the covered service. Consumers should display a readily discoverable source link beside calculation results, for example:

- **Chart calculations powered by KRIPA · Source code**
- **Panchang powered by KRIPA · Source code**

Separating services does not, by itself, determine the licensing obligations of an integrated application. Review the actual integration and the rights to extracted code before changing any existing application's license.

Planned foundation and evaluation references:

- [Swiss Ephemeris](https://www.astro.com/swisseph/) and its upstream license notices.
- [drik-panchanga](https://github.com/webresh/drik-panchanga), a candidate Panchang implementation to evaluate.
- [panchanga-cli](https://github.com/dhoomakethu/panchanga-cli), a related reference implementation to evaluate.

Listing a project here does not mean its code has already been incorporated. Add precise versions, provenance, notices, and modifications when reuse occurs. KRIPA is not affiliated with DrikPanchang.com.

## Contributing

Contributions should make calculations easier to reproduce and verify. A calculation fix should include a minimal example, the relevant convention, an independently justified expected result, and a regression test. New observance rules should document their tradition and review source.

Use synthetic examples in public issues. Do not submit personal birth details, production logs, credentials, or user records.
