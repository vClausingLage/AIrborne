# AIrborne – Architektur & Technologie-Entscheidung

Meta-App zur KI-gestützten Erstellung realistischer, immersiver Missionsdateien für **IL-2 Korea** und **DCS World**.

## 1. Tech-Stack-Empfehlung: Go + Wails v2 + React/TypeScript

| Kriterium | Bewertung |
|---|---|
| Backend-Sprache | **Go** (entsprechend deinem Wunsch): stark bei Datei-IO (ZIP = `archive/zip` in Stdlib), Concurrency (Prompt-Pipeline parallel), Single-Binary-Build |
| Desktop-Framework | **Wails v2** – Go-Backend + Webview-Frontend, kein Chromium-Bundle (kleine Exe), native Fenster |
| Frontend | **React + TypeScript + Vite** – du hast React/TS/JS-Erfahrung; TailwindCSS optional |
| Warum nicht Electron/Tauri | Electron: unnötiges Chromium. Tauri: Rust-Backend (zweite Sprache). Wails: reines Go + dein React-Wissen |

**Setup:**
```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd C:\Users\vince\js_projects\AIrborne\app
wails init -n airborne -t react-ts
wails dev
```

### Struktur (Monorepo)

```
AIrborne/
  app/                  # Wails-App (Go + React/TS)
    main.go             # Wails-Bootstrap
    app.go              # Gebundene Backend-Methoden (front-end callable)
    internal/
      il2/              # IL-2 .Mission-Generator/-Parser (siehe docs/il2-korea-missions.md)
      dcs/              # DCS .miz: unzip → parse lua-Tables → edit → rezip
      ai/               # Provider-Abstraktion (OpenAI-kompatibel, später TTS)
      assets/           # Asset-Katalog (JSON): Einheit→Fraktion→Epoche→Modul→Karte
      pipeline/         # Prompt-Orchestrierung: 5 Rollen → Merge → Missionsplan → Dateien
      prefab/           # DCS-Prefabs: Asset-Gruppen (Schiffe/Fahrzeuge/Statics) relativ in Metern, Katalog + Bibliothek
  prompts/              # Versionierte Prompt-Templates (Markdown, 1 Datei je Rolle)
  prefabs/              # Gespeicherte Prefabs (JSON, je Datei eines) - Bibliothek der App
  docs/                 # Format- und Workflow-Doku
  reference/            # Asset-Kataloge, Beispiel-Missionen, .env.example, il2-payloads.json (PayloadId-Tabelle)
```

## 2. Pipeline (Kern der App)

```
User-Eingaben (5 Felder: Story, Player, Gegner, Widerstand, Briefing)
   – oder ⚡ Blitz: ein Freitext –
        │
        ▼
System-Prompt (prompts/00-system.md: Rolle, Zielspiel, Grundsaetze –
  Wegpunkte mit Aufgabe, plausible Bewaffnung, keine Anachronismen)
+ EIN strukturierter User-Prompt (prompts/06-merge.md bzw. 07-quick.md + schema.md):
  "Deine Aufgabe ist es, eine Mission zu erstellen … # Story … # Player … # Gegner …"
  (kein LLM-Aufruf je Rolle; leere Felder entscheidet die KI passend zur Story)
        │
        ▼
LLM liefert NUR den strukturierten Missionsplan (JSON):
        MissionPlan = { game:"il2|dcs", era, map, date, time, weather,
                        playerGroups[], enemyGroups[], statics[],
                        objectives[], radioQueue[], briefing{de,en} }
        │                     │
        ▼                     ▼
   internal/il2           internal/dcs
   (.Mission + .ger/.eng  (mission/options/theatre/warehouses/l10n von
   UTF-16LE + .list →      Grund auf als Lua-Tabellen, lat/lon→x/y per
   IL2_MISSIONS_DIR)       Transverse Mercator, zip → DCS_SAVED_GAMES\Missions)
        │
        ▼
   Ergebnis im Tab "Plan & Export": Pfad, Hinweise, "Ordner öffnen", "IL-2 Editor starten"
```

Wichtige Generator-Regeln: IL-2 Objekt/TR_Entity-Paare, `OnEvent Type 13` → Counter → Timer 4 s → Objective + Erfolgs-Subtitle, CheckZone auf Spieler-Entities, Bodenrouten werden über MissionBegin → Timer aktiviert. DCS: Spieler-Einheiten als `Client`, Trigger als `trigrules` + `trig.actions/conditions/func/flag`, Texte nur über `l10n/DEFAULT/dictionary`. Beide Generatoren sind mit `go test ./internal/...` gegen Struktur-Checks abgesichert; `cmd/abgen` generiert aus einem Plan-JSON ohne GUI.

**Prefabs (DCS):** Im Modus „🧩 Prefabs" (nur bei DCS sichtbar) beschreibt der Nutzer eine Asset-Gruppe („Trägerverband mit geparkten Hornets", „FARP mit Zelten und Tankwagen"); `prompts/08-prefab.md` liefert ein `prefab.Prefab`-JSON mit Elementen in **relativen Metern** (`dx` Nord, `dy` Ost, `heading` relativ) plus statischem Objektkatalog (`internal/prefab/catalog.go`: type → category/shape_name). Gespeichert wird in `prefabs/<id>.json`. Ein Plan platziert Prefabs über `prefabs: [{prefab, side, position{lat,lon,heading}}]` – entweder per Formular in der App oder durch die Missions-KI selbst (die Liste der gespeicherten Prefabs wird in den DCS-Spiel-Kontext eingeblendet). Der DCS-Generator dreht das Prefab um den Ursprung und erzeugt Schiffs-/Fahrzeuggruppen (gleicher `group`-Name = eine DCS-Gruppe) sowie echte `static`-Gruppen; Elemente mit `linkTo` werden per `linkUnit/offsets` an das Schiff gebunden (geparkte Flugzeuge auf dem Träger). Abgesichert durch `internal/dcs/prefab_test.go`.

- **Asset-Prüfung vor Generierung:** Der Missionsplan wird gegen `assets/`-Katalog validiert („Ist Einheit X zur Epoche Y auf Karte Z verfügbar? Für welche Fraktion?"). DCS: zusätzlich Modul-Besitz-Flag.
- **TTS/Audio (optional, Phase 2):** ElevenLabs → OGG → als Radio-Sound einbetten (IL-2: Subtitle-Texte zuerst; DCS: `mapResource` + `a_out_sound`).
- **Determinismus:** LLM liefert nur den **Missionsplan (JSON, schemavalidiert)**; die Datei-Erzeugung macht deterministischer Go-Code (kein LLM-Output direkt in Dateien). Damit bleibt alles reproduzierbar und debuggbar.

## 3. Backend-Schnittstellen (Wails-Bindings, Entwurf)

```go
type Project struct { ... }                    // ein Missions-Projekt (5 Prompts + Plan + Outputs)

func (a *App) LoadProject(path string) error
func (a *App) SaveProject(path string) error
func (a *App) SetRoleInput(role, text string) error   // eines der 5 Felder
func (a *App) MergePlan() (State, error)              // 5 Felder → ein Prompt → Plan
func (a *App) QuickMission(input string) (State, error)
func (a *App) SetPlanJSON(text string) (State, error) // Plan von Hand editiert
func (a *App) GenerateMission() (State, error)        // Plan → Spielordner (Game aus Plan)
func (a *App) OpenOutputFolder() error
func (a *App) OpenIL2Editor() error                   // IL-2 Editor mit CWD bin\editor
func (a *App) GeneratePrefab(input string) (prefab.Prefab, error)   // Prefab-Entwurf per LLM (noch nicht gespeichert)
func (a *App) SavePrefab(json string) (State, error)                // in prefabs/<id>.json ablegen
func (a *App) DeletePrefab(id string) (State, error)
func (a *App) PlacePrefab(id, side string, lat, lon, heading float64, country string) (State, error)
```

## 4. Konfiguration & Geheimnisse

- `.env` im Projektordner (siehe `reference/.env.example`); Keys **niemals** in Projekte/Repos commiten.
- App liest Pfade (IL-2 Editor, DCS Saved Games, IL-2 Missionsordner) aus `.env`, mit GUI-Settings-Fallback.

## 5. Roadmap

1. **MVP:** Projekt-Verwaltung (5 Prompts), Merge → Missionsplan-JSON, IL-2-Generator (Objekte + Objective + Subtitle, wie in der Praxis-Mission), Validierung, „Im Editor öffnen".
2. **DCS:** .miz-Roundtrip aus einem Referenz-Template (leere Basismission je Karte), dictionary/mapResource-Briefings, Einheitenplatzierung aus Plan.
3. **Komfort:** TTS-Radio (ElevenLabs → OGG), Kneeboard-Bilder, Karten-Vorschau, Asset-DB-Editor, Historien-/Lore-Bibliothek.
