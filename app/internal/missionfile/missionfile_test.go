package missionfile

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"airborne/internal/dcs"
	"airborne/internal/il2"
	"airborne/internal/plan"
)

func testPlan(game string) *plan.MissionPlan {
	mp := &plan.MissionPlan{
		Game:  game,
		Title: plan.Localized{De: "Wadi-Schlag", En: "Wadi Strike"},
		Date:  "1982-06-10", Time: "06:40:00",
		Weather: plan.Weather{CloudLevel: 1500, CloudHeight: 6000, CloudConfig: `summer\00_Clear_00\sky.ini`,
			WindLayers: [][3]int{{0, 150, 3}}},
		Objectives: []plan.Objective{{Title: plan.Localized{De: "Kolonne", En: "Column"}, Counter: 2}},
		Briefing:   plan.Localized{De: "Zeile 1\nZeile \"2\"", En: "Line 1\nLine 2"},
		RadioQueue: []plan.Radio{{Trigger: "mission_begin_delay", Delay: 5, Speaker: "Basis", TextDe: "Hallo", TextEn: "Hello"}},
	}
	if game == "dcs" {
		mp.Map = "Syria"
		mp.PlayerGroups = []plan.Group{{Name: "Rakete", Aircraft: "MiG-21Bis", Count: 1, Country: "Syria",
			Start: &plan.Start{Type: "air", Lat: 33.25, Lon: 36.0, Alt: 2500}}}
		mp.EnemyGroups = []plan.Group{{Name: "APC", Kind: "vehicle", Script: "M-113", Count: 2, Country: "Israel",
			Position: &plan.Position{Lat: 33.1, Lon: 35.72}}}
	} else {
		mp.Map = "korea"
		mp.PlayerGroups = []plan.Group{{Name: "Sowol", Aircraft: "il10", Count: 1, Country: 501,
			Start: &plan.Start{Type: "air", X: 86500, Z: 244000, Alt: 1000}}}
		mp.EnemyGroups = []plan.Group{{Name: "Trucks", Kind: "vehicle", Script: "vehicles/studebakerus6", Count: 2, Country: 601,
			Position: &plan.Position{X: 99870, Z: 261830}}}
	}
	return mp
}

func writePNG(t *testing.T, p string) {
	t.Helper()
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 2)))
}

func TestDCSRoundtrip(t *testing.T) {
	dir := t.TempDir()
	res, err := dcs.Generate(testPlan("dcs"), dir)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Open(res.MainFile)
	if err != nil {
		t.Fatal(err)
	}
	doc := m.Doc()
	if doc.Game != "dcs" || doc.Protected {
		t.Fatalf("doc: %+v", doc)
	}
	if doc.Meta.Title.De != "Wadi-Schlag" || doc.Meta.Date != "1982-06-10" || doc.Meta.Time != "06:40:00" {
		t.Fatalf("meta: %+v", doc.Meta)
	}
	if !strings.Contains(doc.Meta.Briefing.De, "Zeile \"2\"") || !strings.Contains(doc.Meta.Briefing.De, "\n") {
		t.Fatalf("briefing decode: %q", doc.Meta.Briefing.De)
	}

	// edit metadata, attach picture + kneeboards
	meta := doc.Meta
	meta.Title.De = "Neuer Titel"
	meta.Briefing.De = "Erste Zeile\nZweite \"Zeile\" mit \\ Backslash"
	meta.Date = "1973-10-06"
	meta.Time = "14:05:00"
	if err := m.SetMeta(meta); err != nil {
		t.Fatal(err)
	}
	pic := filepath.Join(dir, "Lage.png")
	kb := filepath.Join(dir, "Funk.png")
	writePNG(t, pic)
	writePNG(t, kb)
	if err := m.SetBriefingImage(pic); err != nil {
		t.Fatal(err)
	}
	if err := m.AddKneeboard(kb); err != nil {
		t.Fatal(err)
	}
	if err := m.AddBriefingText("Titel", "Text"); err != nil {
		t.Fatal(err)
	}
	// verbatim text edit of the options file
	opts, err := m.Read("options")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Write("options", strings.Replace(opts, `["labels"] = 0`, `["labels"] = 1`, 1)); err != nil {
		t.Fatal(err)
	}
	if !m.Doc().Dirty {
		t.Fatal("must be dirty")
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.MainFile + ".bak"); err != nil {
		t.Fatal("backup missing")
	}

	m2, err := Open(res.MainFile)
	if err != nil {
		t.Fatal(err)
	}
	d2 := m2.Doc()
	if d2.Meta.Title.De != "Neuer Titel" || d2.Meta.Date != "1973-10-06" || d2.Meta.Time != "14:05:00" {
		t.Fatalf("meta after save: %+v", d2.Meta)
	}
	if d2.Meta.Briefing.De != meta.Briefing.De {
		t.Fatalf("briefing after save: %q", d2.Meta.Briefing.De)
	}
	if len(d2.Kneeboards) != 2 || d2.Kneeboards[0] != "KNEEBOARD/IMAGES/01_Funk.png" || d2.Kneeboards[1] != "KNEEBOARD/IMAGES/02_text.png" {
		t.Fatalf("kneeboards: %v", d2.Kneeboards)
	}
	if len(d2.BriefingImages) != 1 || d2.BriefingImages[0] != "l10n/DEFAULT/Lage.png" {
		t.Fatalf("briefing images: %v", d2.BriefingImages)
	}
	mission, _ := m2.Read("mission")
	for _, field := range []string{"pictureFileNameB", "pictureFileNameR"} {
		if vals := luaArrayValues(mission, field); len(vals) != 1 || vals[0] != "ResKey_ImageBriefing_Lage" {
			t.Fatalf("%s: %v", field, vals)
		}
	}
	if o, _ := m2.Read("options"); !strings.Contains(o, `["labels"] = 1`) {
		t.Fatal("verbatim edit lost")
	}
	// the rest of the mission is untouched: player group still there
	if !strings.Contains(mission, `"MiG-21Bis"`) {
		t.Fatal("units lost")
	}

	// remove the picture: references disappear too
	if err := m2.Remove("l10n/DEFAULT/Lage.png"); err != nil {
		t.Fatal(err)
	}
	if imgs := m2.Doc().BriefingImages; len(imgs) != 0 {
		t.Fatalf("picture still referenced: %v", imgs)
	}
	if err := m2.Remove("mission"); err == nil {
		t.Fatal("core file must be protected")
	}
}

func TestIL2Roundtrip(t *testing.T) {
	dir := t.TempDir()
	res, err := il2.Generate(testPlan("il2"), dir)
	if err != nil {
		t.Fatal(err)
	}
	// a stale binary mirror next to the mission
	os.WriteFile(strings.TrimSuffix(res.MainFile, ".Mission")+".msnbin", []byte{1, 2, 3}, 0o644)

	m, err := Open(res.MainFile)
	if err != nil {
		t.Fatal(err)
	}
	doc := m.Doc()
	if doc.Game != "il2" || doc.Meta.Title.De != "Wadi-Schlag" || doc.Meta.Title.En != "Wadi Strike" {
		t.Fatalf("meta: %+v", doc.Meta)
	}
	if doc.Meta.Briefing.De != "Zeile 1\nZeile \"2\"" || doc.Meta.Date != "1982-06-10" || doc.Meta.Time != "06:40:00" {
		t.Fatalf("meta: %+v", doc.Meta)
	}
	meta := doc.Meta
	meta.Title.En = "New Title"
	meta.Briefing.De = "Absatz 1\n\nAbsatz 2"
	meta.Author = "Vince"
	meta.Date = "1951-01-02"
	meta.Time = "07:05:00"
	if err := m.SetMeta(meta); err != nil {
		t.Fatal(err)
	}
	pic := filepath.Join(dir, "karte.png")
	writePNG(t, pic)
	if err := m.SetBriefingImage(pic); err != nil {
		t.Fatal(err)
	}
	if err := m.AddKneeboard(pic); err == nil {
		t.Fatal("kneeboards are DCS only")
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSuffix(res.MainFile, ".Mission")
	if _, err := os.Stat(base + ".msnbin"); err == nil {
		t.Fatal("stale msnbin must be removed on save")
	}
	if _, err := os.Stat(base + ".png"); err != nil {
		t.Fatal("briefing png missing")
	}
	m2, err := Open(res.MainFile)
	if err != nil {
		t.Fatal(err)
	}
	d2 := m2.Doc()
	if d2.Meta.Title.En != "New Title" || d2.Meta.Briefing.De != "Absatz 1\n\nAbsatz 2" || d2.Meta.Author != "Vince" {
		t.Fatalf("meta after save: %+v", d2.Meta)
	}
	if d2.Meta.Date != "1951-01-02" || d2.Meta.Time != "07:05:00" {
		t.Fatalf("date/time after save: %+v", d2.Meta)
	}
	if len(d2.BriefingImages) != 1 {
		t.Fatalf("briefing images: %v", d2.BriefingImages)
	}
	ger, _ := m2.Read(filepath.Base(base) + ".ger")
	if !strings.HasPrefix(ger, "0:Wadi-Schlag\r\n1:Absatz 1<br><br>Absatz 2\r\n2:Vince\r\n") {
		t.Fatalf("ger: %q", ger)
	}
	mission, _ := m2.Read(filepath.Base(base) + ".Mission")
	if !strings.Contains(mission, "Date = 2.1.1951;") || !strings.Contains(mission, "Time = 7:5:0;") {
		t.Fatal("options date/time not rewritten")
	}

	// Save as: sibling files follow the new base name
	other := filepath.Join(dir, "sub", "Kopie.Mission")
	if err := m2.SaveAs(other); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".Mission", ".ger", ".eng", ".list", ".png"} {
		if _, err := os.Stat(filepath.Join(dir, "sub", "Kopie"+ext)); err != nil {
			t.Fatalf("Kopie%s missing", ext)
		}
	}
}

func TestLuaHelpers(t *testing.T) {
	text := "dictionary = \n{\n\t[\"DictKey_a\"] = \"x\\\ny \\\"q\\\"\",\n} -- end of dictionary\n"
	got := parseLuaStringTable(text)
	if got["DictKey_a"] != "x\ny \"q\"" {
		t.Fatalf("parse: %q", got["DictKey_a"])
	}
	text = setLuaString(text, "DictKey_a", "neu $1")
	text = setLuaString(text, "DictKey_b", "b")
	got = parseLuaStringTable(text)
	if got["DictKey_a"] != "neu $1" || got["DictKey_b"] != "b" {
		t.Fatalf("set: %v", got)
	}
	mission := "mission = \n{\n\t[\"pictureFileNameB\"] = {},\n\t[\"x\"] = 1,\n}"
	mission, err := appendLuaArrayValue(mission, "pictureFileNameB", "ResKey_1")
	if err != nil {
		t.Fatal(err)
	}
	mission, _ = appendLuaArrayValue(mission, "pictureFileNameB", "ResKey_2")
	mission, _ = appendLuaArrayValue(mission, "pictureFileNameB", "ResKey_2")
	if vals := luaArrayValues(mission, "pictureFileNameB"); len(vals) != 2 || vals[1] != "ResKey_2" {
		t.Fatalf("array: %v\n%s", vals, mission)
	}
	if !strings.Contains(mission, "[\"x\"] = 1") {
		t.Fatal("neighbour field damaged")
	}
}
