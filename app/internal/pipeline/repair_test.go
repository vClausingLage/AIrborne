package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"airborne/internal/ai"
	"airborne/internal/plan"
)

// testPipeline builds a pipeline on a scratch root with the real prompt
// templates and no project file.
func testPipeline(t *testing.T) *Pipeline {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	src, _ := filepath.Abs(filepath.Join("..", "..", "..", "prompts"))
	for _, f := range []string{systemFile, mergeFile, quickFile, prefabFile, challengeFile, repairFile, schemaFile} {
		data, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(root, "prompts", f), data, 0o644)
	}
	p, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A minimal DCS plan the generator accepts without complaints.
func cleanDCSPlan() plan.MissionPlan {
	return plan.MissionPlan{
		Game: "dcs", Title: plan.Localized{De: "Test", En: "Test"}, Map: "Syria",
		Date: "1982-06-10", Time: "06:40:00",
		PlayerGroups: []plan.Group{{Name: "Rakete", Aircraft: "MiG-21Bis", Count: 1, Country: "Syria", Task: "CAS",
			Start: &plan.Start{Type: "air", Lat: 33.25, Lon: 36.0, Alt: 2500, Heading: 245},
			Route: []plan.Position{{Lat: 33.1, Lon: 35.72, Alt: 1500}}}},
		EnemyGroups: []plan.Group{{Name: "APC", Kind: "vehicle", Script: "M-113", Count: 2, Country: "Israel",
			Position: &plan.Position{Lat: 33.1, Lon: 35.72, Head: 90}, Movement: &plan.Movement{Type: "static"}}},
		Objectives: []plan.Objective{{Title: plan.Localized{De: "Ziel", En: "Target"}, Counter: 2}},
		RadioQueue: []plan.Radio{{Trigger: "mission_begin_delay", Delay: 10, Speaker: "Basis", TextDe: "Los.", TextEn: "Go."}},
		Briefing:   plan.Localized{De: "Briefing", En: "Briefing"},
	}
}

func TestCheckPlanFindsValidateAndGeneratorProblems(t *testing.T) {
	p := testPipeline(t)
	mp := cleanDCSPlan()
	if got := p.checkPlan(&mp); len(got) != 0 {
		t.Fatalf("clean plan should pass, got %v", got)
	}
	mp.Briefing = plan.Localized{}           // Validate: briefing missing
	mp.EnemyGroups[0].Country = "Atlantis"   // generator: unknown country
	mp.PlayerGroups[0].Start.Type = "runway" // generator: unsupported start
	got := strings.Join(p.checkPlan(&mp), "\n")
	for _, want := range []string{"briefing (de/en) fehlt", `Country "Atlantis" unbekannt`, `Startart "runway"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	mp.Map = "Mars"
	if got := strings.Join(p.checkPlan(&mp), "\n"); !strings.Contains(got, "Generator: DCS-Karte") {
		t.Errorf("generator error should be reported as a problem:\n%s", got)
	}
}

// The quick mission flow: the first answer has a defect, the repair prompt
// carries the problem list and the plan, and the corrected answer replaces
// the first one.
func TestQuickMissionRepairsPlan(t *testing.T) {
	p := testPipeline(t)
	p.SetGame("dcs")
	broken := cleanDCSPlan()
	broken.EnemyGroups[0].Country = "Atlantis"
	fixed := cleanDCSPlan()
	calls := 0
	var repairPrompt string
	p.chatFn = func(_ context.Context, msgs []ai.Message) (string, ai.Usage, error) {
		calls++
		mp := broken
		if calls == 2 {
			repairPrompt = msgs[len(msgs)-1].Content
			mp = fixed
		}
		data, _ := json.Marshal(mp)
		return string(data), ai.Usage{TotalTokens: 10}, nil
	}
	mp, err := p.QuickMission(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected plan + repair call, got %d calls", calls)
	}
	if !strings.Contains(repairPrompt, `Country "Atlantis" unbekannt`) || !strings.Contains(repairPrompt, `"Atlantis"`) {
		t.Fatalf("repair prompt lacks problem list or plan JSON:\n%s", repairPrompt[:400])
	}
	if !strings.Contains(repairPrompt, "# Modus: Reparatur") {
		t.Fatal("repair prompt should use 10-repair.md")
	}
	if plan.CountryString(mp.EnemyGroups[0].Country) != "Israel" {
		t.Fatalf("repaired plan not taken: %v", mp.EnemyGroups[0].Country)
	}
	st := p.State()
	if st.Repair == nil || !st.Repair.Repaired || len(st.Repair.Found) != 1 || len(st.Repair.Remaining) != 0 {
		t.Fatalf("repair info: %+v", st.Repair)
	}
	if st.Config.LastUsage == nil || st.Config.LastUsage.TotalTokens != 10 {
		t.Fatalf("usage not recorded: %+v", st.Config.LastUsage)
	}
}

// A repair answer that is worse than the original is rejected; a failing
// repair call keeps the original plan and does not fail the generation.
func TestRepairKeepsOriginalOnWorseOrFailedAnswer(t *testing.T) {
	broken := cleanDCSPlan()
	broken.EnemyGroups[0].Country = "Atlantis"
	worse := broken
	worse.Map = "Mars"

	p := testPipeline(t)
	p.SetGame("dcs")
	calls := 0
	p.chatFn = func(context.Context, []ai.Message) (string, ai.Usage, error) {
		calls++
		mp := broken
		if calls == 2 {
			mp = worse
		}
		data, _ := json.Marshal(mp)
		return string(data), ai.Usage{}, nil
	}
	mp, err := p.QuickMission(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if mp.Map != "Syria" {
		t.Fatal("worse repair must be rejected")
	}
	if r := p.State().Repair; r == nil || r.Repaired || !strings.Contains(r.Note, "verworfen") || len(r.Remaining) != 1 {
		t.Fatalf("repair info: %+v", r)
	}

	p2 := testPipeline(t)
	p2.SetGame("dcs")
	calls = 0
	p2.chatFn = func(context.Context, []ai.Message) (string, ai.Usage, error) {
		calls++
		if calls == 2 {
			return "not json at all", ai.Usage{}, nil
		}
		data, _ := json.Marshal(broken)
		return string(data), ai.Usage{}, nil
	}
	if _, err := p2.QuickMission(context.Background(), "test"); err != nil {
		t.Fatalf("a failed repair must not fail the generation: %v", err)
	}
	if r := p2.State().Repair; r == nil || r.Repaired || !strings.Contains(r.Note, "fehlgeschlagen") {
		t.Fatalf("repair info: %+v", r)
	}
}

// A clean first answer costs exactly one call and leaves no repair record.
func TestNoRepairWhenClean(t *testing.T) {
	p := testPipeline(t)
	p.SetGame("dcs")
	calls := 0
	p.chatFn = func(context.Context, []ai.Message) (string, ai.Usage, error) {
		calls++
		data, _ := json.Marshal(cleanDCSPlan())
		return string(data), ai.Usage{}, nil
	}
	if _, err := p.QuickMission(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || p.State().Repair != nil {
		t.Fatalf("calls=%d repair=%+v", calls, p.State().Repair)
	}
}
