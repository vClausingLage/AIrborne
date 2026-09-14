package dcs

import "strings"

// DCS country ids (Scripts/Database/db_countries.lua).
var countryIDs = map[string]int{
	"Russia": 0, "Ukraine": 1, "USA": 2, "Turkey": 3, "UK": 4, "France": 5, "Germany": 6,
	"USAF Aggressors": 7, "Canada": 8, "Spain": 9, "The Netherlands": 10, "Belgium": 11,
	"Norway": 12, "Denmark": 13, "Israel": 15, "Georgia": 16, "Insurgents": 17,
	"Abkhazia": 18, "South Ossetia": 19, "Italy": 20, "Australia": 21, "Switzerland": 22,
	"Austria": 23, "Belarus": 24, "Bulgaria": 25, "Czech Republic": 26, "China": 27,
	"Croatia": 28, "Egypt": 29, "Finland": 30, "Greece": 31, "Hungary": 32, "India": 33,
	"Iran": 34, "Iraq": 35, "Japan": 36, "Kazakhstan": 37, "North Korea": 38, "Pakistan": 39,
	"Poland": 40, "Romania": 41, "Saudi Arabia": 42, "Serbia": 43, "Slovakia": 44,
	"South Korea": 45, "Sweden": 46, "Syria": 47, "Yemen": 48, "Vietnam": 49, "Venezuela": 50,
	"Tunisia": 51, "Thailand": 52, "Sudan": 53, "Philippines": 54, "Morocco": 55, "Mexico": 56,
	"Malaysia": 57, "Libya": 58, "Jordan": 59, "Indonesia": 60, "Honduras": 61, "Ethiopia": 62,
	"Chile": 63, "Brazil": 64, "Bahrain": 65, "New Zealand": 66, "Yugoslavia": 67, "Kuwait": 68,
	"Portugal": 69, "GDR": 70, "Lebanon": 71, "Combined Joint Task Forces Blue": 80,
	"Combined Joint Task Forces Red": 81, "United Nations Peacekeepers": 82, "Argentina": 83,
	"Cyprus": 84, "Slovenia": 85, "Bolivia": 86, "Ghana": 87, "Nigeria": 88, "Peru": 89,
	"Ecuador": 90, "Italian Social Republic": 91, "Algeria": 92, "Cuba": 93,
}

var countryAliases = map[string]string{
	"united states": "USA", "united states of america": "USA", "us": "USA", "u.s.a.": "USA",
	"united kingdom": "UK", "great britain": "UK", "britain": "UK", "england": "UK",
	"russian federation": "Russia", "ussr": "Russia", "soviet union": "Russia", "udssr": "Russia",
	"sowjetunion": "Russia", "russland": "Russia", "deutschland": "Germany", "brd": "Germany",
	"ddr": "GDR", "east germany": "GDR", "west germany": "Germany", "frankreich": "France",
	"netherlands": "The Netherlands", "holland": "The Netherlands", "niederlande": "The Netherlands",
	"tuerkei": "Turkey", "türkei": "Turkey", "aegypten": "Egypt", "ägypten": "Egypt",
	"syrien": "Syria", "irak": "Iraq", "nordkorea": "North Korea", "dprk": "North Korea",
	"suedkorea": "South Korea", "südkorea": "South Korea", "rok": "South Korea",
	"cjtf blue": "Combined Joint Task Forces Blue", "cjtf red": "Combined Joint Task Forces Red",
	"aggressors": "USAF Aggressors", "un": "United Nations Peacekeepers", "italien": "Italy",
	"spanien": "Spain", "griechenland": "Greece", "polen": "Poland", "schweden": "Sweden",
	"norwegen": "Norway", "daenemark": "Denmark", "dänemark": "Denmark", "belgien": "Belgium",
	"oesterreich": "Austria", "österreich": "Austria", "schweiz": "Switzerland", "china (prc)": "China",
	"prc": "China", "jugoslawien": "Yugoslavia", "libyen": "Libya", "argentinien": "Argentina",
	"insurgent": "Insurgents", "rebels": "Insurgents", "rebellen": "Insurgents",
}

// canonicalCountry resolves a country string to a DCS country name.
func canonicalCountry(name string) (string, bool) {
	s := strings.TrimSpace(name)
	if s == "" {
		return "", false
	}
	if _, ok := countryIDs[s]; ok {
		return s, true
	}
	low := strings.ToLower(s)
	if a, ok := countryAliases[low]; ok {
		return a, true
	}
	for c := range countryIDs {
		if strings.ToLower(c) == low {
			return c, true
		}
	}
	return "", false
}

// redCountries are the countries DCS conventionally places on the red side.
var redCountries = map[string]bool{
	"Russia": true, "Belarus": true, "Syria": true, "Iran": true, "Iraq": true, "China": true,
	"North Korea": true, "Abkhazia": true, "South Ossetia": true, "Insurgents": true, "Libya": true,
	"Vietnam": true, "Venezuela": true, "Cuba": true, "Yugoslavia": true, "Serbia": true, "GDR": true,
	"Kazakhstan": true, "Combined Joint Task Forces Red": true, "Algeria": true, "Sudan": true,
	"Yemen": true, "Ethiopia": true, "Egypt": true, "USAF Aggressors": true,
}

// numericCallsignCountries use numeric board callsigns instead of NATO names.
var numericCallsignCountries = map[string]bool{
	"Russia": true, "Ukraine": true, "Belarus": true, "Kazakhstan": true, "Syria": true, "Iran": true,
	"Iraq": true, "China": true, "North Korea": true, "Abkhazia": true, "South Ossetia": true,
	"Insurgents": true, "Libya": true, "Algeria": true, "Egypt": true, "Vietnam": true,
	"Yugoslavia": true, "Serbia": true, "GDR": true, "Cuba": true, "Venezuela": true, "Sudan": true,
	"Yemen": true, "Ethiopia": true, "Combined Joint Task Forces Red": true,
}

// helicopterTypes lists DCS unit type prefixes that are helicopters.
var helicopterPrefixes = []string{"Mi-", "Ka-", "UH-", "AH-", "CH-", "OH-", "SA342", "SH-", "AH64", "Mi_", "Ka_"}

func isHelicopterType(t string) bool {
	for _, p := range helicopterPrefixes {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}

// defaultFuel gives a plausible internal fuel load (kg) per aircraft type.
var defaultFuel = map[string]float64{
	"MiG-21Bis": 2280, "MiG-15bis": 1172, "MiG-19P": 1800, "MiG-29A": 3376, "MiG-29S": 3376, "MiG-29G": 3376,
	"Su-25": 2835, "Su-25T": 3790, "Su-27": 9400, "Su-33": 9500, "J-11A": 9400, "JF-17": 2325,
	"F-86F Sabre": 1282, "F-5E-3": 2046, "F-4E-45MC": 4864, "F-14A-135-GR": 7348, "F-14B": 7348,
	"F-15C": 6103, "F-15ESE": 5952, "F-16C_50": 3249, "F-16C bl.52d": 3249, "FA-18C_hornet": 4900,
	"A-10A": 5029, "A-10C": 5029, "A-10C_2": 5029, "AV8BNA": 3519, "M-2000C": 3165, "Mirage-F1CE": 3100,
	"Mirage-F1EE": 3100, "AJS37": 4476, "L-39C": 823, "L-39ZA": 823, "C-101EB": 1796, "C-101CC": 1796,
	"A-4E-C": 2000, "MB-339A": 1400, "Su-34": 9800, "Tu-22M3": 50000, "B-52H": 100000,
	"P-51D": 340, "P-51D-30-NA": 340, "P-47D-30": 1140, "Spitfire LF Mk IX": 383, "Spitfire LF Mk IX CW": 383,
	"Bf-109K-4": 296, "FW-190D9": 292, "FW-190A8": 400, "MosquitoFBMkVI": 1100, "I-16": 200, "F4U-1D": 1100,
	"Mi-24P": 1704, "Mi-8MT": 1929, "Mi-8MTV2": 1929, "Ka-50": 1450, "Ka-50_3": 1450, "UH-1H": 631,
	"AH-64D_BLK_II": 1108, "SA342M": 416, "SA342L": 416, "SA342Mistral": 416, "OH-58D": 400, "CH-47Fbl1": 3000,
	"UH-60L": 1360, "Mi-28N": 1500, "AH-1W": 946,
}

func fuelFor(t string) float64 {
	if f, ok := defaultFuel[t]; ok {
		return f
	}
	if isHelicopterType(t) {
		return 1200
	}
	return 3000
}
