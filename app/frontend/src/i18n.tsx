import { createContext, useContext } from 'react'

// UI texts in German and English. The backend switches its own messages
// (errors, validation, generator notes) via SetLanguage; everything rendered
// by the frontend comes from here. `en` is typed as `typeof de`, so a missing
// or mistyped translation is a compile error.

export type Lang = 'de' | 'en'

const de = {
  locale: 'de-DE',
  loading: 'Lade…',

  // top bar
  noModel: 'kein Modell',
  tokensPerPlan: (n: string) => `max ${n} Tokens/Plan`,
  settings: '⚙ Einstellungen',
  settingsTitle: 'Einstellungen: Token-Limit pro Plan, Modell, Pfade, Sprache',
  newProject: 'Neu',
  newProjectTitle: 'Eingaben, Plan und Export verwerfen',
  newProjectConfirm: 'Neues Projekt: alle Eingaben, den Missionsplan und das Export-Ergebnis verwerfen?',
  save: 'Speichern',
  load: 'Laden',
  saveProjectDialog: 'Projekt speichern',
  openProjectDialog: 'Projekt öffnen',
  languageTitle: 'Sprache der Oberfläche',

  // flow
  stageInput: '1 · Eingabe',
  stageResult: '2 · Ergebnis',
  stageEdit: '3 · Bearbeiten',
  modeQuick: '⚡ Blitz',
  modeSteps: '🪜 Schritt für Schritt',
  modeChallenge: '🎯 Herausforderung',
  modeChallengeTitle: 'Nur das Flugzeug wählen – Gegner bleiben verborgen',
  modePrefabs: '🧩 Prefabs',
  modePrefabsTitle: 'Wiederverwendbare Asset-Gruppen für DCS bauen',
  planExport: 'Plan & Export',
  badgeExported: 'exportiert',
  badgeChallenge: 'Herausforderung',
  badgeReady: 'Plan bereit',
  editMission: '🛠 Mission bearbeiten',
  editMissionTitle: 'Bestehende .miz / .Mission ohne KI öffnen und ändern',

  // step by step
  merge: 'Zusammenführen',
  mergeHint: (
    <>
      Deine fünf Abschnitte werden in einen strukturierten Prompt eingebettet
      (<code>prompts/06-merge.md</code>: „Deine Aufgabe ist es, eine Mission zu erstellen … # Story … # Player …“)
      und in einem Durchgang zum Missionsplan (JSON) verarbeitet. Leere Abschnitte entscheidet die KI passend zur Story.
    </>
  ),
  merging: 'Erstellt Missionsplan…',
  mergeButton: 'Missionsplan erstellen',

  // blitz
  quickHeading: '⚡ Blitz-Skirmish – ein Feld, fertiger Plan',
  quickPlaceholder:
    'Kurz beschreiben, was du willst. Beispiel:\n\n' +
    'Kleine Skirmish auf der Syria Map: MiG-21Bis mit Raketen im Bodenangriff, ' +
    'dazu Mi-24P und Mi-8MT, am Boden ein Infanterieverband mit APC und Artillerie, ' +
    'der sich zum Angriff formiert.',
  quickBusy: 'Generiert Missionsplan…',
  quickButton: 'Skirmish generieren',
  quickHint: (
    <>
      1. Oben das Zielspiel wählen (IL-2 / DCS)
      <br />
      2. Wunsch eintippen – Stichpunkte reichen
      <br />
      3. Generieren → unter „Plan &amp; Export“ prüfen und Mission erzeugen
    </>
  ),

  // prefab builder
  prefabHeading: '🧩 Prefab-Builder – Asset-Gruppe beschreiben',
  prefabPlaceholder:
    'Beispiel:\n\n' +
    'Trägerkampfgruppe: CVN-73 mit zwei Burke-Zerstörern als Eskorte, ' +
    'auf dem Deck sechs geparkte F/A-18C, ein E-2C und Container am Heck.\n\n' +
    'Oder: FARP im Gebirge mit vier Landeplätzen, Zelten, Tanklager, ' +
    'zwei Ural-Tankwagen und einem geparkten Mi-8.',
  prefabBusy: 'Entwirft Prefab…',
  prefabButton: 'Prefab entwerfen',
  prefabDraft: 'Entwurf (JSON, editierbar)',
  saving: 'Speichert…',
  saveToLibrary: 'In Bibliothek speichern',
  prefabDraftPlaceholder:
    '(noch kein Entwurf – oben beschreiben und „Prefab entwerfen“ klicken, oder rechts ein Prefab zum Bearbeiten laden)',
  prefabDraftHint: (
    <>
      Positionen sind relativ in Metern (dx = Nord, dy = Ost) um den Prefab-Ursprung; beim Platzieren wird das Prefab an lat/lon
      gedreht. Geparkte Flugzeuge auf Schiffen brauchen <code>linkTo</code>.
    </>
  ),
  library: (n: number) => `Bibliothek (${n})`,
  folder: 'Ordner',
  prefabFolderTitle: 'prefabs/ im Explorer öffnen',
  libraryEmpty: 'Noch keine Prefabs – links eines entwerfen und speichern.',
  edit: 'Bearbeiten',
  delete: 'Löschen',
  deletePrefabConfirm: (name: string) => `Prefab „${name}“ löschen?`,
  insertIntoPlan: 'In Missionsplan einfügen',
  insertNoPlan: 'Erst einen Plan erzeugen (Blitz / Schritt für Schritt) – die KI kann gespeicherte Prefabs dabei auch selbst platzieren.',
  side: 'Seite',
  choose: '– wählen –',
  sideFriendly: 'Verbündet (Spielerseite)',
  sideEnemy: 'Gegner',
  countryFromPrefab: '(aus Prefab)',
  placeMissing: 'Prefab, Breite (lat) und Länge (lon) angeben',
  placing: 'Fügt ein…',
  placeButton: 'Prefab in Plan einfügen',

  // plan & export
  missionPlan: 'Missionsplan',
  noPlanYet: 'Noch kein Plan – Blitz oder „Schritt für Schritt“ ausführen.',
  planValid: '✓ Plan validiert',
  repairSummary: (found: number) => `🔧 Reparatur: ${found} Problem${found === 1 ? '' : 'e'} gemeldet · `,
  repairFixed: (fixed: number, open: number) => `${fixed} behoben${open > 0 ? `, ${open} offen` : ''}`,
  repairKept: 'erster Plan behalten',
  repairHint: 'Validierung + Probelauf des Generators nach jeder KI-Antwort; bei Problemen ein zweiter KI-Aufruf mit der Fehlerliste.',
  game: 'Spiel',
  map: 'Karte',
  challengeHidden: '(Gegner verborgen)',
  export: 'Export',
  targetFolder: 'Zielordner',
  notSetEnv: '(nicht gesetzt – .env prüfen)',
  applyJsonFirst: 'Erst JSON-Änderungen übernehmen',
  generating: 'Erzeugt Mission…',
  generateDcs: 'Mission erzeugen → .miz exportieren',
  generateIl2: 'Mission erzeugen → IL-2 Ordner',
  openFolder: 'Ordner öffnen',
  startIl2Editor: 'IL-2 Editor starten',
  startIl2EditorTitle: 'Startet den IL-2 Editor mit Arbeitsverzeichnis bin\\editor',
  il2AfterExport: 'Im Editor öffnen, Höhen/Placement prüfen und speichern (erzeugt .msnbin).',
  dcsAfterExport: 'DCS: Mission liegt in Saved Games\\DCS\\Missions – im Editor öffnen, Bewaffnung/Flugplatz prüfen, speichern.',
  project: 'Projekt',
  log: 'Log',
  notInitialized: '(nicht initialisiert)',
  changed: '(geändert)',
  checking: 'Prüft…',
  applyChanges: 'Änderungen übernehmen',
  noPlanPlaceholder: '(noch kein Plan)',

  // challenge
  challengeHeading: '🎯 Herausforderung – nur das Flugzeug wählen',
  aircraft: 'Flugzeug',
  groupBlueDcs: 'Blau / West',
  groupBlueIl2: 'UN / USA',
  groupRedDcs: 'Rot / Ost',
  groupRedIl2: 'KPAF / VVS',
  groupAny: 'Beide Seiten',
  otherModule: 'Anderes Modul (Typname eingeben)…',
  dcsType: 'DCS-Typ',
  dcsTypePlaceholder: 'z. B. "A-4E-C"',
  aiChooses: 'KI wählt passend',
  challengeBusy: 'Entwirft Herausforderung…',
  challengeButton: 'Herausforderung generieren',
  howItWorks: 'So funktioniert es',
  challengeExplain: (
    <>
      Die KI entwirft aus deinem Flugzeug allein eine kleine, fordernde Mission: Epoche, Fraktion, Karte, Wetter, Auftrag und
      Gegner wählt sie selbst – und zwar so, dass das Flugzeug gefordert wird (Jäger: Überzahl und Positionsnachteil plus Flak,
      Bodenangriff: verteidigte Ziele mit Jagdschutz in der Nähe, Hubschrauber: mobile Luftabwehr).
      <br />
      <br />
      <b>Alle Gegnerinformationen bleiben verborgen.</b> Briefing, Missionsziel und Funksprüche nennen weder Typen noch Anzahl
      noch Positionen; Karten-Icons entfallen. Der Plan wird zusätzlich deterministisch geprüft: Erscheint ein Gegnertyp in einem
      Spielertext, meldet „Plan &amp; Export“ eine Warnung.
    </>
  ),
  challengeExplainDcs: (
    <>
      <b>DCS:</b> Alle Gegnergruppen werden „Hidden on map/planner/MFD“ exportiert, und die F10-Karte wird per{' '}
      <code>forcedOptions</code> auf das eigene Flugzeug beschränkt.
    </>
  ),
  challengeExplainAfter:
    'Nach dem Generieren unter „Plan & Export“ prüfen und exportieren. Der Plan enthält die Gegner natürlich vollständig – wer nicht spoilern will, klappt das JSON nicht auf.',

  // media
  mediaHeading: 'Medien (ohne KI)',
  media: 'Medien',
  briefingImage: 'Briefing-Bild',
  none: '(keins)',
  pick: 'Wählen',
  pickBriefingDialog: 'Briefing-Bild wählen (PNG/JPG)',
  pickKneeboardDialog: 'Kneeboard-Seiten wählen (PNG/JPG)',
  briefingImageHintDcs: 'Wird als l10n/DEFAULT/<name>.png in die .miz gepackt und über mapResource als Briefing-Bild der Spielerseite verknüpft.',
  briefingImageHintIl2: 'Wird als <Mission>.png neben die .Mission gelegt (Konvention der mitgelieferten Kampagnen-Missionen).',
  addPages: 'Seiten hinzufügen',
  kneeboardBriefing: 'Briefing-Text als Seite 1 rendern',
  kneeboardHint: 'Landet in KNEEBOARD/IMAGES der .miz (alle Flugzeuge). PNG oder JPG, Hochformat 3:4 empfohlen.',

  // settings
  settingsAi: '⚙ Einstellungen · KI',
  maxTokens: 'Max. Tokens pro Plan',
  apply: 'Übernehmen',
  resetToEnvTitle: 'Zurück auf AI_MAX_TOKENS aus .env',
  defaultN: (n: string) => `Standard (${n})`,
  maxTokensInvalid: 'Ganze Zahl zwischen 1.000 und 1.000.000 (oder 0 = Standard aus .env)',
  maxTokensHint: (
    <>
      Obergrenze für die <b>Ausgabe</b>-Tokens <b>einer</b> LLM-Anfrage (<code>max_tokens</code>). Ein Missionsplan ist genau eine
      Anfrage, das Limit gilt also pro Plan (Blitz, Schritt für Schritt, Herausforderung, Prefab). Bei Reasoning-Modellen zählt das
      Nachdenken mit – zu klein → leere/abgeschnittene Antwort, zu groß → höchstens teurer, nie besser.
    </>
  ),
  maxTokensUsesEnv: ' Aktuell gilt der Standard aus .env (AI_MAX_TOKENS).',
  maxTokensEnvDefault: (n: string) => ` Standard aus .env: ${n}.`,
  lastRequest: 'Letzte Anfrage',
  usage: (p: string, c: string, total: string) => `Eingabe ${p} · Ausgabe ${c} · gesamt ${total} Tokens`,
  usageShare: (pct: number) => (
    <>
      {' '}
      · Ausgabe nutzt <b>{pct}%</b> des Limits
    </>
  ),
  noRequestYet: '(noch keine Anfrage in dieser Sitzung)',
  language: 'Sprache / Language',
  languageHint: 'Oberfläche, Fehlermeldungen und Hinweise. DCS-Missionstexte stehen in dieser Sprache zuerst, die andere folgt darunter.',
  fromEnv: 'Aus .env (nur lesend)',
  model: 'Modell',
  modelMissing: '(AI_MODEL fehlt)',
  apiKey: 'API-Key',
  keySet: '🔑 gesetzt',
  keyMissing: '⚠ AI_API_KEY fehlt',
  il2Missions: 'IL-2 Missionen',
  il2MissionsMissing: '(IL2_MISSIONS_DIR fehlt)',
  dcsMissions: 'DCS Missionen',
  dcsMissionsMissing: '(DCS_SAVED_GAMES fehlt)',
  settingsFile: 'Einstellungen',

  // mission editor
  discardUnsaved: 'Ungespeicherte Änderungen verwerfen?',
  discardText: 'Textänderungen an der aktuellen Datei verwerfen?',
  removeEntryConfirm: (name: string) => `„${name}“ aus der Mission entfernen?`,
  opening: 'Öffnet…',
  openMission: 'Mission öffnen (.miz / .Mission)',
  openLastExport: 'Zuletzt exportierte Mission öffnen',
  pastePath: '…oder Pfad einfügen',
  open: 'Öffnen',
  editorIntro: (
    <>
      Öffnet bestehende Missionen ohne KI: <b>DCS .miz</b> (ZIP mit mission/options/dictionary…) oder <b>IL-2 .Mission</b> samt
      Sprachdateien. Titel, Briefing, Datum/Zeit lassen sich im Formular ändern, Briefing-Bild und Kneeboards anhängen und jede
      Textdatei direkt bearbeiten. Alles andere bleibt Byte für Byte erhalten; beim ersten Speichern wird eine <code>.bak</code>-Kopie
      angelegt.
    </>
  ),
  changedSuffix: ' (geändert)',
  close: 'Schließen',
  protectedMission: '⚠ Geschützte Mission – nur Bilder/Kneeboards änderbar',
  applyTextFirst: 'Erst Text-/Formularänderungen übernehmen',
  saveAs: 'Speichern unter…',
  metadata: 'Metadaten',
  title: 'Titel',
  author: 'Autor',
  date: 'Datum',
  time: 'Zeit',
  addImages: 'Bilder hinzufügen',
  textPageHint: 'Textseite erzeugen (z. B. Briefing, Funkplan):',
  heading: 'Überschrift',
  kneeboardTextPlaceholder: 'Text der Kneeboard-Seite…',
  takeBriefing: 'Briefing übernehmen',
  addAsPage: 'Als Seite hinzufügen',
  files: (n: number) => `Dateien (${n})`,
  editAsText: 'Als Text bearbeiten',
  binaryFile: 'Binärdatei',
  text: 'Text',
  applying: 'Übernimmt…',
  bigFile: (size: string) => `Große Datei (${size}) – der Editor kann träge reagieren.`,
  pickTextFile: '(links eine Textdatei wählen)',
}

export type Messages = typeof de

const en: Messages = {
  locale: 'en-US',
  loading: 'Loading…',

  noModel: 'no model',
  tokensPerPlan: (n: string) => `max ${n} tokens/plan`,
  settings: '⚙ Settings',
  settingsTitle: 'Settings: token limit per plan, model, paths, language',
  newProject: 'New',
  newProjectTitle: 'Discard inputs, plan and export',
  newProjectConfirm: 'New project: discard all inputs, the mission plan and the export result?',
  save: 'Save',
  load: 'Load',
  saveProjectDialog: 'Save project',
  openProjectDialog: 'Open project',
  languageTitle: 'Interface language',

  stageInput: '1 · Input',
  stageResult: '2 · Result',
  stageEdit: '3 · Edit',
  modeQuick: '⚡ Blitz',
  modeSteps: '🪜 Step by step',
  modeChallenge: '🎯 Challenge',
  modeChallengeTitle: 'Pick only the aircraft – the enemy stays hidden',
  modePrefabs: '🧩 Prefabs',
  modePrefabsTitle: 'Build reusable asset groups for DCS',
  planExport: 'Plan & Export',
  badgeExported: 'exported',
  badgeChallenge: 'Challenge',
  badgeReady: 'plan ready',
  editMission: '🛠 Edit mission',
  editMissionTitle: 'Open and change an existing .miz / .Mission without AI',

  merge: 'Merge',
  mergeHint: (
    <>
      Your five sections are embedded into one structured prompt (<code>prompts/06-merge.md</code>: a mission-building task with
      sections # Story, # Player, …) and turned into the mission plan (JSON) in a single pass. The AI fills empty sections to match
      the story.
    </>
  ),
  merging: 'Creating mission plan…',
  mergeButton: 'Create mission plan',

  quickHeading: '⚡ Blitz skirmish – one field, finished plan',
  quickPlaceholder:
    'Briefly describe what you want. Example:\n\n' +
    'Small skirmish on the Syria map: MiG-21Bis with rockets in ground attack, ' +
    'plus Mi-24P and Mi-8MT; on the ground an infantry unit with APCs and artillery ' +
    'forming up for an assault.',
  quickBusy: 'Generating mission plan…',
  quickButton: 'Generate skirmish',
  quickHint: (
    <>
      1. Choose the target game at the top (IL-2 / DCS)
      <br />
      2. Type what you want – bullet points are enough
      <br />
      3. Generate → review under “Plan &amp; Export” and create the mission
    </>
  ),

  prefabHeading: '🧩 Prefab builder – describe an asset group',
  prefabPlaceholder:
    'Example:\n\n' +
    'Carrier strike group: CVN-73 with two Burke destroyers as escort, ' +
    'six parked F/A-18Cs on deck, an E-2C and containers at the stern.\n\n' +
    'Or: mountain FARP with four landing pads, tents, a fuel depot, ' +
    'two Ural fuel trucks and a parked Mi-8.',
  prefabBusy: 'Designing prefab…',
  prefabButton: 'Design prefab',
  prefabDraft: 'Draft (JSON, editable)',
  saving: 'Saving…',
  saveToLibrary: 'Save to library',
  prefabDraftPlaceholder:
    '(no draft yet – describe it above and click “Design prefab”, or load a prefab on the right to edit it)',
  prefabDraftHint: (
    <>
      Positions are relative, in metres (dx = north, dy = east) around the prefab origin; when placed, the prefab is rotated at
      lat/lon. Aircraft parked on ships need <code>linkTo</code>.
    </>
  ),
  library: (n: number) => `Library (${n})`,
  folder: 'Folder',
  prefabFolderTitle: 'Open prefabs/ in Explorer',
  libraryEmpty: 'No prefabs yet – design one on the left and save it.',
  edit: 'Edit',
  delete: 'Delete',
  deletePrefabConfirm: (name: string) => `Delete prefab “${name}”?`,
  insertIntoPlan: 'Insert into mission plan',
  insertNoPlan: 'Create a plan first (Blitz / Step by step) – the AI can also place saved prefabs on its own.',
  side: 'Side',
  choose: '– choose –',
  sideFriendly: 'Friendly (player side)',
  sideEnemy: 'Enemy',
  countryFromPrefab: '(from prefab)',
  placeMissing: 'Enter prefab, latitude (lat) and longitude (lon)',
  placing: 'Inserting…',
  placeButton: 'Insert prefab into plan',

  missionPlan: 'Mission plan',
  noPlanYet: 'No plan yet – run Blitz or “Step by step”.',
  planValid: '✓ Plan validated',
  repairSummary: (found: number) => `🔧 Repair: ${found} problem${found === 1 ? '' : 's'} reported · `,
  repairFixed: (fixed: number, open: number) => `${fixed} fixed${open > 0 ? `, ${open} open` : ''}`,
  repairKept: 'first plan kept',
  repairHint: 'Validation + generator dry run after every AI answer; if problems are found, a second AI call with the problem list.',
  game: 'Game',
  map: 'Map',
  challengeHidden: '(enemy hidden)',
  export: 'Export',
  targetFolder: 'Target folder',
  notSetEnv: '(not set – check .env)',
  applyJsonFirst: 'Apply the JSON changes first',
  generating: 'Creating mission…',
  generateDcs: 'Create mission → export .miz',
  generateIl2: 'Create mission → IL-2 folder',
  openFolder: 'Open folder',
  startIl2Editor: 'Start IL-2 editor',
  startIl2EditorTitle: 'Starts the IL-2 editor with working directory bin\\editor',
  il2AfterExport: 'Open it in the editor, check altitudes/placement and save (creates the .msnbin).',
  dcsAfterExport: 'DCS: the mission is in Saved Games\\DCS\\Missions – open it in the editor, check loadouts/airfield, save.',
  project: 'Project',
  log: 'Log',
  notInitialized: '(not initialized)',
  changed: '(changed)',
  checking: 'Checking…',
  applyChanges: 'Apply changes',
  noPlanPlaceholder: '(no plan yet)',

  challengeHeading: '🎯 Challenge – pick only the aircraft',
  aircraft: 'Aircraft',
  groupBlueDcs: 'Blue / West',
  groupBlueIl2: 'UN / USA',
  groupRedDcs: 'Red / East',
  groupRedIl2: 'KPAF / VVS',
  groupAny: 'Both sides',
  otherModule: 'Other module (enter type name)…',
  dcsType: 'DCS type',
  dcsTypePlaceholder: 'e.g. "A-4E-C"',
  aiChooses: 'AI picks a fitting one',
  challengeBusy: 'Designing challenge…',
  challengeButton: 'Generate challenge',
  howItWorks: 'How it works',
  challengeExplain: (
    <>
      From your aircraft alone, the AI designs a small, demanding mission: it picks era, faction, map, weather, task and enemy
      itself – so that the aircraft is put to the test (fighters: outnumbered and at a positional disadvantage plus flak, ground
      attack: defended targets with fighter cover nearby, helicopters: mobile air defence).
      <br />
      <br />
      <b>All enemy information stays hidden.</b> Briefing, objective and radio messages name neither types nor numbers nor
      positions; map icons are dropped. The plan is also checked deterministically: if an enemy type shows up in a player-facing
      text, “Plan &amp; Export” shows a warning.
    </>
  ),
  challengeExplainDcs: (
    <>
      <b>DCS:</b> all enemy groups are exported “Hidden on map/planner/MFD”, and the F10 map is restricted to your own aircraft
      via <code>forcedOptions</code>.
    </>
  ),
  challengeExplainAfter:
    'After generating, review and export under “Plan & Export”. The plan of course contains the full enemy – if you don’t want spoilers, don’t open the JSON.',

  mediaHeading: 'Media (no AI)',
  media: 'Media',
  briefingImage: 'Briefing image',
  none: '(none)',
  pick: 'Choose',
  pickBriefingDialog: 'Choose briefing image (PNG/JPG)',
  pickKneeboardDialog: 'Choose kneeboard pages (PNG/JPG)',
  briefingImageHintDcs: 'Packed into the .miz as l10n/DEFAULT/<name>.png and linked via mapResource as the player side’s briefing image.',
  briefingImageHintIl2: 'Placed next to the .Mission as <Mission>.png (convention of the bundled campaign missions).',
  addPages: 'Add pages',
  kneeboardBriefing: 'Render briefing text as page 1',
  kneeboardHint: 'Goes into KNEEBOARD/IMAGES of the .miz (all aircraft). PNG or JPG, portrait 3:4 recommended.',

  settingsAi: '⚙ Settings · AI',
  maxTokens: 'Max. tokens per plan',
  apply: 'Apply',
  resetToEnvTitle: 'Back to AI_MAX_TOKENS from .env',
  defaultN: (n: string) => `Default (${n})`,
  maxTokensInvalid: 'Whole number between 1,000 and 1,000,000 (or 0 = default from .env)',
  maxTokensHint: (
    <>
      Upper limit for the <b>output</b> tokens of <b>one</b> LLM request (<code>max_tokens</code>). A mission plan is exactly one
      request, so the limit applies per plan (Blitz, Step by step, Challenge, Prefab). With reasoning models the thinking counts
      too – too small → empty/truncated answer, too large → at most more expensive, never better.
    </>
  ),
  maxTokensUsesEnv: ' The default from .env (AI_MAX_TOKENS) currently applies.',
  maxTokensEnvDefault: (n: string) => ` Default from .env: ${n}.`,
  lastRequest: 'Last request',
  usage: (p: string, c: string, total: string) => `Input ${p} · output ${c} · total ${total} tokens`,
  usageShare: (pct: number) => (
    <>
      {' '}
      · output uses <b>{pct}%</b> of the limit
    </>
  ),
  noRequestYet: '(no request in this session yet)',
  language: 'Sprache / Language',
  languageHint: 'Interface, error messages and hints. DCS mission texts are written in this language first, the other one follows below.',
  fromEnv: 'From .env (read-only)',
  model: 'Model',
  modelMissing: '(AI_MODEL missing)',
  apiKey: 'API key',
  keySet: '🔑 set',
  keyMissing: '⚠ AI_API_KEY missing',
  il2Missions: 'IL-2 missions',
  il2MissionsMissing: '(IL2_MISSIONS_DIR missing)',
  dcsMissions: 'DCS missions',
  dcsMissionsMissing: '(DCS_SAVED_GAMES missing)',
  settingsFile: 'Settings',

  discardUnsaved: 'Discard unsaved changes?',
  discardText: 'Discard text changes to the current file?',
  removeEntryConfirm: (name: string) => `Remove “${name}” from the mission?`,
  opening: 'Opening…',
  openMission: 'Open mission (.miz / .Mission)',
  openLastExport: 'Open last exported mission',
  pastePath: '…or paste a path',
  open: 'Open',
  editorIntro: (
    <>
      Opens existing missions without AI: <b>DCS .miz</b> (ZIP with mission/options/dictionary…) or <b>IL-2 .Mission</b> with its
      language files. Title, briefing and date/time can be changed in the form, a briefing image and kneeboards attached, and every
      text file edited directly. Everything else is kept byte for byte; the first save creates a <code>.bak</code> copy.
    </>
  ),
  changedSuffix: ' (changed)',
  close: 'Close',
  protectedMission: '⚠ Protected mission – only images/kneeboards can be changed',
  applyTextFirst: 'Apply text/form changes first',
  saveAs: 'Save as…',
  metadata: 'Metadata',
  title: 'Title',
  author: 'Author',
  date: 'Date',
  time: 'Time',
  addImages: 'Add images',
  textPageHint: 'Create a text page (e.g. briefing, radio plan):',
  heading: 'Heading',
  kneeboardTextPlaceholder: 'Text of the kneeboard page…',
  takeBriefing: 'Use briefing',
  addAsPage: 'Add as page',
  files: (n: number) => `Files (${n})`,
  editAsText: 'Edit as text',
  binaryFile: 'Binary file',
  text: 'Text',
  applying: 'Applying…',
  bigFile: (size: string) => `Large file (${size}) – the editor may respond slowly.`,
  pickTextFile: '(choose a text file on the left)',
}

export const messages: Record<Lang, Messages> = { de, en }

const LANG_KEY = 'airborne.lang'

// The language lives in settings.json (backend); this cached copy only covers
// the moment before the first GetState answer arrives.
export function cachedLang(): Lang {
  try {
    return localStorage.getItem(LANG_KEY) === 'en' ? 'en' : 'de'
  } catch {
    return 'de'
  }
}

export function cacheLang(lang: Lang) {
  try {
    localStorage.setItem(LANG_KEY, lang)
  } catch {
    // storage unavailable – the backend setting still applies
  }
}

export const LangContext = createContext<Messages>(de)

export const useT = () => useContext(LangContext)
