# CLAUDE.md

Guidance for Claude Code when working in this repository.

## What this is

**AIrborne** is a Windows desktop app (Go + Wails v2 + React/TypeScript) that turns free-text
user input into playable mission files for **IL-2 Sturmovik: Korea** and **DCS World**.

The central design rule: **the LLM only ever produces a schema-validated JSON mission plan.
Deterministic Go code writes the actual mission files.** No LLM output is ever written into a
mission file verbatim. Keep it that way — it is what makes output reproducible and debuggable.

## Commands

All Go commands run from `app/`.

```powershell
cd app
wails dev                     # dev app; browser view on http://localhost:34115
wails build                   # -> build/bin/airborne.exe
go test ./internal/...        # generator + pipeline tests (all currently pass)
go build ./...
go run ./cmd/abgen -plan ../projects/current.json [-out <dir>] [-game il2|dcs]   # generate without GUI
```

Frontend alone (`app/frontend`): `npm run build` = `tsc && vite build`. No JS test suite.

`.env` lives in the repo root (git-ignored, template in `reference/.env.example`): LLM key/base/model,
`AI_MAX_TOKENS`, and the game paths `IL2_MISSIONS_DIR`, `IL2_EDITOR`, `IL2_ROOT`, `DCS_SAVED_GAMES`,
`DCS_ROOT`. `settings.json` in the root is written by the app itself (currently only `maxTokens`) and
is also git-ignored.

## Working in this repo — environment gotchas

These have bitten before and are not visible from the code:

- **The user usually keeps `wails dev` running themselves** (port 34115, process `airborne-dev.exe`).
  Check with `curl http://localhost:34115` before starting anything. Screenshots of that page tend to
  time out — use `get_page_text` / `find` / `javascript_tool` instead.
- **After changing exported `App` methods**, regenerate bindings with `wails generate module`. The Vite
  server inside a *running* `wails dev` then keeps serving the stale `wailsjs/go/main/App.js` (the dir is
  deleted+recreated and chokidar loses the watch on Windows), so the page breaks with
  *"does not provide an export named X"* until `wails dev` is restarted — ask the user to restart it.
- **The local `wails` CLI is v2.15.0 while `app/go.mod` pins v2.16.0.** Every `wails dev` /
  `wails generate module` downgrades `app/go.mod`, `app/go.sum` and rewrites
  `app/frontend/package.json.md5`. Run `git checkout -- app/go.mod app/go.sum app/frontend/package.json.md5`
  afterwards.
- **Most Go/TS files are CRLF** in the working copy. Use the Edit tool (or `perl` with `\r?$`), never
  `sed -i`, which rewrites them to LF.
- In Bash here, never use `python -` or `cat > "$UNSET/x"` — both hang waiting on stdin.
- No Lua interpreter is installed. DCS output is verified with the mini Lua parser in
  `app/internal/dcs/dcs_test.go` (`parseLua`), IL-2 output with `parseMission` in `il2_test.go`.

## Layout

```
app/
  main.go              Wails bootstrap (embeds frontend/dist, binds *App)
  app.go               Wails bindings: state, plan generation, export, prefabs, project I/O
  app_editor.go        Wails bindings for the no-LLM mission editor + file pickers
  cmd/abgen/           headless generator CLI
  internal/
    pipeline/          orchestration: prompts -> LLM -> plan -> check/repair -> generator
    plan/              MissionPlan types + Validate() (the shared contract)
    ai/                OpenAI-compatible chat client, .env loader, ExtractJSON
    il2/               .Mission/.ger/.list writer + IL-2 payload table
    dcs/               .miz builder (Lua tables, zip), lat/lon projection, DCS payload presets
    prefab/            reusable DCS asset groups (library in prefabs/, statics catalog)
    missionfile/       open existing .miz/.Mission, edit metadata in place, save with .bak
    media/             user images (JPG->PNG), briefing text rendered as a kneeboard page
    gen/               shared Result type + MissionName ("AB_" prefix)
    logging/           append-only logs/airborne.log
  frontend/src/        App.tsx (tabs/modes) + ChallengePane, MediaPanel, MissionEditor, SettingsPane
prompts/               versioned prompt templates (see below)
prefabs/               saved prefabs, one JSON per file
projects/current.json  autosaved current project (inputs + plan + output)
reference/il2-payloads.json   IL-2 PayloadId table
docs/                  format and workflow documentation
```

## The pipeline

`internal/pipeline` owns one `*Project` (`Game`, `Inputs`, `Plan`, `Output`, `Repair`) behind a mutex
and exposes a flat `State` struct to the frontend. Four entry points produce a plan:

| Mode | Entry | Prompt template |
|---|---|---|
| ⚡ Blitz (one free-text field) | `QuickMission` | `prompts/07-quick.md` |
| Step by step (5 fields) | `Merge` | `prompts/06-merge.md` |
| 🎯 Challenge (pick aircraft only) | `Challenge` | `prompts/09-challenge.md` |
| 🧩 Prefab builder (DCS) | `GeneratePrefab` | `prompts/08-prefab.md` |

All of them render the template with `{{PLACEHOLDER}}` substitution (`fill`), prepend
`prompts/00-system.md`, inject `prompts/schema.md` and the per-game context from `gameContext.go`,
then make **one** LLM call. The five step-by-step fields are *not* separate LLM calls — they are
embedded into a single structured prompt.

Then, always:

1. `decodePlan` — `ExtractJSON`, then `normalizeJSON` coerces the string-typed numbers LLMs emit
   (see `intFields` / `floatFields` / `boolFields` in `pipeline.go`) before unmarshalling.
2. `checkPlan` — `plan.Validate()` **plus a dry generator run into a temp dir**, collecting
   `gen.Result.Problems`.
3. `repair` (`repair.go`, `prompts/10-repair.md`) — if problems were found, one corrective LLM call.
   **The repaired plan is only accepted when it has strictly fewer problems**; any failure keeps the
   original. The outcome is recorded in `RepairInfo` and shown under "🔧 Reparatur".
4. `storePlan` — carries `plan.Media` over, since user media never comes from the LLM.

`Generate` then dispatches on `plan.Game` to `il2.GenerateOpts` or `dcs.GenerateOpts` with the output
dir from `.env`. `generateTo` is shared by the real export and the dry run, so the repair loop sees
exactly what the export would report.

## Conventions and invariants

- **`gen.Result.Notes` vs `.Problems`**: notes are informational (environment hints); problems are
  plan defects the author/LLM can fix. Only problems are fed back into the repair round. When adding a
  generator warning, use `problem(...)` only if the LLM could actually fix it, otherwise `note(...)`.
- **Generated missions are prefixed `AB_`** (`gen.MissionName`) so they never overwrite hand-made ones.
  Umlauts are transliterated and the base name is capped at 48 chars.
- **Language**: the app is bilingual, **German by default**, switchable to English (top-bar DE|EN or
  Settings; stored as `language` in `settings.json`). Frontend texts live in `frontend/src/i18n.tsx`
  (`en` is typed as `typeof de`, so a missing translation fails `tsc`). Backend user-facing messages use
  `internal/i18n` with inline pairs: `i18n.Errorf(de, en, args...)`, `i18n.Sprintf`, `i18n.T`, and the
  generator helpers `note(de, en, ...)` / `problem(de, en, ...)`. Never branch on message text (it
  depends on the language). Logs and prompts stay German; docs are German. Code comments and identifiers
  are **English**. Mission texts are bilingual via `plan.Localized{De, En}`; DCS (one text per field)
  writes the UI language first and appends the other.
- **Challenge mode invariants** are enforced deterministically, not by trusting the LLM:
  `challengeTrim` re-applies `Challenge=true`, drops `Icons` and forces a single player slot *before and
  after* the repair round; `Validate` reports player-facing texts that name an enemy unit type
  (`challengeLeaks`); IL-2 exports no icons; DCS sets `hidden*` on enemy groups and
  `forcedOptions.optionsView = optview_myaircraft`.
- **Route timing** is expressed through ground speed and distance, never fixed waypoint times — in DCS
  only WP1 has `ETA_locked=true`. This rule lives in `prompts/00-system.md` and in the generators.
- **`missionfile` edits byte-identically**: metadata is patched with targeted regexes, and top-level
  fields are recognised *by indentation* (1 tab from AIrborne, 4 spaces from the DCS editor) so group
  `start_time` is not hit. Everything else is written back unchanged; the first save creates a `.bak`.
- **Tests build plans in Go** (`cleanDCSPlan()` in `repair_test.go` is the canonical minimal DCS plan)
  and assert on the parsed output structure. `testPipeline(t)` copies the *real* prompt templates into
  a temp root and stubs the LLM via `Pipeline.chatFn`. Prefer extending these over new harnesses.
- All logging goes through `internal/logging` into `logs/airborne.log`; `logging.Block` dumps full
  prompts and responses, which is the first place to look when a plan is rejected.

## Reference documentation

Read these before touching a generator — they record hard-won, practice-verified format details:

| Doc | Contents |
|---|---|
| `docs/architecture.md` | stack decision, pipeline, binding list, roadmap |
| `docs/il2-korea-missions.md` | IL-2 formats (.Mission/.msnbin/.ger), MCU catalog, assets, pitfalls |
| `docs/dcs-miz-workflow.md` | .miz structure, dictionary/mapResource, gotchas |
| `docs/getting-started.md` | user-facing setup and troubleshooting |
| `prompts/schema.md` | the shared plan JSON schema and field rules (IL-2 / DCS) |

If you change `plan.MissionPlan`, update `prompts/schema.md` in the same change — the LLM contract and
the Go struct must not drift.
