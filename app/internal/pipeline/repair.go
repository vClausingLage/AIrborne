package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"airborne/internal/i18n"
	"airborne/internal/logging"
	"airborne/internal/plan"
)

// RepairInfo records what the repair round did with a freshly generated
// plan; it is kept in the project so the UI can show it next to the plan.
type RepairInfo struct {
	// Found are the problems the check reported for the LLM's first answer.
	Found []string `json:"found"`
	// Repaired is true when the corrected plan replaced the first answer.
	Repaired bool `json:"repaired"`
	// Remaining are the problems still present after the round (the first
	// answer's problems when the repair was rejected or failed).
	Remaining []string `json:"remaining"`
	// Note explains a rejected or failed repair.
	Note string `json:"note,omitempty"`
}

// checkPlan collects everything the plan author could fix: schema-level
// issues from Validate plus the problems a dry generator run into a scratch
// folder reports (unknown unit types, missing targets, ...). A generator
// error counts as one problem.
func (p *Pipeline) checkPlan(mp *plan.MissionPlan) []string {
	issues := append([]string{}, mp.Validate()...)
	dir, err := os.MkdirTemp("", "airborne-check-*")
	if err != nil {
		logging.Errorf("Probelauf: Temp-Ordner nicht anlegbar: %v", err)
		return issues
	}
	defer os.RemoveAll(dir)
	res, err := p.generateTo(mp, dir)
	if err != nil {
		return append(issues, "Generator: "+err.Error())
	}
	return append(issues, res.Problems...)
}

// repair runs the check on a plan the LLM just produced and, when it finds
// problems, asks the LLM once for a corrected plan. The corrected plan is
// only taken when it has fewer problems than the original; any failure
// keeps the original so a repair can never lose a usable plan.
func (p *Pipeline) repair(ctx context.Context, label, game string, mp plan.MissionPlan) (plan.MissionPlan, *RepairInfo) {
	found := p.checkPlan(&mp)
	if len(found) == 0 {
		logging.Infof("%s: Plan ohne Beanstandungen", label)
		return mp, nil
	}
	info := &RepairInfo{Found: found, Remaining: found}
	logging.Infof("%s: %d Problem(e) - Reparatur-Durchgang", label, len(found))
	for _, f := range found {
		logging.Infof("  - %s", f)
	}

	planJSON, err := json.MarshalIndent(mp, "", "  ")
	if err != nil {
		info.Note = i18n.T("Plan nicht serialisierbar: ", "Plan cannot be serialized:") + err.Error()
		return mp, info
	}
	p.mu.Lock()
	prompt := fill(p.tmpl[repairFile], map[string]string{
		"GAME_CONTEXT": p.gameContext(game),
		"ISSUES":       "- " + strings.Join(found, "\n- "),
		"PLAN_JSON":    string(planJSON),
		"SCHEMA":       p.tmpl[schemaFile],
	})
	p.mu.Unlock()

	fixed, err := p.runPlanPrompt(ctx, label+"-repair", game, prompt)
	if err != nil {
		info.Note = i18n.T("Reparatur fehlgeschlagen, erster Plan behalten: ", "Repair failed, first plan kept:") + err.Error()
		logging.Errorf("%s", info.Note)
		return mp, info
	}
	if fixed.Game == "" {
		fixed.Game = game
	}
	fixed.Challenge = mp.Challenge
	remaining := p.checkPlan(&fixed)
	if len(remaining) >= len(found) {
		info.Note = i18n.T("Reparatur verworfen: korrigierter Plan hat nicht weniger Probleme als der erste", "Repair discarded: the corrected plan does not have fewer problems than the first")
		logging.Infof("%s (%d >= %d)", info.Note, len(remaining), len(found))
		return mp, info
	}
	info.Repaired = true
	info.Remaining = remaining
	logging.Infof("%s: Reparatur uebernommen, %d von %d Problem(en) behoben", label, len(found)-len(remaining), len(found))
	for _, r := range remaining {
		logging.Infof("  - offen: %s", r)
	}
	return fixed, info
}
