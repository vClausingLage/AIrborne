package main

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"airborne/internal/logging"
	"airborne/internal/missionfile"
	"airborne/internal/pipeline"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Files without the LLM: pictures

var imageFilters = []runtime.FileFilter{{DisplayName: "Bilder (*.png, *.jpg)", Pattern: "*.png;*.jpg;*.jpeg"}}

// PickImage opens a PNG/JPG file picker ("" when cancelled).
func (a *App) PickImage(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: title, Filters: imageFilters})
}

// PickImages opens a multi-select PNG/JPG file picker.
func (a *App) PickImages(title string) ([]string, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{Title: title, Filters: imageFilters})
	if paths == nil {
		paths = []string{}
	}
	return paths, err
}

// ---------------------------------------------------------------------------
// Mission editor (existing .miz / .Mission files, see internal/missionfile)

var missionFilters = []runtime.FileFilter{
	{DisplayName: "Missionen (*.miz, *.Mission)", Pattern: "*.miz;*.Mission"},
	{DisplayName: "DCS World (*.miz)", Pattern: "*.miz"},
	{DisplayName: "IL-2 Korea (*.Mission)", Pattern: "*.Mission"},
}

func errNoMission() error { return fmt.Errorf("Keine Mission geoeffnet") }

// PickMissionFile asks for an existing mission, starting in the game folder.
func (a *App) PickMissionFile() (string, error) {
	paths := pipeline.LoadPaths()
	dir := paths.IL2MissionsDir
	if a.pl != nil && a.pl.State().Game == "dcs" {
		dir = paths.DCSMissionsDir
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Mission öffnen", DefaultDirectory: dir, Filters: missionFilters})
}

// OpenMission loads a mission into the editor.
func (a *App) OpenMission(path string) (missionfile.Doc, error) {
	m, err := missionfile.Open(path)
	if err != nil {
		return missionfile.Doc{}, err
	}
	a.mission = m
	logging.Infof("Mission geoeffnet: %s (%s)", path, m.Game())
	return m.Doc(), nil
}

// OpenGeneratedMission loads the last exported mission into the editor.
func (a *App) OpenGeneratedMission() (missionfile.Doc, error) {
	if a.pl == nil {
		return missionfile.Doc{}, errNoProject()
	}
	out := a.pl.State().Output
	if out == nil {
		return missionfile.Doc{}, fmt.Errorf("Noch keine Mission generiert")
	}
	return a.OpenMission(out.MainFile)
}

func (a *App) missionDoc() (missionfile.Doc, error) {
	if a.mission == nil {
		return missionfile.Doc{}, errNoMission()
	}
	return a.mission.Doc(), nil
}

// MissionDoc returns the state of the open mission (error when none is open).
func (a *App) MissionDoc() (missionfile.Doc, error) { return a.missionDoc() }

func (a *App) MissionClose() { a.mission = nil }

// MissionRead returns a text entry of the open mission.
func (a *App) MissionRead(name string) (string, error) {
	if a.mission == nil {
		return "", errNoMission()
	}
	return a.mission.Read(name)
}

// MissionWrite replaces a text entry verbatim.
func (a *App) MissionWrite(name, text string) (missionfile.Doc, error) {
	if a.mission == nil {
		return missionfile.Doc{}, errNoMission()
	}
	if err := a.mission.Write(name, text); err != nil {
		return missionfile.Doc{}, err
	}
	return a.missionDoc()
}

// MissionSetMeta writes title/briefing/author/date/time in place.
func (a *App) MissionSetMeta(meta missionfile.Meta) (missionfile.Doc, error) {
	if a.mission == nil {
		return missionfile.Doc{}, errNoMission()
	}
	if err := a.mission.SetMeta(meta); err != nil {
		return missionfile.Doc{}, err
	}
	return a.missionDoc()
}

// MissionAddKneeboards appends PNG/JPG files as kneeboard pages (DCS).
func (a *App) MissionAddKneeboards(paths []string) (missionfile.Doc, error) {
	if a.mission == nil {
		return missionfile.Doc{}, errNoMission()
	}
	for _, p := range paths {
		if err := a.mission.AddKneeboard(p); err != nil {
			return missionfile.Doc{}, err
		}
	}
	return a.missionDoc()
}

// MissionAddBriefingText renders text as a kneeboard page (DCS).
func (a *App) MissionAddBriefingText(title, body string) (missionfile.Doc, error) {
	if a.mission == nil {
		return missionfile.Doc{}, errNoMission()
	}
	if err := a.mission.AddBriefingText(title, body); err != nil {
		return missionfile.Doc{}, err
	}
	return a.missionDoc()
}

// MissionSetBriefingImage attaches the briefing-screen picture.
func (a *App) MissionSetBriefingImage(path string) (missionfile.Doc, error) {
	if a.mission == nil {
		return missionfile.Doc{}, errNoMission()
	}
	if err := a.mission.SetBriefingImage(path); err != nil {
		return missionfile.Doc{}, err
	}
	return a.missionDoc()
}

// MissionRemove deletes an entry (kneeboard page, picture, ...).
func (a *App) MissionRemove(name string) (missionfile.Doc, error) {
	if a.mission == nil {
		return missionfile.Doc{}, errNoMission()
	}
	if err := a.mission.Remove(name); err != nil {
		return missionfile.Doc{}, err
	}
	return a.missionDoc()
}

// MissionSave writes the mission back (path "" = in place; the first save
// keeps a .bak copy of the original).
func (a *App) MissionSave(path string) (missionfile.Doc, error) {
	if a.mission == nil {
		return missionfile.Doc{}, errNoMission()
	}
	if err := a.mission.SaveAs(path); err != nil {
		return missionfile.Doc{}, err
	}
	logging.Infof("Mission gespeichert: %s", a.mission.Path())
	return a.missionDoc()
}

// PickMissionSavePath asks where to save the open mission.
func (a *App) PickMissionSavePath() (string, error) {
	if a.mission == nil {
		return "", errNoMission()
	}
	filter := missionFilters[1]
	if a.mission.Game() == "il2" {
		filter = missionFilters[2]
	}
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Mission speichern unter",
		DefaultDirectory: filepath.Dir(a.mission.Path()),
		DefaultFilename:  filepath.Base(a.mission.Path()),
		Filters:          []runtime.FileFilter{filter},
	})
}

// OpenMissionFolder shows the open mission in Explorer.
func (a *App) OpenMissionFolder() error {
	if a.mission == nil {
		return errNoMission()
	}
	return exec.Command("explorer.exe", "/select,", a.mission.Path()).Start()
}
