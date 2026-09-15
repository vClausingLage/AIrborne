// Package media handles user-supplied mission content that never passes
// through the LLM: briefing pictures and kneeboard pages. It normalises
// images to PNG and can render briefing text into a kneeboard page.
package media

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	_ "image/jpeg" // register JPEG decoder

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Kneeboard page size (portrait 3:4, the DCS default kneeboard aspect).
const (
	PageWidth  = 1024
	PageHeight = 1366
)

// LoadPNG reads a PNG or JPG file and returns PNG bytes (JPGs are re-encoded).
func LoadPNG(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Bild nicht lesbar: %w", err)
	}
	if strings.EqualFold(filepath.Ext(path), ".png") {
		if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
			return nil, fmt.Errorf("%s ist kein gueltiges PNG: %w", filepath.Base(path), err)
		}
		return data, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s: Bildformat nicht unterstuetzt (PNG/JPG): %w", filepath.Base(path), err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9_\-]+`)

// SafeName turns a file name into an ASCII name usable inside archives.
func SafeName(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var b strings.Builder
	for _, r := range base {
		switch r {
		case 'ä', 'ö', 'ü', 'Ä', 'Ö', 'Ü', 'ß':
			b.WriteString(map[rune]string{'ä': "ae", 'ö': "oe", 'ü': "ue", 'Ä': "Ae", 'Ö': "Oe", 'Ü': "Ue", 'ß': "ss"}[r])
		default:
			if r > unicode.MaxASCII {
				b.WriteRune('_')
			} else {
				b.WriteRune(r)
			}
		}
	}
	s := strings.Trim(unsafeName.ReplaceAllString(b.String(), "_"), "_")
	if s == "" {
		s = "image"
	}
	return s
}

// RenderTextPage draws a title and body text onto a kneeboard-sized PNG
// (dark text on a light "paper" background, word-wrapped).
func RenderTextPage(title, body string) ([]byte, error) {
	regular, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	bold, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}
	bodyFace, err := opentype.NewFace(regular, &opentype.FaceOptions{Size: 30, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	titleFace, err := opentype.NewFace(bold, &opentype.FaceOptions{Size: 44, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, PageWidth, PageHeight))
	paper := color.RGBA{0xf2, 0xee, 0xe3, 0xff}
	ink := color.RGBA{0x1a, 0x1a, 0x1a, 0xff}
	rule := color.RGBA{0x8a, 0x84, 0x74, 0xff}
	draw.Draw(img, img.Bounds(), &image.Uniform{paper}, image.Point{}, draw.Src)

	const margin = 64
	maxWidth := PageWidth - 2*margin
	y := margin + 44

	d := &font.Drawer{Dst: img, Src: &image.Uniform{ink}}
	if strings.TrimSpace(title) != "" {
		d.Face = titleFace
		for _, line := range wrap(titleFace, strings.TrimSpace(title), maxWidth) {
			d.Dot = fixed.P(margin, y)
			d.DrawString(line)
			y += 54
		}
		y += 8
		draw.Draw(img, image.Rect(margin, y, PageWidth-margin, y+3), &image.Uniform{rule}, image.Point{}, draw.Src)
		y += 40
	}

	d.Face = bodyFace
	const lineH = 40
	truncated := false
	for _, para := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		lines := wrap(bodyFace, strings.TrimSpace(para), maxWidth)
		if len(lines) == 0 {
			y += lineH / 2
			continue
		}
		for _, line := range lines {
			if y > PageHeight-margin {
				truncated = true
				break
			}
			d.Dot = fixed.P(margin, y)
			d.DrawString(line)
			y += lineH
		}
		if truncated {
			break
		}
		y += lineH / 3
	}
	if truncated {
		d.Dot = fixed.P(margin, PageHeight-margin+30)
		d.Src = &image.Uniform{rule}
		d.DrawString("…")
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// wrap breaks text into lines no wider than maxWidth pixels.
func wrap(face font.Face, text string, maxWidth int) []string {
	if text == "" {
		return nil
	}
	width := func(s string) int { return font.MeasureString(face, s).Ceil() }
	var lines []string
	var cur string
	for _, word := range strings.Fields(text) {
		try := word
		if cur != "" {
			try = cur + " " + word
		}
		if width(try) <= maxWidth {
			cur = try
			continue
		}
		if cur != "" {
			lines = append(lines, cur)
		}
		// A single word wider than the page is split by characters.
		cur = ""
		for _, r := range word {
			if width(cur+string(r)) > maxWidth && cur != "" {
				lines = append(lines, cur)
				cur = ""
			}
			cur += string(r)
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
