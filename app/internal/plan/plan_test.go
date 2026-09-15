package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func challengePlan() *MissionPlan {
	return &MissionPlan{
		Game: "dcs", Title: Localized{De: "Blind"}, Map: "Syria", Date: "1982-06-10", Time: "06:40:00",
		PlayerGroups: []Group{{Aircraft: "MiG-21Bis", Start: &Start{Lat: 33, Lon: 36}}},
		EnemyGroups: []Group{
			{Kind: "plane", Aircraft: "F-5E-3", Position: &Position{Lat: 33, Lon: 35.6}},
			{Kind: "vehicle", Script: "vehicles/M-113", Position: &Position{Lat: 33.1, Lon: 35.7}},
		},
		Flak:       []Flak{{Script: "ZSU-23-4 Shilka", Position: Position{Lat: 33.1, Lon: 35.7}}},
		Objectives: []Objective{{Title: Localized{De: "Patrouille"}, Counter: 2}},
		Briefing:   Localized{De: "Aufklaerung meldet Aktivitaet im Raum Golan.", En: "Recon reports activity."},
		Challenge:  true,
	}
}

// A clean challenge plan validates without issues; naming an enemy type in
// any player-facing text (or leaving icons) is reported.
func TestChallengeLeaks(t *testing.T) {
	mp := challengePlan()
	if issues := mp.Validate(); len(issues) != 0 {
		t.Fatalf("clean plan reported: %v", issues)
	}
	terms := mp.EnemyTerms()
	if strings.Join(terms, ",") != "F-5E-3,M-113,ZSU-23-4 Shilka" {
		t.Fatalf("enemy terms: %v", terms)
	}

	mp.Briefing.En = "Expect two f-5e-3 fighters."
	mp.RadioQueue = []Radio{{Trigger: "check_zone", TextDe: "Shilka-Stellung voraus", TextEn: "ZSU-23-4 Shilka ahead"}}
	mp.Icons = []Icon{{}}
	issues := mp.Validate()
	want := []string{"briefing.en verraet den Gegner (\"F-5E-3\")", "radioQueue[0] verraet den Gegner (\"ZSU-23-4 Shilka\")", "icons werden nicht exportiert"}
	for _, w := range want {
		found := false
		for _, i := range issues {
			if strings.Contains(i, w) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing issue %q in %v", w, issues)
		}
	}
	mp.Challenge = false
	for _, i := range mp.Validate() {
		if strings.Contains(i, "Herausforderung") {
			t.Errorf("non-challenge plan must not check leaks: %s", i)
		}
	}
}

func TestMediaValidation(t *testing.T) {
	dir := t.TempDir()
	png := filepath.Join(dir, "a.png")
	os.WriteFile(png, []byte("x"), 0o644)
	mp := challengePlan()
	mp.Challenge = false
	mp.Media = &Media{BriefingImage: png, Kneeboards: []string{filepath.Join(dir, "missing.png"), filepath.Join(dir, "a.png")}}
	issues := mp.Validate()
	if len(issues) != 1 || !strings.Contains(issues[0], "media.kneeboards[0]") {
		t.Fatalf("issues: %v", issues)
	}
	mp.Game = "il2"
	mp.PlayerGroups[0].Start = &Start{X: 1, Z: 1}
	found := false
	for _, i := range mp.Validate() {
		if strings.Contains(i, "Kneeboards gibt es nur in DCS") {
			found = true
		}
	}
	if !found {
		t.Fatal("il2 plan with kneeboards must warn")
	}
	if !(&Media{}).IsEmpty() || (*Media)(nil).IsEmpty() != true || mp.Media.IsEmpty() {
		t.Fatal("IsEmpty")
	}
}
