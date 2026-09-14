package il2

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"airborne/internal/plan"
)

func samplePlan() *plan.MissionPlan {
	return &plan.MissionPlan{
		Game:  "il2",
		Title: plan.Localized{De: "Operation Chromite - Jagd auf die Spione", En: "Operation Chromite - Spy Hunt"},
		Map:   "korea", Date: "1950-09-14", Time: "15:35:00",
		Weather: plan.Weather{CloudLevel: 1500, CloudHeight: 6000, CloudConfig: `summer\00_Clear_00\sky.ini`, SeaState: 3, Turbulence: 5,
			WindLayers: [][3]int{{0, 150, 4}, {500, 160, 5}}},
		PlayerGroups: []plan.Group{{
			Name: "Sowol", Aircraft: "il10", Count: 2, Country: float64(501),
			Start: &plan.Start{Type: "air", X: 86500, Z: 244000, Alt: 1000, Heading: 40},
			Route: []plan.Position{{X: 95000, Z: 242000, Alt: 700}, {X: 99800, Z: 261800, Alt: 500}, {X: 88000, Z: 247000, Alt: 800}},
		}},
		EnemyGroups: []plan.Group{
			{Name: "Trucks", Kind: "vehicle", Script: "vehicles/studebakerus6", Count: 3, Country: float64(601),
				Position: &plan.Position{X: 99870, Z: 261830, Head: 15}, Movement: &plan.Movement{Type: "static"}},
			{Name: "Jeep Patrol", Kind: "vehicle", Script: "vehicles/willysmb", Count: 1, Country: float64(601),
				Position: &plan.Position{X: 103080, Z: 264580, Head: 230}, Movement: &plan.Movement{Type: "route"},
				Route: []plan.Position{{X: 103400, Z: 264900}, {X: 102600, Z: 264200}}},
			{Name: "Mustangs", Kind: "plane", Aircraft: "f51d", Count: 2, Country: float64(601),
				Position: &plan.Position{X: 105000, Z: 268000, Alt: 2000, Head: 200}},
		},
		Flak:    []plan.Flak{{Script: "fixedobjects/boforsl60", Count: 2, Position: plan.Position{X: 100180, Z: 261640}, Country: float64(601), Engageable: true}},
		Statics: []plan.StaticObject{{Kind: "block", Script: "blocks/mil_camonet", Count: 2, Country: float64(601), Positions: []plan.Position{{X: 99800, Z: 261880, Head: 12}, {X: 99560, Z: 261860, Head: 192}}}},
		Objectives: []plan.Objective{{Title: plan.Localized{De: "Vernichtet die Aufklärer", En: "Destroy the scouts"},
			Desc: plan.Localized{De: "Alle Fahrzeuge zerstören.", En: "Destroy all vehicles."}, Counter: 4}},
		RadioQueue: []plan.Radio{
			{Trigger: "mission_begin_delay", Delay: 10, Speaker: "Basis", TextDe: "Sowol, hier Basis.", TextEn: "Sowol, this is base."},
			{Trigger: "check_zone", Zone: &plan.Zone{X: 99800, Z: 261800, R: 2500}, Speaker: "Basis", TextDe: "Feindkontakt!", TextEn: "Contact!"},
			{Trigger: "all_destroyed", Speaker: "Basis", TextDe: "Alle Ziele vernichtet.", TextEn: "All targets destroyed."},
		},
		Briefing: plan.Localized{De: "Briefing DE", En: "Briefing EN"},
		Icons:    []plan.Icon{{From: plan.Position{X: 86500, Z: 244000}, To: plan.Position{X: 99800, Z: 261800}, Label: plan.Localized{De: "Ziel", En: "Target"}}},
	}
}

type block struct {
	kind   string
	fields map[string]string
}

var fieldRe = regexp.MustCompile(`^\s*([A-Za-z]+)\s*=\s*(.*);\s*$`)

func parseMission(t *testing.T, text string) []block {
	t.Helper()
	var blocks []block
	lines := strings.Split(text, "\n")
	depth := 0
	var cur *block
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trim := strings.TrimSpace(line)
		switch {
		case trim == "{":
			depth++
		case trim == "}":
			depth--
			if depth == 0 && cur != nil {
				blocks = append(blocks, *cur)
				cur = nil
			}
		case depth == 0 && trim != "" && !strings.HasPrefix(trim, "#"):
			cur = &block{kind: trim, fields: map[string]string{}}
		case depth == 1 && cur != nil:
			if m := fieldRe.FindStringSubmatch(line); m != nil {
				cur.fields[m[1]] = m[2]
			}
		}
	}
	if depth != 0 {
		t.Fatalf("unbalanced braces (depth %d)", depth)
	}
	return blocks
}

func idList(s string) []int {
	s = strings.Trim(s, "[]")
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []int
	for _, p := range strings.Split(s, ",") {
		n, _ := strconv.Atoi(strings.TrimSpace(p))
		out = append(out, n)
	}
	return out
}

func TestGenerateStructure(t *testing.T) {
	dir := t.TempDir()
	res, err := Generate(samplePlan(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(res.MainFile, "AB_Operation_Chromite_Spy_Hunt.Mission") {
		t.Fatalf("unexpected file name %s", res.MainFile)
	}
	data, err := os.ReadFile(res.MainFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "# Mission File Version = 1.0;\r\n") {
		t.Fatalf("bad header: %q", text[:40])
	}
	blocks := parseMission(t, text)

	indices := map[int]block{}
	kinds := map[string]int{}
	for _, b := range blocks {
		kinds[b.kind]++
		if b.kind == "Options" {
			continue
		}
		idx, err := strconv.Atoi(b.fields["Index"])
		if err != nil {
			t.Fatalf("%s without Index", b.kind)
		}
		if _, dup := indices[idx]; dup {
			t.Fatalf("duplicate Index %d", idx)
		}
		indices[idx] = b
	}
	if kinds["Plane"] != 4 { // 2 player + 2 AI
		t.Fatalf("expected 4 planes, got %d", kinds["Plane"])
	}
	if kinds["Vehicle"] != 3+1+2 { // trucks + jeep + flak
		t.Fatalf("expected 6 vehicles, got %d", kinds["Vehicle"])
	}
	if kinds["Block"] != 2 {
		t.Fatalf("expected 2 blocks, got %d", kinds["Block"])
	}
	if kinds["MCU_TR_MissionBegin"] != 1 || kinds["MCU_TR_MissionObjective"] != 1 || kinds["MCU_Counter"] != 1 {
		t.Fatalf("logic blocks missing: %v", kinds)
	}
	if kinds["MCU_TR_Subtitle"] != 3 || kinds["MCU_CheckZone"] != 1 {
		t.Fatalf("radio blocks: %v", kinds)
	}

	// Every object with LinkTrId must be paired with an entity pointing back.
	destroyEvents := 0
	for idx, b := range indices {
		if link, ok := b.fields["LinkTrId"]; ok && link != "0" {
			l, _ := strconv.Atoi(link)
			ent, ok := indices[l]
			if !ok || ent.kind != "MCU_TR_Entity" || ent.fields["MisObjID"] != strconv.Itoa(idx) {
				t.Fatalf("object %d (%s) not paired with entity %d", idx, b.kind, l)
			}
		}
		for _, f := range []string{"Targets", "Objects"} {
			for _, ref := range idList(b.fields[f]) {
				if _, ok := indices[ref]; !ok {
					t.Fatalf("%s %d references missing index %d in %s", b.kind, idx, ref, f)
				}
			}
		}
		if b.kind == "MCU_Waypoint" {
			for _, ref := range idList(b.fields["Objects"]) {
				if indices[ref].kind != "MCU_TR_Entity" {
					t.Fatalf("waypoint %d Objects must reference entities", idx)
				}
			}
		}
		if b.kind == "MCU_Counter" && b.fields["Counter"] != "4" {
			t.Fatalf("counter should be 4 (trucks+jeep), got %s", b.fields["Counter"])
		}
	}
	// destroyed events: count in raw text
	destroyEvents = strings.Count(text, "Type = 13;")
	if destroyEvents != 4 {
		t.Fatalf("expected 4 destroy events, got %d", destroyEvents)
	}
	if !strings.Contains(text, `HMap = "graphics\LANDSCAPE_Korea_sp\height.hini";`) {
		t.Fatal("september should map to spring/autumn landscape")
	}
	if !strings.Contains(text, "Date = 14.9.1950;") || !strings.Contains(text, "Time = 15:35:0;") {
		t.Fatal("date/time not rendered")
	}
	if !strings.Contains(text, "    601 : 2;") || !strings.Contains(text, "    501 : 1;") {
		t.Fatal("countries block incomplete")
	}

	// language files: UTF-16LE with BOM, index:text lines
	ger, err := os.ReadFile(filepath.Join(dir, res.Name+".ger"))
	if err != nil {
		t.Fatal(err)
	}
	if ger[0] != 0xFF || ger[1] != 0xFE {
		t.Fatal("missing UTF-16LE BOM")
	}
	u := make([]uint16, 0, len(ger)/2)
	for i := 2; i+1 < len(ger); i += 2 {
		u = append(u, uint16(ger[i])|uint16(ger[i+1])<<8)
	}
	decoded := string(utf16.Decode(u))
	if !strings.HasPrefix(decoded, "0:Operation Chromite - Jagd auf die Spione\r\n1:Briefing DE\r\n2:Airborne\r\n") {
		t.Fatalf("unexpected language file start: %q", decoded[:80])
	}
	if !strings.Contains(decoded, "Vernichtet die Aufklärer") {
		t.Fatal("umlaut text missing")
	}
	for _, ext := range []string{".eng", ".list", ".rus"} {
		if _, err := os.Stat(filepath.Join(dir, res.Name+ext)); err != nil {
			t.Fatalf("%s missing", ext)
		}
	}
}

func TestResolveAssets(t *testing.T) {
	cases := map[string][2]string{
		"vehicles/studebakerus6":                  {`LuaScripts\WorldObjects\vehicles\studebakerus6.txt`, `graphics\vehicles\studebakerus6\studebakerus6.mgm`},
		"fixedobjects/boforsl60":                  {`LuaScripts\WorldObjects\fixedobjects\boforsl60.txt`, `graphics\fixedobjects\boforsl60\boforsl60.mgm`},
		"blocks/mil_camonet":                      {`LuaScripts\WorldObjects\Blocks\Mil_camonet.txt`, `graphics\blocks\mil_camonet.mgm`},
		`LuaScripts\WorldObjects\Planes\il10.txt`: {`LuaScripts\WorldObjects\Planes\il10.txt`, `graphics\planes\il10\il10.mgm`},
	}
	for in, want := range cases {
		got := resolve(in, "planes")
		if got.script != want[0] || got.model != want[1] {
			t.Errorf("%s -> %+v, want %v", in, got, want)
		}
	}
	if got := resolve("il10", "planes"); got.script != `LuaScripts\WorldObjects\Planes\il10.txt` {
		t.Errorf("bare plane name: %+v", got)
	}
}
