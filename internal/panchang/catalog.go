// SPDX-License-Identifier: AGPL-3.0-or-later
package panchang

// The festival catalogue. Each entry is a rule, not a date: dates are computed
// for a place and year. Regions use ISO 3166-2 subdivision codes ("IN" for
// all of India) so clients can filter by state; communities name who mainly
// observes a festival. Entries are rule previews until reviewed.

type calendarKind int

const (
	lunarTithi          calendarKind = iota // Amanta month and tithi at a decisive time
	solarDayOne                             // first day of a regional solar month
	solarNakshatra                          // nakshatra during a regional solar month
	solarTithi                              // tithi during a regional solar month
	fridayBeforePurnima                     // the Friday on or before the month's Purnima
	holikaDahan                             // Phalguna Purnima pradosh with the Bhadra rule
	dayAfterHolika                          // Holi: the day after Holika Dahan
)

// Regional solar calendars begin a month on different days relative to the
// Sun's sidereal ingress (Sankranti).
type solarRule int

const (
	solarSunriseDay solarRule = iota // Sankranti day: the ingress's civil day if before sunset, else the next (Odia, Punjabi, pan-Indian)
	solarTamil                       // ingress before sunset: that civil day, else the next
	solarMalayalam                   // ingress before 3/5 of daytime: that civil day, else the next
	solarBengali                     // the civil day after the ingress (Bengali, Assamese)
)

var solarRuleNames = map[solarRule]string{solarSunriseDay: "Sankranti day", solarTamil: "Tamil (sunset)", solarMalayalam: "Malayalam (aparahna)", solarBengali: "Bengali (next day)"}

type festival struct {
	id, name, category, scope, tradition string
	regions                              []string // display labels
	codes                                []string // ISO 3166-2 codes
	communities                          []string
	kind                                 calendarKind
	month                                int // Amanta month (lunar) or sidereal sign (solar), both from 0
	tithi                                int // 1-30
	nakshatra                            int // 1-27
	when                                 decisive
	solar                                solarRule
	offset                               int  // days after the matched day (solarDayOne only)
	adhika                               bool // also observed in an Adhika month
	later                                bool // when two days qualify, take the later
}

var (
	allIndia    = []string{"All India"}
	codesIndia  = []string{"IN"}
	northCodes  = []string{"IN-DL", "IN-UP", "IN-UT", "IN-HR", "IN-PB", "IN-HP", "IN-RJ", "IN-MP", "IN-CH", "IN-JK", "IN-BR", "IN-JH", "IN-CT"}
	northWest   = append(append([]string{}, northCodes...), "IN-MH", "IN-GJ", "IN-GA")
	eastCodes   = []string{"IN-WB", "IN-OR", "IN-AS", "IN-TR", "IN-BR", "IN-JH"}
	bengalCodes = []string{"IN-WB", "IN-AS", "IN-TR", "IN-OR"}
	southCodes  = []string{"IN-TN", "IN-KL", "IN-KA", "IN-AP", "IN-TG", "IN-PY"}
	purvanchal  = []string{"IN-BR", "IN-JH", "IN-UP"}
)

// Amanta months: Chaitra=0 … Phalguna=11. Signs: Mesha=0 … Meena=11.
// Krishna-paksha festivals sit under the Amanta month preceding their
// Purnimanta name (e.g. Janmashtami: Amanta Shravana, tithi 23).
var catalogue = []festival{
	// Pan-Indian lunar festivals (unchanged ids from calendar rules v1)
	{id: "vasant_panchami", name: "Vasant Panchami", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 10, tithi: 5, when: atSunrise},
	{id: "maha_shivaratri", name: "Maha Shivaratri", category: "festival", scope: "widely_observed", tradition: "Shaiva", regions: allIndia, codes: codesIndia, month: 10, tithi: 29, when: atNishita},
	{id: "holika_dahan", name: "Holika Dahan", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: []string{"North and West India"}, codes: northWest, kind: holikaDahan, month: 11, tithi: 15, when: atPradosh},
	{id: "dol_jatra", name: "Dol Jatra · Dol Purnima", category: "regional", scope: "regional", tradition: "Vaishnava", regions: []string{"West Bengal", "Odisha", "Assam"}, codes: bengalCodes, communities: []string{"Bengali", "Odia", "Assamese"}, month: 11, tithi: 15, when: atSunrise},
	{id: "holi", name: "Holi", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: []string{"North and West India"}, codes: northWest, kind: dayAfterHolika, month: 11, tithi: 16},
	{id: "chaitra_navaratri", name: "Chaitra Navaratri begins", category: "festival", scope: "widely_observed", tradition: "Shakta", regions: allIndia, codes: codesIndia, month: 0, tithi: 1, when: atSunrise},
	{id: "ugadi", name: "Ugadi", category: "regional", scope: "regional", tradition: "Regional new year", regions: []string{"Karnataka", "Andhra Pradesh", "Telangana"}, codes: []string{"IN-KA", "IN-AP", "IN-TG"}, communities: []string{"Kannadiga", "Telugu"}, month: 0, tithi: 1, when: atSunrise},
	{id: "gudi_padwa", name: "Gudi Padwa", category: "regional", scope: "regional", tradition: "Regional new year", regions: []string{"Maharashtra", "Goa"}, codes: []string{"IN-MH", "IN-GA"}, communities: []string{"Marathi", "Konkani"}, month: 0, tithi: 1, when: atSunrise},
	{id: "navreh", name: "Navreh", category: "regional", scope: "regional", tradition: "Kashmiri new year", regions: []string{"Jammu and Kashmir"}, codes: []string{"IN-JK"}, communities: []string{"Kashmiri Pandit"}, month: 0, tithi: 1, when: atSunrise},
	{id: "cheti_chand", name: "Cheti Chand", category: "regional", scope: "regional", tradition: "Sindhi new year", regions: []string{"Sindhi community"}, codes: []string{"IN-GJ", "IN-MH", "IN-RJ"}, communities: []string{"Sindhi"}, month: 0, tithi: 2, when: atSunrise},
	{id: "gangaur", name: "Gangaur", category: "regional", scope: "regional", tradition: "Shakta", regions: []string{"Rajasthan", "Madhya Pradesh"}, codes: []string{"IN-RJ", "IN-MP"}, communities: []string{"Rajasthani"}, month: 0, tithi: 3, when: atSunrise},
	{id: "rama_navami", name: "Rama Navami", category: "jayanti", scope: "widely_observed", tradition: "Vaishnava", regions: allIndia, codes: codesIndia, month: 0, tithi: 9, when: atMidday},
	{id: "mahavir_jayanti", name: "Mahavir Jayanti", category: "jayanti", scope: "widely_observed", tradition: "Jain", regions: allIndia, codes: codesIndia, communities: []string{"Jain"}, month: 0, tithi: 13, when: atSunrise},
	{id: "hanuman_jayanti", name: "Hanuman Jayanti", category: "jayanti", scope: "regional", tradition: "North Indian convention", regions: []string{"North India"}, codes: northCodes, month: 0, tithi: 15, when: atSunrise},
	{id: "akshaya_tritiya", name: "Akshaya Tritiya", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 1, tithi: 3, when: atPurvahna, later: true},
	{id: "parashurama_jayanti", name: "Parashurama Jayanti", category: "jayanti", scope: "widely_observed", tradition: "Vaishnava", regions: allIndia, codes: codesIndia, month: 1, tithi: 3, when: atPradosh},
	{id: "sita_navami", name: "Sita Navami", category: "jayanti", scope: "regional", tradition: "Vaishnava", regions: []string{"North India", "Nepal"}, codes: append(append([]string{}, northCodes...), "NP"), communities: []string{"Maithil"}, month: 1, tithi: 9, when: atMidday},
	{id: "narasimha_jayanti", name: "Narasimha Jayanti", category: "jayanti", scope: "widely_observed", tradition: "Vaishnava", regions: allIndia, codes: codesIndia, month: 1, tithi: 14, when: atPradosh, later: true},
	{id: "buddha_purnima", name: "Buddha Purnima", category: "jayanti", scope: "widely_observed", tradition: "Buddhist and Vaishnava", regions: allIndia, codes: codesIndia, communities: []string{"Buddhist"}, month: 1, tithi: 15, when: atSunrise},
	{id: "ganga_dussehra", name: "Ganga Dussehra", category: "festival", scope: "regional", tradition: "Smarta", regions: []string{"Uttarakhand", "Uttar Pradesh", "Bihar"}, codes: []string{"IN-UT", "IN-UP", "IN-BR"}, month: 2, tithi: 10, when: atSunrise, adhika: true},
	{id: "vat_purnima", name: "Vat Purnima", category: "vrat", scope: "regional", tradition: "Smarta", regions: []string{"Maharashtra", "Gujarat", "Goa", "Karnataka"}, codes: []string{"IN-MH", "IN-GJ", "IN-GA", "IN-KA"}, communities: []string{"Marathi", "Gujarati"}, month: 2, tithi: 15, when: atMidday},
	{id: "vat_savitri", name: "Vat Savitri Vrat", category: "vrat", scope: "regional", tradition: "Smarta", regions: []string{"North India"}, codes: northCodes, communities: []string{"Bhojpuri", "Maithil"}, month: 1, tithi: 30, when: atMidday},
	{id: "rath_yatra", name: "Jagannath Rath Yatra", category: "festival", scope: "regional", tradition: "Vaishnava", regions: []string{"Odisha", "All India"}, codes: []string{"IN-OR", "IN"}, communities: []string{"Odia"}, month: 3, tithi: 2, when: atSunrise},
	{id: "guru_purnima", name: "Guru Purnima", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 3, tithi: 15, when: atSunrise},
	{id: "hariyali_teej", name: "Hariyali Teej", category: "vrat", scope: "regional", tradition: "Shakta", regions: []string{"Rajasthan", "Haryana", "Uttar Pradesh", "Madhya Pradesh"}, codes: []string{"IN-RJ", "IN-HR", "IN-UP", "IN-MP", "IN-DL"}, communities: []string{"Rajasthani"}, month: 4, tithi: 3, when: atSunrise},
	{id: "nag_panchami", name: "Nag Panchami", category: "festival", scope: "regional", tradition: "Smarta", regions: []string{"North and West India"}, codes: northWest, month: 4, tithi: 5, when: atSunrise},
	{id: "varalakshmi_vratam", name: "Varalakshmi Vratam", category: "vrat", scope: "regional", tradition: "Smarta", regions: []string{"Karnataka", "Andhra Pradesh", "Telangana", "Tamil Nadu"}, codes: []string{"IN-KA", "IN-AP", "IN-TG", "IN-TN"}, communities: []string{"Kannadiga", "Telugu", "Tamil"}, kind: fridayBeforePurnima, month: 4},
	{id: "raksha_bandhan", name: "Raksha Bandhan", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 4, tithi: 15, when: atSunrise},
	{id: "krishna_janmashtami", name: "Krishna Janmashtami", category: "jayanti", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 4, tithi: 23, when: atNishita},
	{id: "hartalika_teej", name: "Hartalika Teej · Gowri Habba", category: "vrat", scope: "regional", tradition: "Shakta", regions: []string{"North India", "Maharashtra", "Karnataka"}, codes: append(append([]string{}, northCodes...), "IN-MH", "IN-KA"), communities: []string{"Bhojpuri", "Marathi", "Kannadiga"}, month: 5, tithi: 3, when: atSunrise},
	{id: "ganesh_chaturthi", name: "Ganesh Chaturthi", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 5, tithi: 4, when: atMidday},
	{id: "rishi_panchami", name: "Rishi Panchami", category: "vrat", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 5, tithi: 5, when: atMidday},
	{id: "nuakhai", name: "Nuakhai", category: "regional", scope: "regional", tradition: "Harvest", regions: []string{"Odisha"}, codes: []string{"IN-OR"}, communities: []string{"Odia"}, month: 5, tithi: 5, when: atSunrise},
	{id: "radha_ashtami", name: "Radha Ashtami", category: "jayanti", scope: "regional", tradition: "Vaishnava", regions: []string{"Uttar Pradesh (Braj)", "North India"}, codes: northCodes, month: 5, tithi: 8, when: atSunrise},
	{id: "anant_chaturdashi", name: "Anant Chaturdashi", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 5, tithi: 14, when: atSunrise},
	{id: "pitru_paksha", name: "Pitru Paksha begins", category: "panchang_event", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 5, tithi: 16, when: atSunrise},
	{id: "jitiya", name: "Jitiya · Jivitputrika Vrat", category: "vrat", scope: "regional", tradition: "Smarta", regions: []string{"Bihar", "Jharkhand", "Eastern Uttar Pradesh", "Nepal"}, codes: append(append([]string{}, purvanchal...), "NP"), communities: []string{"Bhojpuri", "Maithil"}, month: 5, tithi: 23, when: atSunrise},
	{id: "sarva_pitru_amavasya", name: "Sarva Pitru Amavasya", category: "panchang_event", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 5, tithi: 30, when: atAparahna},
	{id: "mahalaya", name: "Mahalaya", category: "regional", scope: "regional", tradition: "Shakta", regions: []string{"West Bengal", "Odisha", "Assam", "Tripura"}, codes: bengalCodes, communities: []string{"Bengali", "Odia", "Assamese"}, month: 5, tithi: 30, when: atSunrise},
	{id: "bathukamma", name: "Bathukamma begins", category: "regional", scope: "regional", tradition: "Shakta", regions: []string{"Telangana"}, codes: []string{"IN-TG"}, communities: []string{"Telugu"}, month: 5, tithi: 30, when: atSunrise},
	{id: "sharad_navaratri", name: "Sharad Navaratri begins", category: "festival", scope: "widely_observed", tradition: "Shakta", regions: allIndia, codes: codesIndia, month: 6, tithi: 1, when: atSunrise},
	{id: "durga_puja", name: "Durga Puja · Maha Shashthi", category: "regional", scope: "regional", tradition: "Shakta", regions: []string{"West Bengal", "Odisha", "Assam", "Tripura", "Bihar", "Jharkhand"}, codes: eastCodes, communities: []string{"Bengali", "Odia", "Assamese"}, month: 6, tithi: 6, when: atSunrise},
	{id: "durga_ashtami", name: "Durga Ashtami", category: "festival", scope: "widely_observed", tradition: "Shakta", regions: allIndia, codes: codesIndia, month: 6, tithi: 8, when: atSunrise},
	{id: "maha_navami", name: "Maha Navami", category: "festival", scope: "widely_observed", tradition: "Shakta", regions: allIndia, codes: codesIndia, month: 6, tithi: 9, when: atAparahna},
	{id: "vijayadashami", name: "Vijayadashami", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 6, tithi: 10, when: atAparahna},
	{id: "sharad_purnima", name: "Sharad Purnima", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 6, tithi: 15, when: atPradosh},
	{id: "kojagari_lakshmi_puja", name: "Kojagari Lakshmi Puja", category: "regional", scope: "regional", tradition: "Shakta", regions: []string{"West Bengal", "Odisha", "Assam", "Tripura"}, codes: bengalCodes, communities: []string{"Bengali", "Odia", "Assamese"}, month: 6, tithi: 15, when: atNishita},
	{id: "valmiki_jayanti", name: "Valmiki Jayanti", category: "jayanti", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 6, tithi: 15, when: atSunrise},
	{id: "karva_chauth", name: "Karva Chauth", category: "vrat", scope: "regional", tradition: "North Indian convention", regions: []string{"North India"}, codes: northCodes, communities: []string{"Punjabi", "Rajasthani"}, month: 6, tithi: 19, when: atPradosh},
	{id: "dhanteras", name: "Dhanteras", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 6, tithi: 28, when: atPradosh},
	{id: "naraka_chaturdashi", name: "Naraka Chaturdashi", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 6, tithi: 29, when: atSunrise},
	{id: "diwali", name: "Diwali · Lakshmi Puja", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 6, tithi: 30, when: atPradosh},
	{id: "kali_puja", name: "Kali Puja", category: "regional", scope: "regional", tradition: "Shakta", regions: []string{"West Bengal", "Assam", "Odisha", "Tripura"}, codes: bengalCodes, communities: []string{"Bengali", "Assamese"}, month: 6, tithi: 30, when: atNishita},
	{id: "govardhan_puja", name: "Govardhan Puja", category: "festival", scope: "regional", tradition: "Vaishnava", regions: []string{"North India"}, codes: northCodes, month: 7, tithi: 1, when: atSunrise},
	{id: "bestu_varas", name: "Bestu Varas · Gujarati New Year", category: "regional", scope: "regional", tradition: "Regional new year", regions: []string{"Gujarat"}, codes: []string{"IN-GJ"}, communities: []string{"Gujarati"}, month: 7, tithi: 1, when: atSunrise},
	{id: "bali_pratipada", name: "Bali Pratipada", category: "regional", scope: "regional", tradition: "Smarta", regions: []string{"Maharashtra", "Karnataka", "Goa"}, codes: []string{"IN-MH", "IN-KA", "IN-GA"}, communities: []string{"Marathi", "Kannadiga"}, month: 7, tithi: 1, when: atSunrise},
	{id: "bhai_dooj", name: "Bhai Dooj", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 7, tithi: 2, when: atMidday},
	{id: "chhath_puja", name: "Chhath Puja", category: "regional", scope: "regional", tradition: "Regional", regions: []string{"Bihar", "Jharkhand", "Eastern Uttar Pradesh"}, codes: append(append([]string{}, purvanchal...), "NP"), communities: []string{"Bhojpuri", "Maithil", "Magahi"}, month: 7, tithi: 6, when: atSunrise},
	{id: "skanda_sashti", name: "Skanda Sashti · Soorasamharam", category: "regional", scope: "regional", tradition: "Shaiva (Murugan)", regions: []string{"Tamil Nadu"}, codes: []string{"IN-TN", "IN-PY"}, communities: []string{"Tamil"}, month: 7, tithi: 6, when: atSunrise},
	{id: "tulsi_vivah", name: "Tulsi Vivah", category: "festival", scope: "widely_observed", tradition: "Vaishnava", regions: []string{"North and West India"}, codes: northWest, month: 7, tithi: 12, when: atSunrise},
	{id: "kartika_purnima", name: "Kartika Purnima · Dev Deepawali", category: "festival", scope: "widely_observed", tradition: "Smarta", regions: allIndia, codes: codesIndia, month: 7, tithi: 15, when: atSunrise},
	{id: "vivah_panchami", name: "Vivah Panchami", category: "regional", scope: "regional", tradition: "Vaishnava", regions: []string{"North India", "Nepal"}, codes: append(append([]string{}, northCodes...), "NP"), communities: []string{"Maithil"}, month: 8, tithi: 5, when: atSunrise},
	{id: "dattatreya_jayanti", name: "Dattatreya Jayanti", category: "jayanti", scope: "regional", tradition: "Smarta", regions: []string{"Maharashtra", "Karnataka", "Gujarat"}, codes: []string{"IN-MH", "IN-KA", "IN-GJ"}, communities: []string{"Marathi"}, month: 8, tithi: 15, when: atPradosh},

	// Regional solar calendars
	{id: "lohri", name: "Lohri", category: "regional", scope: "regional", tradition: "Harvest", regions: []string{"Punjab", "Haryana", "Himachal Pradesh", "Delhi"}, codes: []string{"IN-PB", "IN-HR", "IN-HP", "IN-DL", "IN-CH"}, communities: []string{"Punjabi", "Sikh"}, kind: solarDayOne, month: 9, solar: solarSunriseDay, offset: -1},
	{id: "uttarayan", name: "Uttarayan", category: "regional", scope: "regional", tradition: "Harvest", regions: []string{"Gujarat"}, codes: []string{"IN-GJ"}, communities: []string{"Gujarati"}, kind: solarDayOne, month: 9, solar: solarSunriseDay},
	{id: "bhogi", name: "Bhogi", category: "regional", scope: "regional", tradition: "Harvest", regions: []string{"Tamil Nadu", "Andhra Pradesh", "Telangana"}, codes: []string{"IN-TN", "IN-AP", "IN-TG", "IN-PY"}, communities: []string{"Tamil", "Telugu"}, kind: solarDayOne, month: 9, solar: solarTamil, offset: -1},
	{id: "pongal", name: "Thai Pongal", category: "regional", scope: "regional", tradition: "Harvest", regions: []string{"Tamil Nadu", "Puducherry"}, codes: []string{"IN-TN", "IN-PY"}, communities: []string{"Tamil"}, kind: solarDayOne, month: 9, solar: solarTamil},
	{id: "mattu_pongal", name: "Mattu Pongal", category: "regional", scope: "regional", tradition: "Harvest", regions: []string{"Tamil Nadu"}, codes: []string{"IN-TN", "IN-PY"}, communities: []string{"Tamil"}, kind: solarDayOne, month: 9, solar: solarTamil, offset: 1},
	{id: "magh_bihu", name: "Magh Bihu", category: "regional", scope: "regional", tradition: "Harvest", regions: []string{"Assam"}, codes: []string{"IN-AS"}, communities: []string{"Assamese"}, kind: solarDayOne, month: 9, solar: solarBengali},
	{id: "thaipusam", name: "Thaipusam", category: "regional", scope: "regional", tradition: "Shaiva (Murugan)", regions: []string{"Tamil Nadu", "Kerala"}, codes: []string{"IN-TN", "IN-KL", "IN-PY"}, communities: []string{"Tamil"}, kind: solarNakshatra, month: 9, nakshatra: 8, when: atSunrise, solar: solarTamil},
	{id: "baisakhi", name: "Baisakhi", category: "regional", scope: "regional", tradition: "Harvest new year", regions: []string{"Punjab", "Haryana"}, codes: []string{"IN-PB", "IN-HR", "IN-CH", "IN-DL"}, communities: []string{"Punjabi", "Sikh"}, kind: solarDayOne, month: 0, solar: solarSunriseDay},
	{id: "pana_sankranti", name: "Pana Sankranti · Odia New Year", category: "regional", scope: "regional", tradition: "Solar new year", regions: []string{"Odisha"}, codes: []string{"IN-OR"}, communities: []string{"Odia"}, kind: solarDayOne, month: 0, solar: solarSunriseDay},
	{id: "puthandu", name: "Puthandu · Tamil New Year", category: "regional", scope: "regional", tradition: "Solar new year", regions: []string{"Tamil Nadu", "Puducherry"}, codes: []string{"IN-TN", "IN-PY"}, communities: []string{"Tamil"}, kind: solarDayOne, month: 0, solar: solarTamil},
	{id: "vishu", name: "Vishu", category: "regional", scope: "regional", tradition: "Solar new year", regions: []string{"Kerala"}, codes: []string{"IN-KL"}, communities: []string{"Malayali"}, kind: solarDayOne, month: 0, solar: solarMalayalam},
	{id: "poila_baishakh", name: "Poila Baishakh · Bengali New Year", category: "regional", scope: "regional", tradition: "Solar new year", regions: []string{"West Bengal", "Tripura", "Assam"}, codes: []string{"IN-WB", "IN-TR", "IN-AS"}, communities: []string{"Bengali"}, kind: solarDayOne, month: 0, solar: solarBengali},
	{id: "bohag_bihu", name: "Bohag Bihu · Assamese New Year", category: "regional", scope: "regional", tradition: "Solar new year", regions: []string{"Assam"}, codes: []string{"IN-AS"}, communities: []string{"Assamese"}, kind: solarDayOne, month: 0, solar: solarBengali},
	{id: "onam", name: "Onam · Thiruvonam", category: "regional", scope: "regional", tradition: "Harvest", regions: []string{"Kerala"}, codes: []string{"IN-KL"}, communities: []string{"Malayali"}, kind: solarNakshatra, month: 4, nakshatra: 22, when: atSunrise, solar: solarMalayalam},
	{id: "vishwakarma_puja", name: "Vishwakarma Puja", category: "regional", scope: "regional", tradition: "Smarta", regions: []string{"West Bengal", "Bihar", "Jharkhand", "Odisha", "Assam", "Tripura"}, codes: eastCodes, communities: []string{"Bengali", "Bhojpuri", "Odia"}, kind: solarDayOne, month: 5, solar: solarSunriseDay},
	{id: "karthigai_deepam", name: "Karthigai Deepam", category: "regional", scope: "regional", tradition: "Shaiva", regions: []string{"Tamil Nadu"}, codes: []string{"IN-TN", "IN-PY"}, communities: []string{"Tamil"}, kind: solarNakshatra, month: 7, nakshatra: 3, when: atPradosh, solar: solarTamil},
	{id: "vaikuntha_ekadashi", name: "Vaikuntha Ekadashi", category: "festival", scope: "regional", tradition: "Vaishnava", regions: []string{"Tamil Nadu", "Andhra Pradesh", "Telangana", "Karnataka"}, codes: southCodes, communities: []string{"Tamil", "Telugu", "Kannadiga"}, kind: solarTithi, month: 8, tithi: 11, when: atSunrise, solar: solarTamil},
}

// Recurring observances computed separately (Ekadashi, Pradosh and so on)
// carry these ids; clients can hide them from festival lists.
var recurringIDs = map[string]bool{"ekadashi": true, "pradosh": true, "sankashti_chaturthi": true, "vinayaka_chaturthi": true, "masik_shivaratri": true, "purnima": true, "amavasya": true, "sankranti": true}

// Catalogue returns the rule catalogue as published metadata.
type CatalogueEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Scope       string   `json:"scope"`
	Tradition   string   `json:"tradition"`
	Regions     []string `json:"regions"`
	RegionCodes []string `json:"region_codes"`
	Communities []string `json:"communities"`
	Rule        string   `json:"rule"`
}

func Catalogue() []CatalogueEntry {
	out := make([]CatalogueEntry, 0, len(catalogue))
	for _, f := range catalogue {
		out = append(out, CatalogueEntry{f.id, f.name, f.category, f.scope, f.tradition, f.regions, f.codes, nonNil(f.communities), f.ruleText()})
	}
	return out
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
