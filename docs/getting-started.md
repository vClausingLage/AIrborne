# Airborne – Erste Schritte

Airborne erzeugt aus einer kurzen Beschreibung eine spielbare Mission für **IL-2 Korea** oder **DCS World**. Die KI liefert nur einen Missionsplan (JSON); die eigentlichen Missionsdateien schreibt Airborne selbst – deterministisch, nachvollziehbar, direkt in deinen Spielordner.

---

## 1. Voraussetzungen

| Was | Wozu |
|---|---|
| Windows 10/11 | Airborne ist eine Desktop-App (Wails); die Spiele laufen ohnehin nur dort |
| IL-2 Korea und/oder DCS World | mindestens eines der beiden Spiele installiert |
| Ein LLM-Zugang (OpenAI-kompatibel) | z. B. OpenAI, OpenRouter oder lokal via Ollama |
| Nur zum Selberbauen: Go ≥ 1.22, Node ≥ 18, Wails CLI v2 | nicht nötig, wenn du eine fertige `airborne.exe` hast |

## 2. Installation

**Fertige Exe:** `airborne.exe` in den Projektordner legen (dort, wo `prompts/` liegt) oder von dort starten – Airborne sucht `prompts/`, `prefabs/`, `projects/` und `logs/` relativ zu diesem Ordner.

**Selbst bauen:**

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd app
wails build            # → app\build\bin\airborne.exe
```

Zum Entwickeln stattdessen `wails dev` (Browser-Ansicht mit Go-Bindings unter http://localhost:34115).

## 3. Konfiguration (`.env`)

Kopiere `reference/.env.example` nach `.env` im Projektordner (oder nach `app\.env`) und trage ein:

```ini
# LLM – OpenAI-kompatibel
OPENAI_API_BASE=https://openrouter.ai/api/v1      # oder https://api.openai.com/v1, http://localhost:11434/v1 (Ollama)
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-5.3-flash                          # Modellname deines Providers
OPENAI_MAX_TOKENS=30000                             # Reasoning-Modelle brauchen Luft für Denken + Antwort

# IL-2 Korea
IL2_EDITOR=C:\Program Files (x86)\Steam\steamapps\common\IL2Series\bin\editor\IL2Editor.exe
IL2_MISSIONS_DIR=C:\Users\<du>\OneDrive\Missionen   # Ablageort für generierte .Mission-Dateien

# DCS World
DCS_SAVED_GAMES=C:\Users\<du>\Saved Games\DCS       # Missionen landen in ...\Missions\
```

Ob der Key erkannt wurde, siehst du oben in der Kopfzeile: 🔑 + Modellname. ⚠ bedeutet: kein Key gefunden.

**Der Key gehört nie ins Repo.** `.env` steht in `.gitignore`.

## 4. Die erste Mission in fünf Minuten

1. **Spiel wählen** – oben links **IL-2 Korea** oder **DCS World**. Die Wahl bestimmt Einheitennamen, Koordinatensystem und Zielordner.
2. **Beschreiben** – zwei Wege:
   - **⚡ Blitz**: ein Freitext, Stichpunkte reichen.
     *„Kleine Skirmish auf der Syria Map: MiG-21Bis im Bodenangriff, am Boden ein APC-Verband mit Shilka, dazu ein Paar F-5E als Jagdschutz."*
   - **Schritt für Schritt**: fünf Felder (Story, Player, Gegner, Widerstand, Briefing). Leere Felder ergänzt die KI passend zur Story. Alle Felder wandern gemeinsam in *einen* Prompt – es gibt keine fünf Einzelaufrufe.
3. **Generieren** – „Skirmish generieren" bzw. „Missionsplan erstellen". Je nach Modell 20–90 Sekunden.
4. **Plan prüfen** – Tab **Plan & Export**. Links: Validierung (⚠ Hinweise sind Warnungen, keine Blocker) und Spiel/Karte/Datum. Rechts: der Plan als JSON – du kannst ihn direkt editieren (Anzahl, Positionen, Funktexte …) und mit „Änderungen übernehmen" bestätigen.
5. **Mission erzeugen** – Button „Mission erzeugen". Airborne schreibt die Dateien in den Spielordner und zeigt Pfad + Hinweise. Alle generierten Missionen heißen `AB_<Titel>`, damit nie eine handgebaute Mission überschrieben wird.
6. **Im Spiel öffnen**
   - **IL-2**: „IL-2 Editor starten", Mission öffnen, einmal speichern (erzeugt die `.msnbin`), dann im Spiel unter *Missionen* fliegen.
   - **DCS**: Mission liegt in `Saved Games\DCS\Missions`. Im Missionseditor öffnen, Bewaffnung/Flugplatz kurz prüfen, speichern, fliegen.

Das Projekt (Eingaben + Plan + letzter Export) wird automatisch in `projects/current.json` gesichert. Mit **Speichern/Laden** kannst du Projekte unter eigenem Namen ablegen.

## 5. Prefabs (nur DCS): wiederverwendbare Asset-Gruppen

Ein Prefab ist eine fertige Gruppe von Schiffen, Fahrzeugen, geparkten Flugzeugen und Gebäuden – z. B. ein Trägerverband mit Deckflugzeugen oder ein FARP mit Zelten und Tankwagen. Einmal gebaut, beliebig oft platzierbar, auf jeder Karte.

Der Modus **🧩 Prefabs** erscheint neben Blitz/Schritt für Schritt, sobald DCS gewählt ist.

**Prefab bauen**
1. Links beschreiben, was drin sein soll:
   *„Trägerkampfgruppe: CVN-73 mit zwei Burke-Zerstörern, auf dem Deck sechs geparkte F/A-18C und ein E-2C."*
2. „Prefab entwerfen" – die KI liefert das Prefab als JSON. Positionen sind **relativ in Metern** um den Ursprung (`dx` = Nord/vorwärts, `dy` = Ost/rechts, `heading` relativ).
3. Bei Bedarf im JSON nachjustieren (Abstände, Typen, Anzahl), dann **In Bibliothek speichern**. Die Datei landet in `prefabs/<id>.json` – du kannst sie auch von Hand anlegen oder mit anderen teilen.

**Prefab in eine Mission bringen** – zwei Wege:
- **Automatisch:** gespeicherte Prefabs werden der Missions-KI bei Blitz/Schritt für Schritt mitgeteilt. Schreibst du z. B. *„… vor der Küste liegt unser Träger"*, platziert sie den Trägerverband selbst.
- **Manuell:** rechts im Prefab-Modus ein Prefab wählen, Seite (verbündet/Gegner), Breite/Länge (Dezimalgrad) und Ausrichtung eingeben → „Prefab in Plan einfügen". Der Eintrag erscheint im Plan unter `prefabs`.

Beim Export dreht Airborne das Prefab um seinen Ursprung, macht aus Schiffen/Fahrzeugen DCS-Gruppen und aus Gebäuden/geparkten Flugzeugen echte Statics. Flugzeuge mit `linkTo` (z. B. auf dem Träger) werden an das Schiff gekoppelt und fahren mit.

Zwei Beispiele liegen bereit: `prefabs/carrier-strike-group.json` (US-Trägerverband) und `prefabs/mountain-farp.json` (russischer FARP).

## 6. Tipps für gute Ergebnisse

- **Epoche und Ort nennen.** „Juni 1982, Bekaa-Tal" liefert passende Einheiten; ohne Angabe rät die KI.
- **Wenige, konkrete Ziele.** Eine Skirmish mit 1–3 Gegnergruppen und einem klaren Auftrag ist spielbarer als eine Großoffensive.
- **DCS: Modulbesitz.** Die KI kennt deine gekauften Module nicht. Nenne dein Flugzeug im Text („… ich fliege die F-16C").
- **Plan-JSON ist dein Freund.** Kleine Korrekturen (Position, Anzahl, Callsign) gehen dort schneller als ein neuer KI-Lauf.
- **Ohne GUI** (Skripte, Batch): `cd app && go run ./cmd/abgen -plan ../projects/current.json [-out <Ordner>]`.

## 7. Wenn etwas nicht klappt

| Symptom | Ursache / Lösung |
|---|---|
| ⚠ statt 🔑 in der Kopfzeile | `.env` nicht gefunden oder `OPENAI_API_KEY` leer. `.env` muss im Projektordner (neben `prompts/`) oder in `app\` liegen. |
| „Antwort enthält kein JSON" / „Missionsplan-JSON ungültig" | Modell hat abgebrochen oder geplaudert. `OPENAI_MAX_TOKENS` erhöhen, kleineres/anderes Modell, oder einfach nochmal generieren. Rohantwort steht in `logs/airborne.log`. |
| „IL2_MISSIONS_DIR ist nicht gesetzt" / „DCS_SAVED_GAMES ist nicht gesetzt" | Pfad in `.env` eintragen; Anführungszeichen sind erlaubt, Backslashes bleiben Backslashes. |
| IL-2 Editor startet mit Paketfehler | Immer über „IL-2 Editor starten" öffnen – der Editor braucht `bin\editor` als Arbeitsverzeichnis. |
| DCS lädt die Mission nicht | `Saved Games\DCS\Logs\dcs.log` ansehen. Meist ein unbekannter Einheiten-Typ im Plan-JSON – Name korrigieren, „Änderungen übernehmen", neu exportieren. |
| Prefab: Flugzeuge stehen im Wasser statt auf dem Träger | Dem Element im Prefab-JSON `"linkTo": "<Name des Schiff-Elements>"` geben. |
| Prefab-Hinweis „Static-Typ nicht im Katalog" | Typ ist unbekannt und wird als Fahrzeug platziert. Die bekannten Gebäude/Objekte stehen in `app/internal/prefab/catalog.go`; alternativ `category` und `shapeName` im JSON selbst setzen. |
| „Projekt-Root nicht gefunden (prompts/ fehlt)" | Airborne aus dem Projektordner starten oder die Exe dorthin legen. |

Jeder KI-Aufruf (Prompt + Rohantwort + normalisierter Plan) wird nach `logs/airborne.log` geschrieben – der erste Anlaufpunkt bei Problemen. Der Pfad steht unten im Tab „Plan & Export".

## 8. Wo liegt was

```
AIrborne/
  .env                 deine Keys und Pfade (nicht committen)
  prompts/             die Prompt-Vorlagen – anpassbar, wirken sofort nach Neustart
  prefabs/             deine Prefab-Bibliothek (eine JSON je Prefab)
  projects/            gespeicherte Projekte (current.json = zuletzt bearbeitet)
  logs/airborne.log    alle KI-Aufrufe und Generator-Hinweise
  docs/                Formate und Hintergründe (IL-2 .Mission, DCS .miz, Architektur)
```
