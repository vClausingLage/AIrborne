package prefab

import (
	"path/filepath"
	"testing"
)

func TestNormalizeAndLibrary(t *testing.T) {
	p := Prefab{Name: "FARP Alpha", Elements: []Element{
		{Kind: "static", Type: "farp"},
		{Kind: "heli", Type: "UH-1H", Count: 2},
		{Kind: "static", Type: "Unknown Thing"},
		{Kind: "plane", Type: "F-14B", LinkTo: "Nope"},
	}}
	issues := p.Normalize()
	if p.ID != "farp-alpha" || p.Game != "dcs" {
		t.Fatalf("id/game: %s %s", p.ID, p.Game)
	}
	if e := p.Elements[0]; e.Type != "FARP" || e.Category != "Heliports" || e.ShapeName != "FARPS" {
		t.Fatalf("catalog lookup failed: %+v", e)
	}
	if e := p.Elements[1]; e.Kind != "helicopter" || e.Category != "Helicopters" || !e.IsStatic() {
		t.Fatalf("heli normalize: %+v", e)
	}
	if p.Elements[2].Category != "" {
		t.Fatal("unknown static should have no category")
	}
	found := 0
	for _, is := range issues {
		t.Log(is)
		found++
	}
	if found != 2 { // unknown static + bad linkTo
		t.Fatalf("issues = %d", found)
	}

	lib := NewLibrary(filepath.Join(t.TempDir(), "prefabs"))
	p.Elements = p.Elements[:2]
	saved, err := lib.Save(p)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := lib.Find("farp alpha"); !ok || got.ID != saved.ID {
		t.Fatal("find by name failed")
	}
	list, _ := lib.List()
	if len(list) != 1 {
		t.Fatalf("list = %d", len(list))
	}
	if err := lib.Delete(saved.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := lib.Find(saved.ID); ok {
		t.Fatal("still found after delete")
	}
}
