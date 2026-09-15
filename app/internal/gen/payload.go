package gen

import (
	"strings"
	"unicode"
)

// PayloadTokens normalises loadout text such as "AIM-120C*2, FUEL*3" or
// "2x FAB-250, volle Kanonen" into a comparable token set
// ({aim120c, fab250, ...}); counts and filler words drop out. Used to match
// the AI's free-text payload wish against preset names.
func PayloadTokens(s string) map[string]bool {
	out := map[string]bool{}
	s = strings.ToLower(s)
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '+' || r == '*' || r == '(' || r == ')' || r == '/' || r == ';' || unicode.IsSpace(r)
	}) {
		// "2xb8v20" -> "b8v20"
		if i := strings.IndexByte(part, 'x'); i > 0 && i <= 2 && isDigits(part[:i]) {
			part = part[i+1:]
		}
		clean := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, part)
		if len(clean) < 2 || isDigits(clean) {
			continue
		}
		switch clean {
		case "fuel", "tank", "tanks", "ptb", "volle", "kanonen", "gun", "guns", "und", "and", "mit", "with",
			"bombs", "bomben", "rockets", "raketen", "lb", "kg", "default", "standard":
			continue
		}
		out[clean] = true
	}
	return out
}

// PayloadScore counts how many tokens of the wish appear in the preset name.
func PayloadScore(wish, name string) int {
	want := PayloadTokens(wish)
	have := PayloadTokens(name)
	score := 0
	for tok := range want {
		if have[tok] {
			score++
		}
	}
	return score
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
