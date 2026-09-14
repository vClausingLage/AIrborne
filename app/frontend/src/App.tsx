import { useCallback, useEffect, useState } from 'react'
import { pipeline, plan as planNS, gen, prefab } from '../wailsjs/go/models'
import {
  GetState,
  SetRoleInput,
  MergePlan,
  SetGame,
  QuickMission,
  SetPlanJSON,
  GenerateMission,
  OpenOutputFolder,
  OpenIL2Editor,
  SaveProject,
  LoadProject,
  PickFile,
  OpenFile,
  GeneratePrefab,
  SavePrefab,
  DeletePrefab,
  PlacePrefab,
  OpenPrefabFolder,
} from '../wailsjs/go/main/App'

interface State {
  roles: pipeline.RoleState[]
  config: pipeline.ConfigState
  game: string
  plan?: planNS.MissionPlan | null
  planJson: string
  projectPath: string
  root: string
  issues: string[]
  output?: gen.Result | null
  prefabs: prefab.Prefab[]
}

type Tab = 'input' | 'plan'
type Mode = 'quick' | 'steps' | 'prefab'

interface Placement {
  id: string
  side: string
  lat: string
  lon: string
  heading: string
  country: string
}

function prefabSummary(p: prefab.Prefab): string {
  const counts: Record<string, number> = {}
  for (const e of p.elements || []) counts[e.kind] = (counts[e.kind] || 0) + (e.count && e.count > 0 ? e.count : 1)
  return Object.entries(counts)
    .map(([k, n]) => `${n} ${k}`)
    .join(' · ')
}

function App() {
  const [state, setState] = useState<State | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState('')
  const [activeTab, setActiveTab] = useState<Tab>('input')
  const [mode, setMode] = useState<Mode>('steps')
  const [quickInput, setQuickInput] = useState('')
  const [planDraft, setPlanDraft] = useState('')
  const [planDirty, setPlanDirty] = useState(false)
  const [prefabInput, setPrefabInput] = useState('')
  const [prefabDraft, setPrefabDraft] = useState('')
  const [placement, setPlacement] = useState<Placement>({ id: '', side: 'friendly', lat: '', lon: '', heading: '0', country: '' })

  const applyState = useCallback((s: State) => {
    setState(s)
    setPlanDraft(s.planJson || '')
    setPlanDirty(false)
  }, [])

  const refresh = useCallback(async () => {
    try {
      applyState(await GetState())
      setError('')
    } catch (e: any) {
      setError(String(e))
    }
  }, [applyState])

  useEffect(() => {
    refresh()
  }, [refresh])

  const run = async (label: string, fn: () => Promise<State | void>, after?: () => void) => {
    setBusy(label)
    setError('')
    try {
      const s = await fn()
      if (s) applyState(s)
      after?.()
    } catch (e: any) {
      setError(String(e))
    } finally {
      setBusy('')
    }
  }

  const selectMode = (m: Mode) => {
    setMode(m)
    setActiveTab('input')
  }

  const merge = () => run('merge', MergePlan, () => setActiveTab('plan'))
  const quick = () => {
    const input = quickInput.trim()
    if (!input) return
    run('quick', () => QuickMission(input), () => setActiveTab('plan'))
  }
  const generate = () => run('generate', GenerateMission)
  const applyPlan = () => run('apply', () => SetPlanJSON(planDraft))
  const openFolder = () => run('open', () => OpenOutputFolder())
  const openEditor = () => run('editor', () => OpenIL2Editor())

  const buildPrefab = () => {
    const input = prefabInput.trim()
    if (!input) return
    run('prefab', async () => {
      const p = await GeneratePrefab(input)
      setPrefabDraft(JSON.stringify(p, null, 2))
    })
  }
  const savePrefab = () =>
    run('prefab-save', async () => {
      const s = await SavePrefab(prefabDraft)
      setPrefabDraft('')
      return s
    })
  const deletePrefab = (id: string) => run('prefab-delete', () => DeletePrefab(id))
  const openPrefabFolder = () => run('open', () => OpenPrefabFolder())
  const loadPrefabIntoEditor = (p: prefab.Prefab) => setPrefabDraft(JSON.stringify(p, null, 2))
  const placePrefab = () => {
    const lat = parseFloat(placement.lat.replace(',', '.'))
    const lon = parseFloat(placement.lon.replace(',', '.'))
    const hdg = parseFloat(placement.heading.replace(',', '.')) || 0
    if (!placement.id || isNaN(lat) || isNaN(lon)) {
      setError('Prefab, Breite (lat) und Länge (lon) angeben')
      return
    }
    run('prefab-place', () => PlacePrefab(placement.id, placement.side, lat, lon, hdg, placement.country), () => setActiveTab('plan'))
  }

  const switchGame = async (game: string) => {
    try {
      await SetGame(game)
      if (state) setState({ ...state, game })
      if (game !== 'dcs' && mode === 'prefab') setMode('steps')
    } catch (e: any) {
      setError(String(e))
    }
  }

  const save = () =>
    run('save', async () => {
      const path = await PickFile('Projekt speichern')
      if (path) {
        await SaveProject(path)
        return await GetState()
      }
    })

  const load = () =>
    run('load', async () => {
      const path = await OpenFile('Projekt öffnen')
      if (path) return await LoadProject(path)
    })

  const setInput = (key: string, value: string) => {
    if (!state) return
    const roles = state.roles.map((r) => (r.key === key ? { ...r, input: value } : r))
    setState({ ...state, roles: roles as pipeline.RoleState[] })
    SetRoleInput(key, value).catch((e) => setError(String(e)))
  }

  if (!state) {
    return (
      <div className="wrap">
        <div className="loading">{error || 'Lade…'}</div>
      </div>
    )
  }

  const isDcs = state.game === 'dcs'
  const filled = state.roles.filter((r) => r.input.trim() !== '').length
  const outDir = isDcs ? state.config.dcsMissionsDir : state.config.il2MissionsDir
  const output = state.output && state.output.game === state.plan?.game ? state.output : null

  return (
    <div className="wrap">
      <header className="topbar">
        <div className="brand">✈ Airborne</div>
        <div className="gamesel">
          <button className={!isDcs ? 'gamesel-btn active' : 'gamesel-btn'} onClick={() => switchGame('il2')} disabled={busy !== ''}>
            IL-2 Korea
          </button>
          <button className={isDcs ? 'gamesel-btn active' : 'gamesel-btn'} onClick={() => switchGame('dcs')} disabled={busy !== ''}>
            DCS World
          </button>
        </div>
        <div className="cfg">
          {state.config.hasKey ? '🔑' : '⚠'} {state.config.model || 'kein Modell'} · {state.config.base}
        </div>
        <button onClick={save} disabled={busy !== ''}>Speichern</button>
        <button onClick={load} disabled={busy !== ''}>Laden</button>
      </header>

      {error && <div className="error">{error}</div>}

      <nav className="flow">
        <div className="stage">
          <span className="stage-label">1 · Eingabe</span>
          <div className={activeTab === 'input' ? 'segment' : 'segment dim'}>
            <button className={mode === 'quick' ? 'seg active' : 'seg'} onClick={() => selectMode('quick')}>
              ⚡ Blitz
            </button>
            <button className={mode === 'steps' ? 'seg active' : 'seg'} onClick={() => selectMode('steps')}>
              Schritt für Schritt{filled > 0 ? ` (${filled}/5)` : ''}
            </button>
            {isDcs && (
              <button className={mode === 'prefab' ? 'seg active' : 'seg'} onClick={() => selectMode('prefab')} title="Wiederverwendbare Asset-Gruppen für DCS bauen">
                🧩 Prefabs{state.prefabs.length > 0 ? ` (${state.prefabs.length})` : ''}
              </button>
            )}
          </div>
        </div>
        <div className="arrow">→</div>
        <div className="stage">
          <span className="stage-label">2 · Ergebnis</span>
          <button className={activeTab === 'plan' ? 'stagebtn active' : 'stagebtn'} onClick={() => setActiveTab('plan')}>
            Plan &amp; Export
            {state.plan && <span className="badge">{output ? 'exportiert' : 'Plan bereit'}</span>}
          </button>
        </div>
      </nav>

      {activeTab === 'input' && mode === 'steps' && (
        <section className="steps">
          <div className="steps-grid">
            {state.roles.map((r) => (
              <div className="step" key={r.key}>
                <h3>{r.title}</h3>
                <div className="hint">{r.hint}</div>
                <textarea
                  className="io"
                  value={r.input}
                  onChange={(e) => setInput(r.key, e.target.value)}
                  placeholder={r.placeholder}
                />
              </div>
            ))}
            <div className="step action">
              <h3>Zusammenführen</h3>
              <div className="hint" style={{ lineHeight: 1.6 }}>
                Deine fünf Abschnitte werden in einen strukturierten Prompt eingebettet
                (<code>prompts/06-merge.md</code>: „Deine Aufgabe ist es, eine Mission zu erstellen … # Story … # Player …“)
                und in einem Durchgang zum Missionsplan (JSON) verarbeitet. Leere Abschnitte entscheidet die KI passend zur Story.
              </div>
              <button className="primary" onClick={merge} disabled={busy !== '' || filled === 0}>
                {busy === 'merge' ? 'Erstellt Missionsplan…' : 'Missionsplan erstellen'}
              </button>
            </div>
          </div>
        </section>
      )}

      {activeTab === 'input' && mode === 'quick' && (
        <section className="pane">
          <div className="pane-col">
            <h3>⚡ Blitz-Skirmish – ein Feld, fertiger Plan</h3>
            <textarea
              className="io"
              value={quickInput}
              onChange={(e) => setQuickInput(e.target.value)}
              placeholder={
                'Kurz beschreiben, was du willst. Beispiel:\n\n' +
                'Kleine Skirmish auf der Syria Map: MiG-21Bis mit Raketen im Bodenangriff, ' +
                'dazu Mi-24P und Mi-8MT, am Boden ein Infanterieverband mit APC und Artillerie, ' +
                'der sich zum Angriff formiert.'
              }
            />
            <button className="primary" onClick={quick} disabled={busy !== '' || quickInput.trim() === ''}>
              {busy === 'quick' ? 'Generiert Missionsplan…' : 'Skirmish generieren'}
            </button>
            <div className="hint" style={{ fontSize: 13, lineHeight: 1.6 }}>
              1. Oben das Zielspiel wählen (IL-2 / DCS)<br />
              2. Wunsch eintippen – Stichpunkte reichen<br />
              3. Generieren → unter „Plan &amp; Export“ prüfen und Mission erzeugen
            </div>
          </div>
        </section>
      )}

      {activeTab === 'input' && mode === 'prefab' && isDcs && (
        <section className="pane">
          <div className="pane-col">
            <h3>🧩 Prefab-Builder – Asset-Gruppe beschreiben</h3>
            <textarea
              className="io short"
              value={prefabInput}
              onChange={(e) => setPrefabInput(e.target.value)}
              placeholder={
                'Beispiel:\n\n' +
                'Trägerkampfgruppe: CVN-73 mit zwei Burke-Zerstörern als Eskorte, ' +
                'auf dem Deck sechs geparkte F/A-18C, ein E-2C und Container am Heck.\n\n' +
                'Oder: FARP im Gebirge mit vier Landeplätzen, Zelten, Tanklager, ' +
                'zwei Ural-Tankwagen und einem geparkten Mi-8.'
              }
            />
            <button className="primary" onClick={buildPrefab} disabled={busy !== '' || prefabInput.trim() === ''}>
              {busy === 'prefab' ? 'Entwirft Prefab…' : 'Prefab entwerfen'}
            </button>
            <div className="row" style={{ alignItems: 'center', marginTop: 16 }}>
              <h3 style={{ margin: 0, flex: 1 }}>Entwurf (JSON, editierbar)</h3>
              <button onClick={savePrefab} disabled={busy !== '' || prefabDraft.trim() === ''}>
                {busy === 'prefab-save' ? 'Speichert…' : 'In Bibliothek speichern'}
              </button>
            </div>
            <textarea
              className="io result mono"
              value={prefabDraft}
              onChange={(e) => setPrefabDraft(e.target.value)}
              placeholder="(noch kein Entwurf – oben beschreiben und „Prefab entwerfen“ klicken, oder rechts ein Prefab zum Bearbeiten laden)"
              spellCheck={false}
            />
            <div className="hint" style={{ lineHeight: 1.5 }}>
              Positionen sind relativ in Metern (dx = Nord, dy = Ost) um den Prefab-Ursprung; beim Platzieren wird das Prefab an lat/lon gedreht.
              Geparkte Flugzeuge auf Schiffen brauchen <code>linkTo</code>.
            </div>
          </div>
          <div className="pane-col narrow">
            <div className="row" style={{ alignItems: 'center', marginTop: 0 }}>
              <h3 style={{ margin: 0, flex: 1 }}>Bibliothek ({state.prefabs.length})</h3>
              <button onClick={openPrefabFolder} disabled={busy !== ''} title="prefabs/ im Explorer öffnen">Ordner</button>
            </div>
            {state.prefabs.length === 0 && <div className="hint">Noch keine Prefabs – links eines entwerfen und speichern.</div>}
            <div className="prefab-list">
              {state.prefabs.map((p) => (
                <div className={placement.id === p.id ? 'prefab-card active' : 'prefab-card'} key={p.id} onClick={() => setPlacement({ ...placement, id: p.id })}>
                  <div className="prefab-title">{p.name}</div>
                  <div className="hint">{prefabSummary(p)}{p.country ? ` · ${p.country}` : ''}</div>
                  {p.description && <div className="prefab-desc">{p.description}</div>}
                  <div className="row" style={{ marginTop: 6 }}>
                    <button className="small" onClick={(e) => { e.stopPropagation(); loadPrefabIntoEditor(p) }} disabled={busy !== ''}>Bearbeiten</button>
                    <button className="small" onClick={(e) => { e.stopPropagation(); if (confirm(`Prefab „${p.name}“ löschen?`)) deletePrefab(p.id) }} disabled={busy !== ''}>Löschen</button>
                  </div>
                </div>
              ))}
            </div>

            <h3 style={{ marginTop: 16 }}>In Missionsplan einfügen</h3>
            {!state.plan && <div className="hint">Erst einen Plan erzeugen (Blitz / Schritt für Schritt) – die KI kann gespeicherte Prefabs dabei auch selbst platzieren.</div>}
            <div className="form-grid">
              <label>Prefab</label>
              <select value={placement.id} onChange={(e) => setPlacement({ ...placement, id: e.target.value })}>
                <option value="">– wählen –</option>
                {state.prefabs.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
              <label>Seite</label>
              <select value={placement.side} onChange={(e) => setPlacement({ ...placement, side: e.target.value })}>
                <option value="friendly">Verbündet (Spielerseite)</option>
                <option value="enemy">Gegner</option>
              </select>
              <label>Lat / Lon</label>
              <div className="row" style={{ marginTop: 0 }}>
                <input value={placement.lat} onChange={(e) => setPlacement({ ...placement, lat: e.target.value })} placeholder="35.40" />
                <input value={placement.lon} onChange={(e) => setPlacement({ ...placement, lon: e.target.value })} placeholder="35.95" />
              </div>
              <label>Heading °</label>
              <input value={placement.heading} onChange={(e) => setPlacement({ ...placement, heading: e.target.value })} placeholder="0" />
              <label>Country</label>
              <input value={placement.country} onChange={(e) => setPlacement({ ...placement, country: e.target.value })} placeholder="(aus Prefab)" />
            </div>
            <button className="primary" onClick={placePrefab} disabled={busy !== '' || !state.plan || !placement.id}>
              {busy === 'prefab-place' ? 'Fügt ein…' : 'Prefab in Plan einfügen'}
            </button>
            {state.plan && (state.plan.prefabs || []).length > 0 && (
              <ul className="issues" style={{ marginTop: 10 }}>
                {(state.plan.prefabs || []).map((pp, i) => (
                  <li key={i}>🧩 {pp.name || pp.prefab} · {pp.side} · {pp.position?.lat?.toFixed(3)}, {pp.position?.lon?.toFixed(3)} · {pp.position?.heading || 0}°</li>
                ))}
              </ul>
            )}
          </div>
        </section>
      )}

      {activeTab === 'plan' && (
        <section className="pane">
          <div className="pane-col narrow">
            <h3>Missionsplan</h3>
            {!state.plan && <div className="hint">Noch kein Plan – Blitz oder „Schritt für Schritt“ ausführen.</div>}
            {state.plan && (state.issues || []).length > 0 && (
              <ul className="issues">
                {state.issues.map((i, n) => (
                  <li key={n}>⚠ {i}</li>
                ))}
              </ul>
            )}
            {state.plan && (state.issues || []).length === 0 && <div className="ok">✓ Plan validiert</div>}
            {state.plan && (
              <div className="hint">
                Spiel: <b>{state.plan.game === 'dcs' ? 'DCS World' : 'IL-2 Korea'}</b> · Karte: {state.plan.map} · {state.plan.date} {state.plan.time}
              </div>
            )}

            <h3 style={{ marginTop: 20 }}>Export</h3>
            <div className="hint">Zielordner: {outDir || '(nicht gesetzt – .env prüfen)'}</div>
            <button
              className="primary"
              onClick={generate}
              disabled={busy !== '' || !state.plan || planDirty}
              title={planDirty ? 'Erst JSON-Änderungen übernehmen' : ''}
            >
              {busy === 'generate' ? 'Erzeugt Mission…' : isDcs ? 'Mission erzeugen → .miz exportieren' : 'Mission erzeugen → IL-2 Ordner'}
            </button>
            {output && (
              <div className="output">
                <div className="ok">✓ {output.name}</div>
                <div className="hint">{output.mainFile}</div>
                {(output.notes || []).length > 0 && (
                  <ul className="issues">
                    {output.notes.map((n, i) => (
                      <li key={i}>ℹ {n}</li>
                    ))}
                  </ul>
                )}
                <div className="row">
                  <button onClick={openFolder} disabled={busy !== ''}>Ordner öffnen</button>
                  {output.game === 'il2' && (
                    <button onClick={openEditor} disabled={busy !== ''} title="Startet den IL-2 Editor mit Arbeitsverzeichnis bin\editor">
                      IL-2 Editor starten
                    </button>
                  )}
                </div>
                <div className="hint" style={{ lineHeight: 1.5 }}>
                  {output.game === 'il2'
                    ? 'Im Editor öffnen, Höhen/Placement prüfen und speichern (erzeugt .msnbin).'
                    : 'DCS: Mission liegt in Saved Games\\DCS\\Missions – im Editor öffnen, Bewaffnung/Flugplatz prüfen, speichern.'}
                </div>
              </div>
            )}
            <div className="hint" style={{ marginTop: 16 }}>Projekt: {state.projectPath}</div>
            <div className="hint">Log: {state.config.logPath || '(nicht initialisiert)'}</div>
          </div>
          <div className="pane-col">
            <div className="row" style={{ alignItems: 'center' }}>
              <h3 style={{ margin: 0, flex: 1 }}>JSON {planDirty ? '(geändert)' : ''}</h3>
              <button onClick={applyPlan} disabled={busy !== '' || !planDirty}>
                {busy === 'apply' ? 'Prüft…' : 'Änderungen übernehmen'}
              </button>
            </div>
            <textarea
              className="io result mono"
              value={planDraft}
              onChange={(e) => {
                setPlanDraft(e.target.value)
                setPlanDirty(e.target.value !== state.planJson)
              }}
              placeholder="(noch kein Plan)"
              spellCheck={false}
            />
          </div>
        </section>
      )}
    </div>
  )
}

export default App
