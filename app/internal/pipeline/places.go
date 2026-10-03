package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"airborne/internal/logging"
)

// Place is a named point of interest from the IL-2 Korea scene template, in
// game metres (x north, z east). The data is shared with the Zeno planning map
// (reference/il2-korea-places.json, extracted by zeno/tools/extract_places.py).
type Place struct {
	Cat  string  `json:"cat"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Z    float64 `json:"z"`
}

const placesFile = "il2-korea-places.json"

// maxMatchedCities caps the cities pulled in by name from the user's text.
const maxMatchedCities = 40

// majorCities are always listed; the other ~850 villages only when named.
var majorCities = []string{
	"Seoul", "Inchon", "Suwon", "Uijongbu", "Munsan-ni", "Kaesong", "Chorwon", "Kumhwa",
	"Chunchon", "Haeju", "Sariwon", "Pyongyang", "Chinnampo", "Anju", "Sinuiju",
	"Wonsan", "Hamhung", "Hungnam",
}

// placeAliases maps modern or German spellings to the names in the game data.
var placeAliases = map[string]string{
	"incheon":   "Inchon",
	"chuncheon": "Chunchon",
	"kimpo":     "K-14 Kimpo",
	"gimpo":     "K-14 Kimpo",
	"nampo":     "Chinnampo",
	"pjöngjang": "Pyongyang",
	"pjongjang": "Pyongyang",
	"cheorwon":  "Chorwon",
	"gaeseong":  "Kaesong",
	"hamheung":  "Hamhung",
	"heungnam":  "Hungnam",
}

// loadPlaces reads reference/il2-korea-places.json. A missing file only drops
// the place list from the prompt.
func loadPlaces(root string) []Place {
	path := filepath.Join(root, "reference", placesFile)
	data, err := os.ReadFile(path)
	if err != nil {
		logging.Infof("IL-2 Ortsliste nicht geladen (%s): %v", path, err)
		return nil
	}
	var places []Place
	if err := json.Unmarshal(data, &places); err != nil {
		logging.Errorf("IL-2 Ortsliste %s: %v", path, err)
		return nil
	}
	return places
}

// placesContext renders the place list for the IL-2 prompt: every airfield
// and military/industrial site, the major cities, plus any further city whose
// name occurs in text (user input or the plan being repaired).
func placesContext(places []Place, text string) string {
	if len(places) == 0 {
		return ""
	}
	wanted := map[string]bool{}
	for _, n := range majorCities {
		wanted[n] = true
	}
	lower := strings.ToLower(text)
	for alias, name := range placeAliases {
		if containsWord(lower, alias) {
			wanted[name] = true
		}
	}

	groups := map[string][]Place{}
	seen := map[string]bool{}
	matched := 0
	for _, pl := range places {
		key := pl.Cat + "|" + pl.Name
		if seen[key] {
			continue
		}
		cat := pl.Cat
		if cat == "city" {
			switch {
			case wanted[pl.Name]:
			case matched < maxMatchedCities && len(pl.Name) >= 4 && containsWord(lower, strings.ToLower(pl.Name)):
				matched++
				cat = "named"
			default:
				continue
			}
		}
		seen[key] = true
		groups[cat] = append(groups[cat], pl)
	}

	var b strings.Builder
	b.WriteString("\n\n## Orte auf der Karte (IL-2 Korea, x/z in Metern)\n")
	b.WriteString("Verwende fuer Starts, Ziele, Routen, Zonen und Icons diese Koordinaten (bzw. sinnvolle Versaetze davon) statt geschaetzter Werte. Bodenstarts an Flugplaetzen mit genau diesen Koordinaten.\n")
	sections := []struct{ cat, title string }{
		{"airfield", "Flugplaetze"},
		{"city", "Wichtige Staedte"},
		{"named", "In der Beschreibung genannte Orte"},
		{"camp", "Militaerlager"},
		{"dam", "Staudaemme"},
		{"tunnel", "Tunnel"},
		{"mine", "Bergwerke"},
		{"industry", "Industrie"},
		{"station", "Bahnhoefe"},
	}
	for _, s := range sections {
		list := groups[s.cat]
		if len(list) == 0 {
			continue
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		fmt.Fprintf(&b, "%s: ", s.title)
		for i, pl := range list {
			if i > 0 {
				b.WriteString("; ")
			}
			fmt.Fprintf(&b, "%s (%.0f, %.0f)", pl.Name, pl.X, pl.Z)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// containsWord reports whether word occurs in text (both lower case) and is
// not glued to further letters, so "Anju" does not match "Anjuri".
func containsWord(text, word string) bool {
	for off := 0; ; {
		i := strings.Index(text[off:], word)
		if i < 0 {
			return false
		}
		start, end := off+i, off+i+len(word)
		before, _ := utf8.DecodeLastRuneInString(text[:start])
		after, _ := utf8.DecodeRuneInString(text[end:])
		if !isWordRune(before) && !isWordRune(after) {
			return true
		}
		off = start + 1
	}
}

func isWordRune(r rune) bool {
	return r != utf8.RuneError && (unicode.IsLetter(r) || unicode.IsDigit(r))
}

// userText joins the step-by-step inputs for the place-name match.
func userText(inputs map[string]string) string {
	var parts []string
	for _, r := range Roles {
		parts = append(parts, inputs[r.Key])
	}
	return strings.Join(parts, "\n")
}
