// Package i18n holds the UI language ("de" or "en") for every user-facing
// message the backend produces: errors, validation issues, generator notes
// and problems. Messages are written inline as German/English pairs, so the
// German text stays next to its translation and nothing can go missing.
package i18n

import (
	"fmt"
	"sync/atomic"
)

const (
	DE = "de"
	EN = "en"
	// Default is the language used until settings.json says otherwise.
	Default = DE
)

var lang atomic.Value

func init() { lang.Store(Default) }

// Valid reports whether l is a supported language code.
func Valid(l string) bool { return l == DE || l == EN }

// SetLang switches the message language; unknown codes fall back to Default.
func SetLang(l string) {
	if !Valid(l) {
		l = Default
	}
	lang.Store(l)
}

// Lang returns the current message language.
func Lang() string { return lang.Load().(string) }

// Other returns the second language for bilingual output.
func Other(l string) string {
	if l == EN {
		return DE
	}
	return EN
}

// T picks the German or English text.
func T(de, en string) string {
	if Lang() == EN {
		return en
	}
	return de
}

// Sprintf formats the German or English format string.
func Sprintf(de, en string, args ...any) string {
	return fmt.Sprintf(T(de, en), args...)
}

// Errorf is fmt.Errorf with a German/English format pair (supports %w).
func Errorf(de, en string, args ...any) error {
	return fmt.Errorf(T(de, en), args...)
}
