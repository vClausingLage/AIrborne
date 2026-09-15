// Package missionfile opens existing missions (DCS .miz archives and IL-2
// Korea .Mission file sets) for editing without going through the LLM: the
// text parts can be edited verbatim, common metadata (title, briefing, date,
// time) is read and written in place, and briefing pictures / kneeboard
// pages can be attached. Everything else in the mission is preserved
// byte-for-byte.
package missionfile

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"

	"airborne/internal/dcs"
	"airborne/internal/media"
	"airborne/internal/plan"
)

// il2Exts are the files that make up one IL-2 mission (same base name).
var il2Exts = []string{".Mission", ".msnbin", ".list", ".png", ".jpg", ".ger", ".eng", ".rus", ".fra", ".spa", ".chs"}

var il2LangExts = map[string]bool{".ger": true, ".eng": true, ".rus": true, ".fra": true, ".spa": true, ".chs": true}

// Meta is the metadata common to both games. For DCS only the German
// (primary) fields are used; the .miz has one language.
type Meta struct {
	Title    plan.Localized `json:"title"`
	Briefing plan.Localized `json:"briefing"`
	Author   string         `json:"author"`
	Date     string         `json:"date"` // YYYY-MM-DD
	Time     string         `json:"time"` // HH:MM:SS
}

// Entry is one file of the mission (zip entry for DCS, file for IL-2).
type Entry struct {
	Name string `json:"name"`
	Size int    `json:"size"`
	Text bool   `json:"text"` // editable as text in the app
}

// Doc is the state handed to the UI.
type Doc struct {
	Game           string   `json:"game"`
	Path           string   `json:"path"`
	Name           string   `json:"name"`
	Entries        []Entry  `json:"entries"`
	Meta           Meta     `json:"meta"`
	Kneeboards     []string `json:"kneeboards"`     // DCS: KNEEBOARD/IMAGES entries
	BriefingImages []string `json:"briefingImages"` // pictures shown on the briefing screen
	Protected      bool     `json:"protected"`      // DCS: mission is not plain Lua
	Dirty          bool     `json:"dirty"`
	Notes          []string `json:"notes"`
}

// Mission is an opened mission held in memory.
type Mission struct {
	game  string
	path  string            // .miz or .Mission
	files map[string][]byte // zip entry (DCS) or file name (IL-2) -> content
	dirty bool
	notes []string
}

// Open loads a .miz or .Mission (plus its sibling files) into memory.
func Open(p string) (*Mission, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, fmt.Errorf("%s ist ein Ordner", p)
	}
	m := &Mission{path: p, files: map[string][]byte{}}
	switch strings.ToLower(filepath.Ext(p)) {
	case ".miz":
		m.game = "dcs"
		if err := m.openMiz(); err != nil {
			return nil, err
		}
	case ".mission":
		m.game = "il2"
		if err := m.openIL2(); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unbekannter Missionstyp %q (erwartet .miz oder .Mission)", filepath.Ext(p))
	}
	return m, nil
}

func (m *Mission) openMiz() error {
	zr, err := zip.OpenReader(m.path)
	if err != nil {
		return fmt.Errorf(".miz ist kein ZIP-Archiv: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		m.files[path.Clean(strings.ReplaceAll(f.Name, "\\", "/"))] = data
	}
	if _, ok := m.files["mission"]; !ok {
		return fmt.Errorf("Archiv enthaelt keine 'mission'-Datei")
	}
	if m.protected() {
		m.note("Die 'mission'-Datei ist kein Lua-Text (geschuetzte/binaere Mission) - nur Kneeboards und Bilder koennen geaendert werden")
	}
	return nil
}

func (m *Mission) openIL2() error {
	base := strings.TrimSuffix(m.path, filepath.Ext(m.path))
	for _, ext := range il2Exts {
		p := base + ext
		if ext == ".Mission" {
			p = m.path
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		m.files[filepath.Base(p)] = data
	}
	if _, ok := m.files[filepath.Base(m.path)]; !ok {
		return fmt.Errorf("%s nicht lesbar", m.path)
	}
	if _, ok := m.files[filepath.Base(base)+".msnbin"]; ok {
		m.note("Eine .msnbin liegt daneben - sie wird beim Speichern entfernt, damit die Text-.Mission wieder gilt (Editor erzeugt sie neu)")
	}
	return nil
}

func (m *Mission) note(format string, args ...any) {
	m.notes = append(m.notes, fmt.Sprintf(format, args...))
}

func (m *Mission) protected() bool {
	return m.game == "dcs" && !bytes.HasPrefix(bytes.TrimLeft(m.files["mission"], "\xef\xbb\xbf \t\r\n"), []byte("mission"))
}

// Game returns "il2" or "dcs".
func (m *Mission) Game() string { return m.game }

// Path returns the file the mission was opened from (or last saved to).
func (m *Mission) Path() string { return m.path }

func (m *Mission) baseName() string {
	return strings.TrimSuffix(filepath.Base(m.path), filepath.Ext(m.path))
}

// isText reports whether an entry is edited as text in the app.
func (m *Mission) isText(name string) bool {
	if m.game == "il2" {
		ext := strings.ToLower(filepath.Ext(name))
		return ext == ".mission" || ext == ".list" || il2LangExts[ext]
	}
	if strings.HasPrefix(name, "l10n/") && (path.Base(name) == "dictionary" || path.Base(name) == "mapResource") {
		return true
	}
	switch name {
	case "mission", "options", "theatre", "warehouses":
		return name != "mission" || !m.protected()
	}
	return false
}

// Doc renders the UI state.
func (m *Mission) Doc() Doc {
	d := Doc{Game: m.game, Path: m.path, Name: m.baseName(), Entries: []Entry{}, Kneeboards: []string{},
		BriefingImages: []string{}, Protected: m.protected(), Dirty: m.dirty, Notes: append([]string{}, m.notes...)}
	if d.Notes == nil {
		d.Notes = []string{}
	}
	names := make([]string, 0, len(m.files))
	for n := range m.files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		d.Entries = append(d.Entries, Entry{Name: n, Size: len(m.files[n]), Text: m.isText(n)})
		if m.game == "dcs" && strings.HasPrefix(n, dcs.KneeboardDir) {
			d.Kneeboards = append(d.Kneeboards, n)
		}
	}
	d.Meta = m.Meta()
	d.BriefingImages = m.briefingImages()
	return d
}

// Read returns a text entry (IL-2 language files are decoded from UTF-16).
func (m *Mission) Read(name string) (string, error) {
	data, ok := m.files[name]
	if !ok {
		return "", fmt.Errorf("Eintrag %q nicht vorhanden", name)
	}
	if !m.isText(name) {
		return "", fmt.Errorf("%q ist keine Textdatei", name)
	}
	if m.game == "il2" && il2LangExts[strings.ToLower(filepath.Ext(name))] {
		return decodeUTF16(data), nil
	}
	return string(data), nil
}

// Write replaces a text entry.
func (m *Mission) Write(name, text string) error {
	if _, ok := m.files[name]; !ok {
		return fmt.Errorf("Eintrag %q nicht vorhanden", name)
	}
	if !m.isText(name) {
		return fmt.Errorf("%q ist keine Textdatei", name)
	}
	if m.game == "il2" && il2LangExts[strings.ToLower(filepath.Ext(name))] {
		m.files[name] = encodeUTF16(text)
	} else {
		m.files[name] = []byte(text)
	}
	m.dirty = true
	return nil
}

// Remove deletes an entry (kneeboard page, picture, ...). Core files are protected.
func (m *Mission) Remove(name string) error {
	if _, ok := m.files[name]; !ok {
		return fmt.Errorf("Eintrag %q nicht vorhanden", name)
	}
	if m.game == "dcs" {
		switch name {
		case "mission", "options", "theatre", "warehouses", "l10n/DEFAULT/dictionary", "l10n/DEFAULT/mapResource":
			return fmt.Errorf("%q gehoert zum Kern der Mission und kann nicht entfernt werden", name)
		}
		if strings.HasPrefix(name, "l10n/") {
			m.unreferenceResource(path.Base(name))
		}
	} else {
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".mission" || il2LangExts[ext] {
			return fmt.Errorf("%q gehoert zum Kern der Mission und kann nicht entfernt werden", name)
		}
	}
	delete(m.files, name)
	m.dirty = true
	return nil
}

// AddKneeboard appends a PNG/JPG as kneeboard page (DCS only).
func (m *Mission) AddKneeboard(p string) error {
	if m.game != "dcs" {
		return fmt.Errorf("Kneeboards gibt es nur in DCS")
	}
	data, err := media.LoadPNG(p)
	if err != nil {
		return err
	}
	n := 0
	for name := range m.files {
		if strings.HasPrefix(name, dcs.KneeboardDir) {
			n++
		}
	}
	name := fmt.Sprintf("%s%02d_%s.png", dcs.KneeboardDir, n+1, media.SafeName(p))
	for {
		if _, exists := m.files[name]; !exists {
			break
		}
		n++
		name = fmt.Sprintf("%s%02d_%s.png", dcs.KneeboardDir, n+1, media.SafeName(p))
	}
	m.files[name] = data
	m.dirty = true
	return nil
}

// AddBriefingText renders text as a kneeboard page (DCS only).
func (m *Mission) AddBriefingText(title, body string) error {
	if m.game != "dcs" {
		return fmt.Errorf("Kneeboards gibt es nur in DCS")
	}
	data, err := media.RenderTextPage(title, body)
	if err != nil {
		return err
	}
	n := 0
	for name := range m.files {
		if strings.HasPrefix(name, dcs.KneeboardDir) {
			n++
		}
	}
	m.files[fmt.Sprintf("%s%02d_text.png", dcs.KneeboardDir, n+1)] = data
	m.dirty = true
	return nil
}

// SetBriefingImage attaches the briefing-screen picture. IL-2: <Name>.png
// next to the mission; DCS: l10n/DEFAULT/<file>.png + mapResource key +
// pictureFileNameB/R (both coalitions).
func (m *Mission) SetBriefingImage(p string) error {
	data, err := media.LoadPNG(p)
	if err != nil {
		return err
	}
	if m.game == "il2" {
		delete(m.files, m.baseName()+".jpg")
		m.files[m.baseName()+".png"] = data
		m.dirty = true
		return nil
	}
	if m.protected() {
		return fmt.Errorf("geschuetzte Mission: Briefing-Bild kann nicht verknuepft werden")
	}
	name := media.SafeName(p) + ".png"
	m.files["l10n/DEFAULT/"+name] = data
	key := "ResKey_ImageBriefing_" + media.SafeName(p)
	if err := m.setResource(key, name); err != nil {
		return err
	}
	mission := string(m.files["mission"])
	for _, field := range []string{"pictureFileNameB", "pictureFileNameR"} {
		mission, err = appendLuaArrayValue(mission, field, key)
		if err != nil {
			return err
		}
	}
	m.files["mission"] = []byte(mission)
	m.dirty = true
	return nil
}

// briefingImages lists the pictures currently shown on the briefing screen.
func (m *Mission) briefingImages() []string {
	out := []string{}
	if m.game == "il2" {
		for _, ext := range []string{".png", ".jpg"} {
			if _, ok := m.files[m.baseName()+ext]; ok {
				out = append(out, m.baseName()+ext)
			}
		}
		return out
	}
	if m.protected() {
		return out
	}
	res := parseLuaStringTable(string(m.files["l10n/DEFAULT/mapResource"]))
	mission := string(m.files["mission"])
	seen := map[string]bool{}
	for _, field := range []string{"pictureFileNameB", "pictureFileNameR", "pictureFileNameN"} {
		for _, key := range luaArrayValues(mission, field) {
			if file, ok := res[key]; ok && !seen[file] {
				seen[file] = true
				out = append(out, "l10n/DEFAULT/"+file)
			}
		}
	}
	return out
}

// Meta reads title/briefing/author/date/time from the mission.
func (m *Mission) Meta() Meta {
	if m.game == "il2" {
		return m.metaIL2()
	}
	return m.metaDCS()
}

// SetMeta writes the metadata back in place.
func (m *Mission) SetMeta(meta Meta) error {
	var err error
	if m.game == "il2" {
		err = m.setMetaIL2(meta)
	} else {
		err = m.setMetaDCS(meta)
	}
	if err == nil {
		m.dirty = true
	}
	return err
}

// Save writes the mission back to its path (a .bak copy is kept once).
func (m *Mission) Save() error { return m.SaveAs(m.path) }

// SaveAs writes the mission to a new path; IL-2 sibling files follow the new base name.
func (m *Mission) SaveAs(p string) error {
	if p == "" {
		p = m.path
	}
	if m.game == "dcs" {
		if !strings.EqualFold(filepath.Ext(p), ".miz") {
			p += ".miz"
		}
		backup(p)
		if err := dcs.WriteZip(p, m.files); err != nil {
			return err
		}
		m.path = p
		m.dirty = false
		return nil
	}
	if !strings.EqualFold(filepath.Ext(p), ".mission") {
		p += ".Mission"
	}
	oldBase := m.baseName()
	newBase := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	renamed := map[string][]byte{}
	for name, data := range m.files {
		ext := filepath.Ext(name)
		if strings.EqualFold(ext, ".msnbin") {
			continue // stale binary mirror would shadow the edited text mission
		}
		renamed[newBase+ext] = data
	}
	for name, data := range renamed {
		target := filepath.Join(dir, name)
		backup(target)
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	if err := os.Remove(filepath.Join(dir, newBase+".msnbin")); err == nil {
		m.note("%s.msnbin entfernt (wird beim Speichern im IL-2 Editor neu erzeugt)", newBase)
	}
	_ = oldBase
	m.files = renamed
	m.path = filepath.Join(dir, newBase+".Mission")
	m.dirty = false
	return nil
}

// backup keeps the first pre-edit copy of a file as NAME.bak.
func backup(p string) {
	if _, err := os.Stat(p); err != nil {
		return
	}
	bak := p + ".bak"
	if _, err := os.Stat(bak); err == nil {
		return
	}
	data, err := os.ReadFile(p)
	if err == nil {
		_ = os.WriteFile(bak, data, 0o644)
	}
}

// ---------------------------------------------------------------------------
// UTF-16 helpers (IL-2 language files)

func decodeUTF16(data []byte) string {
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		data = data[2:]
	} else if !(len(data) >= 2 && data[1] == 0 && data[0] != 0) {
		return string(data) // not UTF-16LE: keep as-is
	}
	u := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		u = append(u, uint16(data[i])|uint16(data[i+1])<<8)
	}
	return string(utf16.Decode(u))
}

func encodeUTF16(s string) []byte {
	out := []byte{0xFF, 0xFE}
	for _, u := range utf16.Encode([]rune(s)) {
		out = append(out, byte(u), byte(u>>8))
	}
	return out
}
