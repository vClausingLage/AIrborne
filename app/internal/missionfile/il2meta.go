package missionfile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"airborne/internal/plan"
)

// IL-2 language files: "Index:Text" per line; 0 = title, 1 = briefing,
// 2 = author, line breaks inside a text are "<br>". Date/time live in the
// Options block of the .Mission ("Date = 14.9.1950;", "Time = 15:35:0;").

var (
	il2DateRe = regexp.MustCompile(`(?m)^(\s*Date\s*=\s*)(\d{1,2})\.(\d{1,2})\.(\d{4})(\s*;)`)
	il2TimeRe = regexp.MustCompile(`(?m)^(\s*Time\s*=\s*)(\d{1,2}):(\d{1,2}):(\d{1,2})(\s*;)`)
)

func (m *Mission) langFile(ext string) string { return m.baseName() + ext }

// langLines splits a language file into index -> text.
func langLines(text string) (map[int]string, []int) {
	lines := map[int]string{}
	var order []int
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		i := strings.Index(l, ":")
		if i <= 0 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(l[:i]))
		if err != nil {
			continue
		}
		lines[n] = l[i+1:]
		order = append(order, n)
	}
	return lines, order
}

func brToNewline(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	return strings.TrimSpace(s)
}

func newlineToBr(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "<br>")
}

func (m *Mission) metaIL2() Meta {
	var meta Meta
	read := func(ext string) map[int]string {
		data, ok := m.files[m.langFile(ext)]
		if !ok {
			return nil
		}
		lines, _ := langLines(decodeUTF16(data))
		return lines
	}
	ger, eng := read(".ger"), read(".eng")
	meta.Title = plan.Localized{De: brToNewline(ger[0]), En: brToNewline(eng[0])}
	meta.Briefing = plan.Localized{De: brToNewline(ger[1]), En: brToNewline(eng[1])}
	meta.Author = strings.TrimSpace(ger[2])
	if meta.Author == "" {
		meta.Author = strings.TrimSpace(eng[2])
	}
	mission := string(m.files[m.baseName()+".Mission"])
	if d := il2DateRe.FindStringSubmatch(mission); d != nil {
		meta.Date = fmt.Sprintf("%s-%s-%s", d[4], pad2(d[3]), pad2(d[2]))
	}
	if t := il2TimeRe.FindStringSubmatch(mission); t != nil {
		meta.Time = fmt.Sprintf("%s:%s:%s", pad2(t[2]), pad2(t[3]), pad2(t[4]))
	}
	return meta
}

func pad2(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

// setMetaIL2 rewrites lines 0-2 of every language file (German text into
// .ger, English into all others, falling back to whichever is set) and the
// Options date/time.
func (m *Mission) setMetaIL2(meta Meta) error {
	for ext := range il2LangExts {
		name := m.langFile(ext)
		data, ok := m.files[name]
		if !ok {
			continue
		}
		lang := "en"
		if ext == ".ger" {
			lang = "de"
		}
		text := decodeUTF16(data)
		lines, order := langLines(text)
		lines[0] = newlineToBr(meta.Title.Pick(lang))
		lines[1] = newlineToBr(meta.Briefing.Pick(lang))
		lines[2] = strings.TrimSpace(meta.Author)
		seen := map[int]bool{}
		var sb strings.Builder
		for _, n := range append([]int{0, 1, 2}, order...) {
			if seen[n] {
				continue
			}
			seen[n] = true
			sb.WriteString(strconv.Itoa(n) + ":" + lines[n] + "\r\n")
		}
		m.files[name] = encodeUTF16(sb.String())
	}
	name := m.baseName() + ".Mission"
	mission := string(m.files[name])
	if meta.Date != "" {
		var y, mo, d int
		if _, err := fmt.Sscanf(meta.Date, "%d-%d-%d", &y, &mo, &d); err != nil {
			return fmt.Errorf("Datum %q nicht im Format YYYY-MM-DD", meta.Date)
		}
		if !il2DateRe.MatchString(mission) {
			return fmt.Errorf("Options.Date nicht in der .Mission gefunden")
		}
		mission = il2DateRe.ReplaceAllString(mission, fmt.Sprintf("${1}%d.%d.%d${5}", d, mo, y))
	}
	if meta.Time != "" {
		var h, mi, s int
		if n, err := fmt.Sscanf(meta.Time, "%d:%d:%d", &h, &mi, &s); err != nil && n < 2 {
			return fmt.Errorf("Zeit %q nicht im Format HH:MM:SS", meta.Time)
		}
		if !il2TimeRe.MatchString(mission) {
			return fmt.Errorf("Options.Time nicht in der .Mission gefunden")
		}
		mission = il2TimeRe.ReplaceAllString(mission, fmt.Sprintf("${1}%d:%d:%d${5}", h, mi, s))
	}
	m.files[name] = []byte(mission)
	return nil
}
