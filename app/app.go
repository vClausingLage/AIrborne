package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"airborne/internal/ai"
	"airborne/internal/gen"
	"airborne/internal/logging"
	"airborne/internal/pipeline"
	"airborne/internal/prefab"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
	pl  *pipeline.Pipeline
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cwd, _ := os.Getwd()
	exe, _ := os.Executable()
	root := pipeline.FindRoot(cwd, exe, "C:\\Users\\vince\\js_projects\\AIrborne")
	if root == "" {
		logging.Init(os.TempDir())
		return
	}
	logging.Init(filepath.Join(root, "logs"))
	ai.LoadEnv(ai.FindEnvFile(root, root+"\\app", root+"\\reference")...)
	pl, err := pipeline.New(root)
	if err == nil {
		a.pl = pl
	}
}

func (a *App) GetState() (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	return a.pl.State(), nil
}

func (a *App) SetRoleInput(role, text string) error {
	if a.pl == nil {
		return errNoProject()
	}
	a.pl.SetInput(role, text)
	return nil
}

// MergePlan builds one structured prompt from the five sections and returns the plan.
func (a *App) MergePlan() (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	if _, err := a.pl.Merge(a.ctx); err != nil {
		return pipeline.State{}, err
	}
	a.saveQuietly()
	return a.pl.State(), nil
}

func (a *App) SetGame(game string) error {
	if a.pl == nil {
		return errNoProject()
	}
	a.pl.SetGame(game)
	a.saveQuietly()
	return nil
}

func (a *App) QuickMission(input string) (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	if _, err := a.pl.QuickMission(a.ctx, input); err != nil {
		return pipeline.State{}, err
	}
	a.saveQuietly()
	return a.pl.State(), nil
}

// SetPlanJSON accepts a hand-edited plan.
func (a *App) SetPlanJSON(text string) (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	if err := a.pl.SetPlanJSON(text); err != nil {
		return pipeline.State{}, err
	}
	a.saveQuietly()
	return a.pl.State(), nil
}

// GenerateMission writes the mission files for the current plan into the
// game's mission folder (IL2_MISSIONS_DIR / DCS_SAVED_GAMES\Missions).
func (a *App) GenerateMission() (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	if _, err := a.pl.Generate(); err != nil {
		return pipeline.State{}, err
	}
	a.saveQuietly()
	return a.pl.State(), nil
}

// OpenOutputFolder shows the generated mission in Explorer.
func (a *App) OpenOutputFolder() error {
	if a.pl == nil {
		return errNoProject()
	}
	out := a.pl.State().Output
	if out == nil {
		return fmt.Errorf("Noch keine Mission generiert")
	}
	return exec.Command("explorer.exe", "/select,", out.MainFile).Start()
}

// OpenIL2Editor starts the IL-2 mission editor with the mandatory working
// directory bin\editor (see docs/il2-korea-missions.md).
func (a *App) OpenIL2Editor() error {
	paths := pipeline.LoadPaths()
	if paths.IL2Editor == "" {
		return fmt.Errorf("IL2_EDITOR ist nicht gesetzt (.env)")
	}
	if _, err := os.Stat(paths.IL2Editor); err != nil {
		return fmt.Errorf("IL-2 Editor nicht gefunden: %s", paths.IL2Editor)
	}
	cmd := exec.Command(paths.IL2Editor)
	cmd.Dir = filepath.Dir(paths.IL2Editor)
	return cmd.Start()
}

func (a *App) SaveProject(path string) (string, error) {
	if a.pl == nil {
		return "", errNoProject()
	}
	p, err := a.pl.SaveProject(path)
	if err != nil {
		return "", err
	}
	return p, nil
}

func (a *App) LoadProject(path string) (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	if err := a.pl.LoadProject(path); err != nil {
		return pipeline.State{}, err
	}
	return a.pl.State(), nil
}

func (a *App) PickFile(title string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "airborne-project.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "Airborne Project (*.json)", Pattern: "*.json"},
		},
	})
}

func (a *App) OpenFile(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "Airborne Project (*.json)", Pattern: "*.json"},
		},
	})
}

func (a *App) saveQuietly() {
	if _, err := a.pl.SaveProject(""); err != nil {
		runtime.LogWarningf(a.ctx, "Projekt konnte nicht gespeichert werden: %v", err)
	}
}

func errNoProject() error {
	return fmt.Errorf("Projekt-Root nicht gefunden (prompts/ fehlt). Airborne aus dem Projektordner starten oder .env/PROMPTS_DIR pruefen")
}

// keep gen imported for the binding generator (Result is part of State).
var _ gen.Result

// ---------------------------------------------------------------------------
// Prefabs (DCS asset groups, see internal/prefab)

// GeneratePrefab asks the LLM for a prefab layout; the draft is returned for
// review and is not stored until SavePrefab is called.
func (a *App) GeneratePrefab(input string) (prefab.Prefab, error) {
	if a.pl == nil {
		return prefab.Prefab{}, errNoProject()
	}
	return a.pl.GeneratePrefab(a.ctx, input)
}

// SavePrefab stores (possibly hand-edited) prefab JSON in <root>/prefabs.
func (a *App) SavePrefab(text string) (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	if _, err := a.pl.SavePrefab(text); err != nil {
		return pipeline.State{}, err
	}
	return a.pl.State(), nil
}

func (a *App) DeletePrefab(id string) (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	if err := a.pl.DeletePrefab(id); err != nil {
		return pipeline.State{}, err
	}
	return a.pl.State(), nil
}

// PlacePrefab adds a prefab placement (lat/lon/heading, side) to the plan.
func (a *App) PlacePrefab(id, side string, lat, lon, heading float64, country string) (pipeline.State, error) {
	if a.pl == nil {
		return pipeline.State{}, errNoProject()
	}
	if err := a.pl.PlacePrefab(id, side, lat, lon, heading, country); err != nil {
		return pipeline.State{}, err
	}
	a.saveQuietly()
	return a.pl.State(), nil
}

// OpenPrefabFolder shows the prefab library in Explorer.
func (a *App) OpenPrefabFolder() error {
	if a.pl == nil {
		return errNoProject()
	}
	if err := os.MkdirAll(a.pl.Lib.Dir, 0o755); err != nil {
		return err
	}
	return exec.Command("explorer.exe", a.pl.Lib.Dir).Start()
}
