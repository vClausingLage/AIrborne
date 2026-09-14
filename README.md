# AIrborne

KI-gestützte Meta-App zur Erstellung realistischer, immersiver Missionen für **IL-2 Korea** und **DCS World** (Go + Wails v2 + React/TypeScript).

**Neu hier? → [docs/getting-started.md](docs/getting-started.md)** – Installation, `.env`, erste Mission, Prefabs, Fehlersuche.

## Dokumentation

| Dok | Inhalt |
|---|---|
| `docs/getting-started.md` | **Nutzer-Anleitung**: Setup, erste Mission, Prefabs, Troubleshooting |
| `docs/il2-korea-missions.md` | IL-2: Missionsformate (.Mission/.msnbin/.ger), MCU-Katalog, Assets, Workflow, Stolperfallen (praxisverifiziert) |
| `docs/dcs-miz-workflow.md` | DCS: .miz entpacken/editieren/zippen, mission-Struktur, dictionary/mapResource, Gotchas |
| `docs/architecture.md` | Tech-Stack (Go+Wails+React/TS), Pipeline, Schnittstellen, Roadmap |
| `prompts/06-merge.md` | Schritt-für-Schritt: strukturierter Prompt (# Story / # Player / # Gegner / # Widerstand / # Briefing aus den 5 Nutzer-Feldern) → Missionsplan-JSON |
| `prompts/07-quick.md` | Blitz-Skirmish: ein Freitext → Missionsplan-JSON |
| `prompts/08-prefab.md` | Prefab-Builder (DCS): Beschreibung → Asset-Gruppe als JSON (relative Meter, Statics-Katalog) |
| `prompts/schema.md` | Gemeinsames JSON-Schema + Feldregeln (IL-2 / DCS), wird in beide Prompts eingesetzt |
| `reference/.env.example` | API-Keys (LLM, ElevenLabs) + alle Pfade (IL-2 Editor, DCS, Missionsordner) |
| `prefabs/*.json` | Prefab-Bibliothek (Trägerverband, FARP, …) – wird in den DCS-Spiel-Kontext eingeblendet und per `plan.prefabs` platziert |

## Schnellstart

```powershell
cd app
wails dev        # Entwicklung (Browser-Ansicht unter http://localhost:34115)
wails build      # build/bin/airborne.exe
```

`.env` nach `reference/.env.example` anlegen (LLM-Key, `IL2_MISSIONS_DIR`, `DCS_SAVED_GAMES`, `IL2_EDITOR`).

## Prinzip

1. **Eingabe** – entweder ⚡ Blitz (ein Freitext) oder Schritt für Schritt (5 Felder: Story, Player, Gegner, Widerstand, Briefing). Die Felder werden **nicht** einzeln an die KI geschickt, sondern in *einen* strukturierten Prompt eingebettet (`prompts/06-merge.md`).
2. **Missionsplan (JSON)** – die KI liefert ausschließlich den schemavalidierten Plan (`prompts/schema.md`); er ist im Tab „Plan & Export“ editierbar.
3. **Generieren & Exportieren** – deterministische Go-Generatoren schreiben die Missionsdateien direkt in den Spielordner:
   - IL-2 Korea (`app/internal/il2`): `AB_<Titel>.Mission` + `.ger/.eng/.rus/.fra/.spa/.chs` (UTF-16LE) + `.list` nach `IL2_MISSIONS_DIR`, anschließend „IL-2 Editor starten“ (CWD `bin\editor`).
   - DCS World (`app/internal/dcs`): `AB_<Titel>.miz` (mission/options/theatre/warehouses/l10n) nach `DCS_SAVED_GAMES\Missions`; lat/lon → Kartenkoordinaten per Transverse-Mercator je Theatre.

4. **Prefabs (DCS)** – im Modus 🧩 Prefabs wiederverwendbare Asset-Gruppen (Schiffe, Fahrzeuge, geparkte Flugzeuge, Statics) per Freitext entwerfen, in `prefabs/` speichern und manuell oder durch die Missions-KI in Pläne setzen; der DCS-Generator dreht sie um den Ursprung und erzeugt echte Static-/Schiffsgruppen inkl. Deck-Kopplung (`linkTo`).

Ohne GUI: `cd app && go run ./cmd/abgen -plan ../projects/current.json [-out <Ordner>]`.
