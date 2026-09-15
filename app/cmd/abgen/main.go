// abgen: generate mission files from a plan JSON without the GUI.
//
//	go run ./cmd/abgen -plan ../projects/current.json -out C:\tmp\out
//
// Without -out the game folder from .env is used (like the app does).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"airborne/internal/ai"
	"airborne/internal/dcs"
	"airborne/internal/gen"
	"airborne/internal/il2"
	"airborne/internal/pipeline"
	"airborne/internal/plan"
	"airborne/internal/prefab"
)

func main() {
	planPath := flag.String("plan", "", "plan JSON (MissionPlan or project file with .plan)")
	out := flag.String("out", "", "output folder (default: game folder from .env)")
	game := flag.String("game", "", "override game: il2 | dcs")
	flag.Parse()
	if *planPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	data, err := os.ReadFile(*planPath)
	if err != nil {
		fail(err)
	}
	var mp plan.MissionPlan
	var proj struct {
		Plan *plan.MissionPlan `json:"plan"`
	}
	if json.Unmarshal(data, &proj) == nil && proj.Plan != nil {
		mp = *proj.Plan
	} else if err := json.Unmarshal(data, &mp); err != nil {
		fail(err)
	}
	if *game != "" {
		mp.Game = *game
	}
	for _, i := range mp.Validate() {
		fmt.Println("WARN:", i)
	}
	cwd, _ := os.Getwd()
	root := pipeline.FindRoot(cwd, filepath.Dir(cwd))
	if *out == "" {
		ai.LoadEnv(ai.FindEnvFile(root, filepath.Join(root, "app"))...)
		p := pipeline.LoadPaths()
		if mp.Game == "dcs" {
			*out = p.DCSMissionsDir
		} else {
			*out = p.IL2MissionsDir
		}
		if *out == "" {
			fail(fmt.Errorf("no output folder in .env for %s", mp.Game))
		}
	}
	var res *gen.Result
	if mp.Game == "dcs" {
		res, err = dcs.GenerateWith(&mp, *out, prefab.NewLibrary(filepath.Join(root, "prefabs")))
	} else {
		table, terr := il2.LoadPayloadTable(filepath.Join(root, "reference", "il2-payloads.json"))
		if terr != nil {
			fmt.Println("warn:", terr)
		}
		res, err = il2.GenerateOpts(&mp, *out, table)
	}
	if err != nil {
		fail(err)
	}
	fmt.Println("written:", res.MainFile)
	for _, f := range res.Files {
		fmt.Println("  ", f)
	}
	for _, n := range res.Notes {
		fmt.Println("note:", n)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
