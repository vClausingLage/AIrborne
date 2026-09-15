package dcs

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

const sampleUnitPayloads = `local unitPayloads = {
	["name"] = "MiG-21Bis",
	["payloads"] = {
		[1] = {
			["name"] = "Patrol, long range",
			["pylons"] = {
				[1] = { ["CLSID"] = "{PTB_800_MIG21}", ["num"] = 3 },
				[2] = { ["CLSID"] = "{R-3R}", ["num"] = 2 },
				[3] = { ["CLSID"] = "{R-3S}", ["num"] = 1 },
			},
			["tasks"] = { [1] = 11 },
		},
		[2] = {
			["name"] = "R-60M*4, PTB-490",
			["pylons"] = {
				[1] = { ["CLSID"] = "{R-60M 2L}", ["num"] = 1 },
				[2] = { ["CLSID"] = "{R-60M 2R}", ["num"] = 5 },
			},
			["tasks"] = { [1] = 11, [2] = 19 },
		},
		[3] = {
			["name"] = "FAB-250*2, S-5 UB-16*2",
			["pylons"] = {
				[1] = { ["CLSID"] = "{FAB-250}", ["num"] = 1 },
				[2] = { ["CLSID"] = "{UB-16-57UMP}", ["num"] = 2 },
			},
			["tasks"] = { [1] = 31, [2] = 32 },
		},
	},
	["unitType"] = "MiG-21Bis",
}
return unitPayloads
`

func TestParsePayloadFile(t *testing.T) {
	typ, presets := parsePayloadFile(sampleUnitPayloads)
	if typ != "MiG-21Bis" || len(presets) != 3 {
		t.Fatalf("type=%q presets=%d", typ, len(presets))
	}
	p := presets[0]
	if p.Name != "Patrol, long range" || p.Pylons[3] != "{PTB_800_MIG21}" || p.Pylons[1] != "{R-3S}" || len(p.Tasks) != 1 || p.Tasks[0] != 11 {
		t.Fatalf("preset 1 parsed wrong: %+v", p)
	}
	if got := presets[1].Tasks; len(got) != 2 || got[1] != 19 {
		t.Fatalf("tasks of preset 2: %v", got)
	}
}

func TestPickPreset(t *testing.T) {
	typ, presets := parsePayloadFile(sampleUnitPayloads)
	db := &PayloadDB{byType: map[string][]Preset{typ: presets}, Files: 1}
	cap, _ := lookupRole("CAP")
	cas, _ := lookupRole("cas")
	sead, _ := lookupRole("SEAD")

	if p, ok := db.Pick("MiG-21Bis", cap, ""); !ok || p.Name != "Patrol, long range" {
		t.Fatalf("CAP default: %+v %v", p, ok)
	}
	if p, _ := db.Pick("MiG-21Bis", cap, "4x R-60M IR-Lenkwaffen"); p.Name != "R-60M*4, PTB-490" {
		t.Fatalf("CAP with wish: %q", p.Name)
	}
	if p, _ := db.Pick("mig-21bis", cas, "2x FAB-250"); p.Name != "FAB-250*2, S-5 UB-16*2" {
		t.Fatalf("CAS (case-insensitive type): %q", p.Name)
	}
	// no SEAD preset: chain falls back to strike/ground-attack presets
	if p, _ := db.Pick("MiG-21Bis", sead, ""); p.Name != "FAB-250*2, S-5 UB-16*2" {
		t.Fatalf("SEAD fallback: %q", p.Name)
	}
	if _, ok := db.Pick("F-16C_50", cap, ""); ok {
		t.Fatal("unknown type must report ok=false")
	}
	if _, ok := lookupRole("Bomber-Eskorte"); ok {
		t.Fatal("unknown role must not resolve")
	}
	if r, ok := lookupRole("Anti-Ship"); !ok || r.key != "AntiShip" {
		t.Fatalf("alias: %+v %v", r, ok)
	}
}

func TestPylonsInMiz(t *testing.T) {
	typ, presets := parsePayloadFile(sampleUnitPayloads)
	db := &PayloadDB{byType: map[string][]Preset{typ: presets}, Files: 1}
	mp := samplePlan()
	mp.PlayerGroups[0].Task = "CAS"
	mp.PlayerGroups[0].Payload = "2x FAB-250, 2x UB-16"
	mp.EnemyGroups[2].Task = "Intercept"
	res, err := GenerateOpts(mp, t.TempDir(), nil, db)
	if err != nil {
		t.Fatal(err)
	}
	mission := readZipEntry(t, res.MainFile, "mission")
	if !strings.Contains(mission, `["CLSID"] = "{FAB-250}"`) || !strings.Contains(mission, `["CLSID"] = "{UB-16-57UMP}"`) {
		t.Fatal("player pylons not written")
	}
	if !strings.Contains(mission, `["task"] = "Intercept"`) {
		t.Fatal("enemy task not written")
	}
	notes := strings.Join(res.Notes, "\n")
	if !strings.Contains(notes, "Bewaffnung Rakete (MiG-21Bis, CAS): FAB-250*2") {
		t.Fatalf("missing payload note:\n%s", notes)
	}
	if !strings.Contains(notes, "keine Presets fuer Typ \"F-5E-3\"") {
		t.Fatalf("missing no-preset note:\n%s", notes)
	}
}

// TestRealPayloadDB reads the presets of the local DCS installation (skipped
// when DCS_ROOT is not set or the folder is missing).
func TestRealPayloadDB(t *testing.T) {
	root := strings.Trim(os.Getenv("DCS_ROOT"), `"`)
	if root == "" {
		root = `C:\Program Files\Eagle Dynamics\DCS World`
	}
	if _, err := os.Stat(root); err != nil {
		t.Skip("DCS not installed:", root)
	}
	db := LoadPayloadDB(root, "")
	if db.Files < 20 {
		t.Fatalf("only %d preset files found under %s", db.Files, root)
	}
	cap, _ := lookupRole("CAP")
	for _, typ := range []string{"MiG-21Bis", "Mi-24P", "F-16C_50", "Mi-8MT", "F-5E-3"} {
		p, ok := db.Pick(typ, cap, "")
		if !ok || len(p.Pylons) == 0 {
			t.Errorf("%s: no preset picked (%v)", typ, ok)
			continue
		}
		t.Logf("%s CAP -> %s (%d pylons)", typ, p.Name, len(p.Pylons))
	}
	sead, _ := lookupRole("SEAD")
	if p, _ := db.Pick("F-16C_50", sead, "AGM-88"); !strings.Contains(p.Name, "AGM-88") {
		t.Errorf("F-16 SEAD picked %q", p.Name)
	}
	cas, _ := lookupRole("CAS")
	if p, _ := db.Pick("Mi-24P", cas, "S-8KOM und 9M114"); !strings.Contains(p.Name, "S-8KOM") {
		t.Errorf("Mi-24P CAS picked %q", p.Name)
	}
}

func readZipEntry(t *testing.T, path, name string) string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name == name {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			return string(data)
		}
	}
	t.Fatalf("zip entry %s missing", name)
	return ""
}
