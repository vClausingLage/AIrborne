package dcs

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"airborne/internal/plan"
)

func samplePlan() *plan.MissionPlan {
	return &plan.MissionPlan{
		Game:  "dcs",
		Title: plan.Localized{De: "Wadi-Schlag", En: "Wadi Strike"},
		Map:   "Syria", Date: "1982-06-10", Time: "06:40:00",
		Weather: plan.Weather{CloudLevel: 1500, CloudHeight: 6000, CloudConfig: `summer\00_Clear_00\sky.ini`, Turbulence: 3,
			WindLayers: [][3]int{{0, 150, 3}, {2000, 180, 6}, {5000, 190, 8}}},
		PlayerGroups: []plan.Group{{
			Name: "Rakete", Aircraft: "MiG-21Bis", Count: 2, Country: "Syria",
			Start: &plan.Start{Type: "air", Lat: 33.25, Lon: 36.0, Alt: 2500, Heading: 245},
			Route: []plan.Position{{Lat: 33.1, Lon: 35.72, Alt: 1500}},
		}},
		EnemyGroups: []plan.Group{
			{Name: "APC column", Kind: "vehicle", Script: "M-113", Count: 4, Country: "Israel",
				Position: &plan.Position{Lat: 33.1, Lon: 35.72, Head: 90}, Movement: &plan.Movement{Type: "static"}},
			{Name: "Tanks", Kind: "vehicle", Script: "M-60", Count: 2, Country: "Israel",
				Position: &plan.Position{Lat: 33.095, Lon: 35.73}, Movement: &plan.Movement{Type: "route"},
				Route: []plan.Position{{Lat: 33.09, Lon: 35.74}}},
			{Name: "Kfir CAP", Kind: "plane", Aircraft: "F-5E-3", Count: 2, Country: "Israel",
				Position: &plan.Position{Lat: 33.0, Lon: 35.6, Alt: 4000, Head: 60}},
		},
		FriendlyGroups: []plan.Group{{Name: "Hinds", Kind: "helicopter", Script: "Mi-24P", Count: 2, Country: "Syria",
			Position: &plan.Position{Lat: 33.15, Lon: 35.86, Alt: 300}}},
		Flak: []plan.Flak{{Script: "ZSU-23-4 Shilka", Count: 1, Position: plan.Position{Lat: 33.098, Lon: 35.725}, Country: "Israel"}},
		Objectives: []plan.Objective{{Title: plan.Localized{De: "Kolonne zerstören", En: "Destroy the column"},
			Desc: plan.Localized{De: "Alle APC und Panzer.", En: "All APCs and tanks."}, Counter: 6}},
		RadioQueue: []plan.Radio{
			{Trigger: "mission_begin_delay", Delay: 10, Speaker: "Basis", TextDe: "Rakete, hier Basis.", TextEn: "Rakete, base."},
			{Trigger: "check_zone", Zone: &plan.Zone{Lat: 33.1, Lon: 35.72, R: 8000}, Speaker: "Basis", TextDe: "Zielgebiet erreicht.", TextEn: "Target area."},
			{Trigger: "all_destroyed", Speaker: "Basis", TextDe: "Alle Ziele vernichtet.", TextEn: "All destroyed."},
		},
		Briefing: plan.Localized{De: "Briefing \"DE\"\nZeile 2", En: "Briefing EN"},
	}
}

// --- minimal Lua table reader for the generated files -----------------------

type luaParser struct {
	s   string
	pos int
}

func (p *luaParser) ws() {
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
		} else if c == '-' && p.pos+1 < len(p.s) && p.s[p.pos+1] == '-' {
			for p.pos < len(p.s) && p.s[p.pos] != '\n' {
				p.pos++
			}
		} else {
			return
		}
	}
}

func (p *luaParser) expect(c byte) error {
	p.ws()
	if p.pos >= len(p.s) || p.s[p.pos] != c {
		return fmt.Errorf("expected %q at %d: %q", c, p.pos, p.rest())
	}
	p.pos++
	return nil
}

func (p *luaParser) rest() string {
	if p.pos+40 < len(p.s) {
		return p.s[p.pos : p.pos+40]
	}
	return p.s[p.pos:]
}

func (p *luaParser) value() (any, error) {
	p.ws()
	if p.pos >= len(p.s) {
		return nil, fmt.Errorf("eof")
	}
	switch c := p.s[p.pos]; {
	case c == '{':
		p.pos++
		m := map[string]any{}
		for {
			p.ws()
			if p.s[p.pos] == '}' {
				p.pos++
				return m, nil
			}
			if err := p.expect('['); err != nil {
				return nil, err
			}
			key, err := p.value()
			if err != nil {
				return nil, err
			}
			if err := p.expect(']'); err != nil {
				return nil, err
			}
			if err := p.expect('='); err != nil {
				return nil, err
			}
			val, err := p.value()
			if err != nil {
				return nil, err
			}
			m[fmt.Sprint(key)] = val
			p.ws()
			if p.s[p.pos] == ',' {
				p.pos++
			}
		}
	case c == '"':
		p.pos++
		var b strings.Builder
		for p.pos < len(p.s) {
			ch := p.s[p.pos]
			if ch == '\\' {
				p.pos++
				b.WriteByte(p.s[p.pos])
				p.pos++
				continue
			}
			if ch == '"' {
				p.pos++
				return b.String(), nil
			}
			b.WriteByte(ch)
			p.pos++
		}
		return nil, fmt.Errorf("unterminated string")
	case strings.HasPrefix(p.s[p.pos:], "true"):
		p.pos += 4
		return true, nil
	case strings.HasPrefix(p.s[p.pos:], "false"):
		p.pos += 5
		return false, nil
	default:
		start := p.pos
		for p.pos < len(p.s) && strings.ContainsRune("-+.0123456789eE", rune(p.s[p.pos])) {
			p.pos++
		}
		if start == p.pos {
			return nil, fmt.Errorf("unexpected %q at %d", p.rest(), p.pos)
		}
		return strconv.ParseFloat(p.s[start:p.pos], 64)
	}
}

func parseLua(t *testing.T, name, text string) map[string]any {
	t.Helper()
	if !strings.HasPrefix(text, name+" = ") {
		t.Fatalf("%s: bad prefix %q", name, text[:20])
	}
	p := &luaParser{s: text[len(name)+3:]}
	v, err := p.value()
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return v.(map[string]any)
}

func get(t *testing.T, m any, path ...string) any {
	t.Helper()
	cur := m
	for _, key := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("path %v: not a table at %q", path, key)
		}
		cur, ok = mm[key]
		if !ok {
			t.Fatalf("path %v: key %q missing", path, key)
		}
	}
	return cur
}

func TestGenerateMiz(t *testing.T) {
	dir := t.TempDir()
	res, err := Generate(samplePlan(), dir)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(res.MainFile)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	files := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(data)
	}
	for _, want := range []string{"mission", "options", "theatre", "warehouses", "l10n/DEFAULT/dictionary", "l10n/DEFAULT/mapResource"} {
		if _, ok := files[want]; !ok {
			t.Fatalf("zip entry %s missing (have %v)", want, keys(files))
		}
	}
	if files["theatre"] != "Syria" {
		t.Fatalf("theatre = %q", files["theatre"])
	}
	m := parseLua(t, "mission", files["mission"])
	parseLua(t, "options", files["options"])
	parseLua(t, "warehouses", files["warehouses"])
	dict := parseLua(t, "dictionary", files["l10n/DEFAULT/dictionary"])

	if get(t, m, "theatre") != "Syria" || get(t, m, "start_time") != 6.0*3600+40*60 {
		t.Fatal("theatre/start_time wrong")
	}
	if get(t, m, "date", "Year") != 1982.0 || get(t, m, "date", "Month") != 6.0 {
		t.Fatal("date wrong")
	}
	// briefing text lands in the dictionary, referenced by key
	descKey := get(t, m, "descriptionText").(string)
	if !strings.Contains(dict[descKey].(string), `Briefing "DE"`) || !strings.Contains(dict[descKey].(string), "\nZeile 2") {
		t.Fatalf("briefing text: %q", dict[descKey])
	}

	// Syria player is red; Israel enemy is blue
	red := get(t, m, "coalition", "red", "country").(map[string]any)
	blue := get(t, m, "coalition", "blue", "country").(map[string]any)
	if len(red) != 1 || len(blue) != 1 {
		t.Fatalf("countries: red=%d blue=%d", len(red), len(blue))
	}
	syria := red["1"].(map[string]any)
	if syria["id"] != 47.0 {
		t.Fatalf("red country id %v", syria["id"])
	}
	pg := get(t, syria, "plane", "group", "1").(map[string]any)
	units := pg["units"].(map[string]any)
	if len(units) != 2 || get(t, units, "1", "skill") != "Client" || get(t, units, "1", "type") != "MiG-21Bis" {
		t.Fatalf("player units wrong: %v", units)
	}
	if _, ok := get(t, units, "1", "callsign").(float64); !ok {
		t.Fatal("syrian callsign should be numeric")
	}
	if n := len(get(t, pg, "route", "points").(map[string]any)); n != 2 {
		t.Fatalf("player route points = %d", n)
	}
	if _, ok := syria["helicopter"]; !ok {
		t.Fatal("friendly helicopters missing on red side")
	}
	israel := blue["1"].(map[string]any)
	vgroups := get(t, israel, "vehicle", "group").(map[string]any)
	if len(vgroups) != 3 { // APC, tanks, flak
		t.Fatalf("expected 3 vehicle groups, got %d", len(vgroups))
	}
	if _, ok := get(t, israel, "plane", "group", "1").(map[string]any)["units"]; !ok {
		t.Fatal("enemy CAP missing")
	}
	if get(t, israel, "plane", "group", "1", "task") != "CAP" {
		t.Fatal("enemy air task should be CAP")
	}

	// projection: Israel column near Golan -> plausible Syria-map coords (Khmeimim is ~ x=42000,y=6000)
	apc := get(t, vgroups, "1", "units", "1").(map[string]any)
	x, y := apc["x"].(float64), apc["y"].(float64)
	if math.Abs(x-(-214000)) > 15000 || math.Abs(y-(-24000)) > 15000 {
		t.Fatalf("projected APC position off: x=%.0f y=%.0f", x, y)
	}

	// triggers: 3 rules, matching actions/conditions/func/flags
	rules := get(t, m, "trigrules").(map[string]any)
	if len(rules) != 3 {
		t.Fatalf("trigrules = %d", len(rules))
	}
	for _, key := range []string{"actions", "conditions", "func", "flag"} {
		if n := len(get(t, m, "trig", key).(map[string]any)); n != 3 {
			t.Fatalf("trig.%s has %d entries", key, n)
		}
	}
	cond := get(t, m, "trig", "conditions", "3").(string)
	if !strings.Contains(cond, "c_group_dead(") || !strings.Contains(cond, " and ") {
		t.Fatalf("all_destroyed condition: %s", cond)
	}
	action := get(t, m, "trig", "actions", "1").(string)
	if !strings.HasPrefix(action, `a_out_text_delay(getValueDictByKey("DictKey_ActionText_`) {
		t.Fatalf("action string: %s", action)
	}
	if !strings.Contains(files["mission"], `getValueDictByKey(\"DictKey_ActionText_`) {
		t.Fatal("quotes inside action strings must be escaped like the mission editor does")
	}
	zones := get(t, m, "triggers", "zones").(map[string]any)
	if len(zones) != 1 || get(t, zones, "1", "radius") != 8000.0 {
		t.Fatalf("zones: %v", zones)
	}
	if get(t, m, "coalitions", "red", "1") != 47.0 {
		t.Fatal("coalitions.red should list Syria")
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestProjectionKhmeimim(t *testing.T) {
	terr, _ := lookupTerrain("syria")
	x, y := terr.toXY(35.4008, 35.9487)
	// reference mission parks a Mi-24 at Khmeimim around x=42925, y=6191
	if math.Abs(x-42925) > 2500 || math.Abs(y-6191) > 2500 {
		t.Fatalf("Khmeimim projected to x=%.0f y=%.0f", x, y)
	}
	if _, ok := lookupTerrain("Persian Gulf"); !ok {
		t.Fatal("alias lookup failed")
	}
}

func TestLuaStringEscaping(t *testing.T) {
	got := luaString("a\"b\\c\nd")
	if got != "\"a\\\"b\\\\c\\\nd\"" {
		t.Fatalf("got %q", got)
	}
}

func readMiz(t *testing.T, path string) map[string][]byte {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	files := map[string][]byte{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = data
	}
	return files
}

func writeTestPNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// Media: briefing picture -> l10n/DEFAULT + mapResource + pictureFileNameR
// (player is red here); kneeboards -> KNEEBOARD/IMAGES in page order.
func TestGenerateMizMedia(t *testing.T) {
	dir := t.TempDir()
	pic := filepath.Join(dir, "Lage Karte.png")
	kb := filepath.Join(dir, "Funkplan.png")
	writeTestPNG(t, pic)
	writeTestPNG(t, kb)
	mp := samplePlan()
	mp.Media = &plan.Media{BriefingImage: pic, Kneeboards: []string{kb}, KneeboardBriefing: true}
	res, err := Generate(mp, dir)
	if err != nil {
		t.Fatal(err)
	}
	files := readMiz(t, res.MainFile)
	if _, ok := files["l10n/DEFAULT/Lage_Karte.png"]; !ok {
		t.Fatalf("briefing picture missing: %v", mapKeys(files))
	}
	if _, ok := files["KNEEBOARD/IMAGES/01_briefing.png"]; !ok {
		t.Fatalf("rendered briefing page missing: %v", mapKeys(files))
	}
	if _, ok := files["KNEEBOARD/IMAGES/02_Funkplan.png"]; !ok {
		t.Fatalf("user kneeboard missing: %v", mapKeys(files))
	}
	if _, err := png.DecodeConfig(bytes.NewReader(files["KNEEBOARD/IMAGES/01_briefing.png"])); err != nil {
		t.Fatalf("briefing page is not a PNG: %v", err)
	}
	res1 := parseLua(t, "mapResource", string(files["l10n/DEFAULT/mapResource"]))
	if res1["ResKey_ImageBriefing_1"] != "Lage_Karte.png" {
		t.Fatalf("mapResource: %v", res1)
	}
	m := parseLua(t, "mission", string(files["mission"]))
	if get(t, m, "pictureFileNameR", "1") != "ResKey_ImageBriefing_1" {
		t.Fatalf("pictureFileNameR: %v", get(t, m, "pictureFileNameR"))
	}
	if got := get(t, m, "pictureFileNameB").(map[string]any); len(got) != 0 {
		t.Fatalf("blue (enemy) must get no picture: %v", got)
	}
}

// Challenge: every enemy group hidden, F10 restricted to own aircraft.
func TestGenerateMizChallenge(t *testing.T) {
	dir := t.TempDir()
	mp := samplePlan()
	mp.Challenge = true
	res, err := Generate(mp, dir)
	if err != nil {
		t.Fatal(err)
	}
	files := readMiz(t, res.MainFile)
	m := parseLua(t, "mission", string(files["mission"]))
	if get(t, m, "forcedOptions", "optionsView") != "optview_myaircraft" {
		t.Fatalf("forcedOptions: %v", get(t, m, "forcedOptions"))
	}
	blue := get(t, m, "coalition", "blue", "country").(map[string]any)
	hidden, total := 0, 0
	for _, c := range blue {
		for _, cat := range []string{"plane", "vehicle", "helicopter", "ship"} {
			cm, ok := c.(map[string]any)[cat]
			if !ok {
				continue
			}
			for _, g := range cm.(map[string]any)["group"].(map[string]any) {
				total++
				if g.(map[string]any)["hidden"] == true && g.(map[string]any)["hiddenOnPlanner"] == true {
					hidden++
				}
			}
		}
	}
	if total == 0 || hidden != total {
		t.Fatalf("hidden %d of %d enemy groups", hidden, total)
	}
	red := get(t, m, "coalition", "red", "country").(map[string]any)
	for _, c := range red {
		pg := get(t, c, "plane", "group").(map[string]any)
		for _, g := range pg {
			if g.(map[string]any)["hidden"] == true {
				t.Fatal("player side must stay visible")
			}
		}
	}
}

func mapKeys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
