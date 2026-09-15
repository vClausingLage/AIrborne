package il2

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const sampleTable = `{
  "_doc": "ignored",
  "planes": {
    "IL10": {"name": "Il-10", "payloads": [
      {"id": 0, "name": "Kanonen", "roles": ["CAP"]},
      {"id": 1, "name": "4x FAB-100", "roles": ["CAS", "GroundAttack"]},
      {"id": 2, "name": "8x RS-82 + 4x FAB-100", "roles": ["CAS"], "modMask": "3"},
      {"id": 3, "name": "2x FAB-250", "roles": ["Strike"]}
    ]},
    "f51d": {"name": "F-51D", "payloads": [
      {"id": 0, "name": "6x .50 cal", "roles": ["CAP", "Escort"]},
      {"id": 2, "name": "6x HVAR", "roles": ["GroundAttack"]}
    ]}
  }
}`

func writeTable(t *testing.T) *PayloadTable {
	t.Helper()
	path := filepath.Join(t.TempDir(), "il2-payloads.json")
	if err := os.WriteFile(path, []byte(sampleTable), 0o644); err != nil {
		t.Fatal(err)
	}
	table, err := LoadPayloadTable(path)
	if err != nil {
		t.Fatal(err)
	}
	return table
}

func TestPayloadTablePick(t *testing.T) {
	table := writeTable(t)
	if !table.Known("il10") || !table.Known("F51D") || table.Known("mig15bis") {
		t.Fatalf("Known() wrong: %v", table.PlaneNames())
	}
	if p, _ := table.Pick("il10", "CAS", ""); p.ID != 1 {
		t.Fatalf("CAS default -> %d", p.ID)
	}
	if p, _ := table.Pick("il10", "CAS", "8x RS-82 Raketen und FAB-100"); p.ID != 2 || p.ModMask != "3" {
		t.Fatalf("CAS with wish -> %+v", p)
	}
	// Strike is a ground role: exact match wins over the family
	if p, _ := table.Pick("il10", "strike", ""); p.ID != 3 {
		t.Fatalf("Strike -> %d", p.ID)
	}
	// AntiShip has no exact entry: fall back to the ground family, best wish
	if p, _ := table.Pick("il10", "AntiShip", "FAB-250"); p.ID != 3 {
		t.Fatalf("AntiShip family fallback -> %d", p.ID)
	}
	if p, _ := table.Pick("f51d", "Escort", ""); p.ID != 0 {
		t.Fatalf("Escort -> %d", p.ID)
	}
	if _, ok := table.Pick("mig15bis", "CAP", ""); ok {
		t.Fatal("unknown plane must report ok=false")
	}
	// missing file is not an error
	empty, err := LoadPayloadTable(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil || len(empty.Planes) != 0 {
		t.Fatalf("missing table: %v %d", err, len(empty.Planes))
	}
}

func TestPayloadIdInMission(t *testing.T) {
	table := writeTable(t)
	mp := samplePlan()
	mp.PlayerGroups[0].Task = "CAS"
	mp.PlayerGroups[0].Payload = "8x RS-82, 4x FAB-100"
	mp.EnemyGroups[2].Task = "Escort"
	res, err := GenerateOpts(mp, t.TempDir(), table)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(res.MainFile)
	text := string(data)
	re := regexp.MustCompile(`Script = "LuaScripts\\WorldObjects\\Planes\\(\w+)\.txt";[\s\S]*?PayloadId = (\d+);\s*ModMask = (\w+);`)
	got := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		got[m[1]] = m[2] + "/" + m[3]
	}
	if got["il10"] != "2/3" {
		t.Fatalf("player il10 payload = %q (want 2/3)", got["il10"])
	}
	if got["f51d"] != "0/1" {
		t.Fatalf("enemy f51d payload = %q (want 0/1)", got["f51d"])
	}
	notes := strings.Join(res.Notes, "\n")
	if !strings.Contains(notes, "Bewaffnung Sowol (il10, CAS): #2 8x RS-82") || !strings.Contains(notes, "Bewaffnung Mustangs (f51d, Escort): #0") {
		t.Fatalf("notes:\n%s", notes)
	}
}

// TestProjectTable makes sure the shipped reference/il2-payloads.json parses.
func TestProjectTable(t *testing.T) {
	path, _ := filepath.Abs(filepath.Join("..", "..", "..", "reference", "il2-payloads.json"))
	table, err := LoadPayloadTable(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, plane := range []string{"f86a5", "mig15bis", "il10", "f51d"} {
		if _, ok := table.Planes[plane]; !ok {
			t.Errorf("plane %s missing from %s", plane, path)
		}
	}
}
