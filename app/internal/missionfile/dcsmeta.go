package missionfile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"airborne/internal/plan"
)

// DCS: texts live in l10n/DEFAULT/dictionary as ["DictKey_x"] = "text";
// the mission references them (["sortie"], ["descriptionText"]). Date/time
// are ["start_time"] = seconds and ["date"] = { ["Day"], ["Year"], ["Month"] }.

// top matches a field at the first indentation level of the mission table
// (one tab in AIrborne output, four spaces in mission-editor files) so that
// group-level fields with the same name (start_time, ...) are skipped.
func top(field string) string {
	return `(?m)^[ \t]{1,4}\["` + regexp.QuoteMeta(field) + `"\]\s*=\s*`
}

var (
	dcsStartTimeRe = regexp.MustCompile(`(` + top("start_time") + `)(\d+)`)
	dcsDateBlockRe = regexp.MustCompile(`(?s)` + top("date") + `\{(.*?)\}`)
	dcsDateFieldRe = regexp.MustCompile(`(\["(Day|Year|Month)"\]\s*=\s*)(\d+)`)
	// ["key"] = "value" with Lua escapes (\" \\ \n and \<newline> continuation)
	luaStringEntryRe = regexp.MustCompile(`(?s)\["([^"]+)"\]\s*=\s*"((?:[^"\\]|\\.|\\\n)*)"`)
)

func dcsRef(mission, field string) string {
	re := regexp.MustCompile(top(field) + `"([^"]*)"`)
	if mm := re.FindStringSubmatch(mission); mm != nil {
		return mm[1]
	}
	return ""
}

// parseLuaStringTable reads every ["key"] = "string" pair of a flat table.
func parseLuaStringTable(text string) map[string]string {
	out := map[string]string{}
	for _, mm := range luaStringEntryRe.FindAllStringSubmatch(text, -1) {
		out[mm[1]] = luaUnescape(mm[2])
	}
	return out
}

func luaUnescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		i++
		switch s[i] {
		case 'n', '\n':
			b.WriteByte('\n')
		case 'r':
			// dropped: DCS writes \r\n pairs as "\<newline>" only
		case 't':
			b.WriteByte('\t')
		case '"', '\\', '\'':
			b.WriteByte(s[i])
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// luaEscape renders text the way the DCS mission editor does (line breaks
// as backslash + newline).
func luaEscape(s string) string {
	var b strings.Builder
	for _, r := range strings.ReplaceAll(s, "\r\n", "\n") {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString("\\\n")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// setLuaString replaces (or appends before the closing brace) one
// ["key"] = "..." entry in a flat table text.
func setLuaString(text, key, value string) string {
	re := regexp.MustCompile(`(?s)(\["` + regexp.QuoteMeta(key) + `"\]\s*=\s*)"(?:[^"\\]|\\.|\\\n)*"`)
	repl := `${1}"` + strings.ReplaceAll(luaEscape(value), "$", "$$") + `"`
	if re.MatchString(text) {
		return re.ReplaceAllString(text, repl)
	}
	entry := fmt.Sprintf("\t[%q] = \"%s\",\n", key, luaEscape(value))
	if i := strings.LastIndex(text, "}"); i >= 0 {
		return text[:i] + entry + text[i:]
	}
	return text + entry
}

// luaArrayValues returns the string values of ["field"] = { [1] = "a", ... }.
func luaArrayValues(mission, field string) []string {
	re := regexp.MustCompile(`(?s)` + top(field) + `\{(.*?)\}`)
	mm := re.FindStringSubmatch(mission)
	if mm == nil {
		return nil
	}
	var out []string
	itemRe := regexp.MustCompile(`\[\d+\]\s*=\s*"([^"]*)"`)
	for _, e := range itemRe.FindAllStringSubmatch(mm[1], -1) {
		out = append(out, e[1])
	}
	return out
}

// appendLuaArrayValue adds value to ["field"] = { ... } (created when missing).
func appendLuaArrayValue(mission, field, value string) (string, error) {
	re := regexp.MustCompile(`(?s)(` + top(field) + `)\{(.*?)\}`)
	mm := re.FindStringSubmatchIndex(mission)
	if mm == nil {
		return "", fmt.Errorf("Feld %s nicht in der mission-Datei gefunden", field)
	}
	body := mission[mm[4]:mm[5]]
	for _, v := range luaArrayValues(mission, field) {
		if v == value {
			return mission, nil
		}
	}
	n := len(regexp.MustCompile(`\[\d+\]\s*=`).FindAllString(body, -1)) + 1
	entry := fmt.Sprintf("\n\t\t[%d] = %q,\n\t", n, value)
	if strings.TrimSpace(body) == "" {
		body = entry
	} else {
		body = strings.TrimRight(body, " \t\r\n")
		if !strings.HasSuffix(body, ",") {
			body += ","
		}
		body += entry
	}
	return mission[:mm[4]] + body + mission[mm[5]:], nil
}

func (m *Mission) setResource(key, file string) error {
	name := "l10n/DEFAULT/mapResource"
	text, ok := m.files[name]
	if !ok {
		text = []byte("mapResource = \n{\n} -- end of mapResource\n")
	}
	m.files[name] = []byte(setLuaString(string(text), key, file))
	return nil
}

// unreferenceResource drops mapResource keys (and picture references) that
// point at a removed file.
func (m *Mission) unreferenceResource(file string) {
	name := "l10n/DEFAULT/mapResource"
	text, ok := m.files[name]
	if !ok {
		return
	}
	res := parseLuaStringTable(string(text))
	for key, f := range res {
		if f != file {
			continue
		}
		re := regexp.MustCompile(`(?m)^\s*\["` + regexp.QuoteMeta(key) + `"\]\s*=\s*"[^"]*",?\r?\n?`)
		text = []byte(re.ReplaceAllString(string(text), ""))
		if !m.protected() {
			itemRe := regexp.MustCompile(`(?m)^\s*\[\d+\]\s*=\s*"` + regexp.QuoteMeta(key) + `",?\r?\n?`)
			m.files["mission"] = []byte(itemRe.ReplaceAllString(string(m.files["mission"]), ""))
		}
	}
	m.files[name] = text
}

func (m *Mission) metaDCS() Meta {
	var meta Meta
	if m.protected() {
		return meta
	}
	mission := string(m.files["mission"])
	dict := parseLuaStringTable(string(m.files["l10n/DEFAULT/dictionary"]))
	lookup := func(field string) string {
		ref := dcsRef(mission, field)
		if v, ok := dict[ref]; ok {
			return v
		}
		return ref
	}
	meta.Title = plan.Localized{De: lookup("sortie")}
	meta.Briefing = plan.Localized{De: lookup("descriptionText")}
	if t := dcsStartTimeRe.FindStringSubmatch(mission); t != nil {
		secs, _ := strconv.Atoi(t[2])
		meta.Time = fmt.Sprintf("%02d:%02d:%02d", secs/3600, secs%3600/60, secs%60)
	}
	if d := dcsDateBlockRe.FindStringSubmatch(mission); d != nil {
		vals := map[string]int{}
		for _, f := range dcsDateFieldRe.FindAllStringSubmatch(d[1], -1) {
			vals[f[2]], _ = strconv.Atoi(f[3])
		}
		meta.Date = fmt.Sprintf("%04d-%02d-%02d", vals["Year"], vals["Month"], vals["Day"])
	}
	return meta
}

func (m *Mission) setMetaDCS(meta Meta) error {
	if m.protected() {
		return fmt.Errorf("geschuetzte Mission: Metadaten koennen nicht geaendert werden")
	}
	mission := string(m.files["mission"])
	dict := string(m.files["l10n/DEFAULT/dictionary"])
	setText := func(field, value string) {
		ref := dcsRef(mission, field)
		if ref == "" {
			ref = "DictKey_" + field + "_AB"
			mission = strings.Replace(mission, "\n{", fmt.Sprintf("\n{\n\t[%q] = %q,", field, ref), 1)
		}
		dict = setLuaString(dict, ref, value)
	}
	setText("sortie", meta.Title.Pick("de"))
	setText("descriptionText", meta.Briefing.Pick("de"))
	if meta.Time != "" {
		var h, mi, s int
		if n, err := fmt.Sscanf(meta.Time, "%d:%d:%d", &h, &mi, &s); err != nil && n < 2 {
			return fmt.Errorf("Zeit %q nicht im Format HH:MM:SS", meta.Time)
		}
		if !dcsStartTimeRe.MatchString(mission) {
			return fmt.Errorf("start_time nicht in der mission-Datei gefunden")
		}
		mission = dcsStartTimeRe.ReplaceAllString(mission, fmt.Sprintf("${1}%d", h*3600+mi*60+s))
	}
	if meta.Date != "" {
		var y, mo, d int
		if _, err := fmt.Sscanf(meta.Date, "%d-%d-%d", &y, &mo, &d); err != nil {
			return fmt.Errorf("Datum %q nicht im Format YYYY-MM-DD", meta.Date)
		}
		loc := dcsDateBlockRe.FindStringSubmatchIndex(mission)
		if loc == nil {
			return fmt.Errorf("date nicht in der mission-Datei gefunden")
		}
		vals := map[string]int{"Day": d, "Year": y, "Month": mo}
		block := dcsDateFieldRe.ReplaceAllStringFunc(mission[loc[2]:loc[3]], func(s string) string {
			f := dcsDateFieldRe.FindStringSubmatch(s)
			return f[1] + strconv.Itoa(vals[f[2]])
		})
		mission = mission[:loc[2]] + block + mission[loc[3]:]
	}
	m.files["mission"] = []byte(mission)
	m.files["l10n/DEFAULT/dictionary"] = []byte(dict)
	return nil
}
