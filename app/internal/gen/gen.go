// Package gen holds types shared by the IL-2 and DCS mission generators.
package gen

import (
	"regexp"
	"strings"
	"unicode"
)

// Result describes the files written by a generator.
type Result struct {
	Game      string   `json:"game"`
	Name      string   `json:"name"`
	OutputDir string   `json:"outputDir"`
	MainFile  string   `json:"mainFile"`
	Files     []string `json:"files"`
	Notes     []string `json:"notes"`
}

var nonWord = regexp.MustCompile(`[^A-Za-z0-9]+`)

// MissionName derives a safe file base name from a mission title.
// Generated missions are prefixed with "AB_" so they never clobber hand-made ones.
func MissionName(title string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(title) {
		switch r {
		case 'ä':
			b.WriteString("ae")
		case 'ö':
			b.WriteString("oe")
		case 'ü':
			b.WriteString("ue")
		case 'Ä':
			b.WriteString("Ae")
		case 'Ö':
			b.WriteString("Oe")
		case 'Ü':
			b.WriteString("Ue")
		case 'ß':
			b.WriteString("ss")
		default:
			if r > unicode.MaxASCII {
				b.WriteRune(' ')
			} else {
				b.WriteRune(r)
			}
		}
	}
	s := nonWord.ReplaceAllString(b.String(), "_")
	s = strings.Trim(s, "_")
	if len(s) > 48 {
		s = s[:48]
		s = strings.TrimRight(s, "_")
	}
	if s == "" {
		s = "Mission"
	}
	return "AB_" + s
}

// NonNil returns an empty slice instead of nil so JSON renders [] not null.
func NonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
