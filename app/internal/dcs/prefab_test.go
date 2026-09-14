package dcs

import (
	"archive/zip"
	"io"
	"math"
	"testing"

	"airborne/internal/plan"
	"airborne/internal/prefab"
)

type memLib map[string]prefab.Prefab

func (m memLib) Find(id string) (*prefab.Prefab, bool) {
	p, ok := m[id]
	return &p, ok
}

func carrierPrefab() prefab.Prefab {
	return prefab.Prefab{
		ID: "carrier-group", Name: "Carrier Group", Game: "dcs", Country: "USA",
		Elements: []prefab.Element{
			{Name: "Carrier", Kind: "ship", Type: "CVN_73", Group: "CSG", DX: 0, DY: 0, Heading: 0},
			{Name: "Escort", Kind: "ship", Type: "USS_Arleigh_Burke_IIa", Group: "CSG", Count: 2, Spacing: 1500, DX: -1500, DY: 0},
			{Name: "Hornets", Kind: "plane", Type: "FA-18C_hornet", Count: 3, Spacing: 15, DX: -80, DY: 20, Heading: 90, LinkTo: "Carrier"},
			{Name: "Tender", Kind: "vehicle", Type: "Ural-375", DX: 500, DY: 500},
			{Name: "Depot", Kind: "static", Type: "warehouse", DX: 600, DY: 500},
			{Name: "Crate", Kind: "static", Type: "iso_container", DX: 620, DY: 500},
		},
	}
}

func TestPrefabExpansion(t *testing.T) {
	mp := samplePlan()
	mp.Prefabs = []plan.PrefabPlacement{{
		Prefab: "carrier-group", Side: "enemy", Position: plan.Position{Lat: 33.5, Lon: 34.5, Head: 90},
	}}
	dir := t.TempDir()
	res, err := GenerateWith(mp, dir, memLib{"carrier-group": carrierPrefab()})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range res.Notes {
		t.Log("note:", n)
	}
	zr, err := zip.OpenReader(res.MainFile)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var text string
	for _, f := range zr.File {
		if f.Name == "mission" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			text = string(data)
		}
	}
	m := parseLua(t, "mission", text)

	// enemy side is blue (player is Syria); prefab country USA joins Israel there
	blue := get(t, m, "coalition", "blue", "country").(map[string]any)
	var usa map[string]any
	for _, c := range blue {
		if cm := c.(map[string]any); cm["name"] == "USA" {
			usa = cm
		}
	}
	if usa == nil {
		t.Fatalf("USA missing on blue side: %v", blue)
	}
	ships := get(t, usa, "ship", "group").(map[string]any)
	if len(ships) != 1 {
		t.Fatalf("expected one ship group, got %d", len(ships))
	}
	csg := ships["1"].(map[string]any)
	units := csg["units"].(map[string]any)
	if len(units) != 3 {
		t.Fatalf("CSG units = %d", len(units))
	}
	carrier := units["1"].(map[string]any)
	if carrier["type"] != "CVN_73" || carrier["frequency"] != 127500.0 {
		t.Fatalf("carrier unit: %v", carrier)
	}
	cx, cy := carrier["x"].(float64), carrier["y"].(float64)
	// placement heading 90: local "north" (dx) maps to +y (east); the escort at dx=-1500 sits west
	esc := units["2"].(map[string]any)
	if math.Abs(esc["y"].(float64)-(cy-1500)) > 1 || math.Abs(esc["x"].(float64)-cx) > 1 {
		t.Fatalf("escort rotation wrong: carrier (%.0f,%.0f) escort (%.0f,%.0f)", cx, cy, esc["x"], esc["y"])
	}
	if math.Abs(carrier["heading"].(float64)-math.Pi/2) > 1e-6 {
		t.Fatalf("carrier heading = %v", carrier["heading"])
	}

	statics := get(t, usa, "static", "group").(map[string]any)
	if len(statics) != 5 { // 3 hornets + depot + crate
		t.Fatalf("static groups = %d", len(statics))
	}
	linked, warehouse, cargo := 0, 0, 0
	for _, g := range statics {
		u := get(t, g, "units", "1").(map[string]any)
		switch u["type"] {
		case "FA-18C_hornet":
			if u["category"] != "Planes" || u["linkUnit"] != carrier["unitId"] || u["linkOffset"] != true {
				t.Fatalf("hornet not linked to carrier: %v", u)
			}
			off := u["offsets"].(map[string]any)
			// facing starboard (90°), the row runs along the ship axis: x = -80, -95, -110
			if math.Abs(off["y"].(float64)-20) > 1e-6 || off["x"].(float64) < -111 || off["x"].(float64) > -79 {
				t.Fatalf("hornet offsets: %v", off)
			}
			if math.Abs(off["angle"].(float64)-math.Pi/2) > 1e-6 {
				t.Fatalf("hornet angle: %v", off["angle"])
			}
			linked++
		case "Warehouse":
			if u["category"] != "Warehouses" || u["shape_name"] != "warehouse" {
				t.Fatalf("warehouse static: %v", u)
			}
			warehouse++
		case "iso_container":
			if u["category"] != "Cargos" || u["canCargo"] != true {
				t.Fatalf("cargo static: %v", u)
			}
			cargo++
		default:
			t.Fatalf("unexpected static %v", u["type"])
		}
	}
	if linked != 3 || warehouse != 1 || cargo != 1 {
		t.Fatalf("statics: hornets=%d warehouse=%d cargo=%d", linked, warehouse, cargo)
	}
	// the tender truck is a normal vehicle group
	if _, ok := get(t, usa, "vehicle", "group", "1", "units", "1").(map[string]any); !ok {
		t.Fatal("prefab vehicle group missing")
	}
}
