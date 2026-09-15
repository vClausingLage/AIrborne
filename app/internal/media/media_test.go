package media

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderTextPage(t *testing.T) {
	long := ""
	for i := 0; i < 40; i++ {
		long += "Aufklärung meldet Aktivität im Raum nordwestlich von Inchon. "
	}
	data, err := RenderTextPage("Briefing – Operation Chromite", long+"\n\nRückkehr nach Kimpo.")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	if cfg.Width != PageWidth || cfg.Height != PageHeight {
		t.Errorf("size %dx%d", cfg.Width, cfg.Height)
	}
}

func TestLoadPNGConvertsJPEG(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	jp := filepath.Join(dir, "Bild ä.jpg")
	f, _ := os.Create(jp)
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatal(err)
	}
	f.Close()
	data, err := LoadPNG(jp)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
		t.Fatalf("jpeg was not converted to png: %v", err)
	}
	if got := SafeName(jp); got != "Bild_ae" {
		t.Errorf("SafeName = %q", got)
	}

	pp := filepath.Join(dir, "x.png")
	f, _ = os.Create(pp)
	png.Encode(f, img)
	f.Close()
	if _, err := LoadPNG(pp); err != nil {
		t.Errorf("png must load: %v", err)
	}
	if _, err := LoadPNG(filepath.Join(dir, "missing.png")); err == nil {
		t.Error("missing file must fail")
	}
	bad := filepath.Join(dir, "bad.png")
	os.WriteFile(bad, []byte("not a png"), 0o644)
	if _, err := LoadPNG(bad); err == nil {
		t.Error("invalid png must fail")
	}
}
