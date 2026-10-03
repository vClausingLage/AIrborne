import { useCallback, useEffect, useState } from 'react'
import { pipeline, plan as planNS, gen, prefab } from '../wailsjs/go/models'
import ChallengePane from './ChallengePane'
import MediaPanel from './MediaPanel'
import MissionEditor from './MissionEditor'
import SettingsPane from './SettingsPane'
import { Lang, LangContext, cacheLang, cachedLang, messages } from './i18n'
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
  NewProject,
  PickFile,
  OpenFile,
  GeneratePrefab,
  SavePrefab,
  DeletePrefab,
  PlacePrefab,
  OpenPrefabFolder,
  ChallengeMission,
  SetMedia,
  SetMaxTokens,
  SetLanguage,
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
  repair?: pipeline.RepairInfo | null
  output?: gen.Result | null
  prefabs: prefab.Prefab[]
  aircraft: Record<string, pipeline.AircraftOption[]>
  maps: string[]
}

type Tab = 'input' | 'plan' | 'edit' | 'settings'
type Mode = 'quick' | 'steps' | 'prefab' | 'challenge'

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
  const [lang, setLang] = useState<Lang>(cachedLang)
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
  const t = messages[lang]

  const applyState = useCallback((s: State) => {
    setState(s)
    setPlanDraft(s.planJson || '')
    setPlanDirty(false)
    const l: Lang = s.config.language === 'en' ? 'en' : 'de'
    setLang(l)
    cacheLang(l)
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

  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])

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
  const challenge = (aircraft: string, mapHint: string) =>
    run('challenge', () => ChallengeMission(aircraft, mapHint), () => setActiveTab('plan'))
  const setMedia = (m: planNS.Media) => run('media', () => SetMedia(m))
  const saveMaxTokens = (n: number) => run('settings', () => SetMaxTokens(n))
  // Unsaved JSON edits survive the switch; everything else re-renders from the new state.
  const switchLanguage = (l: Lang) => {
    if (l === lang) return
    const draft = planDraft
    const dirty = planDirty
    run('language', () => SetLanguage(l), () => {
      if (dirty) {
        setPlanDraft(draft)
        setPlanDirty(true)
      }
    })
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
      setError(t.placeMissing)
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
      const path = await PickFile(t.saveProjectDialog)
      if (path) {
        await SaveProject(path)
        return await GetState()
      }
    })

  const load = () =>
    run('load', async () => {
      const path = await OpenFile(t.openProjectDialog)
      if (path) return await LoadProject(path)
    })

  const reset = () => {
    if (!confirm(t.newProjectConfirm)) return
    run('new', NewProject, () => {
      setQuickInput('')
      setPrefabInput('')
      setPrefabDraft('')
      setPlacement({ id: '', side: 'friendly', lat: '', lon: '', heading: '0', country: '' })
      setActiveTab('input')
    })
  }

  const setInput = (key: string, value: string) => {
    if (!state) return
    const roles = state.roles.map((r) => (r.key === key ? { ...r, input: value } : r))
    setState({ ...state, roles: roles as pipeline.RoleState[] })
    SetRoleInput(key, value).catch((e) => setError(String(e)))
  }

  if (!state) {
    return (
      <div className="wrap">
        <div className="loading">{error || t.loading}</div>
      </div>
    )
  }

  const isDcs = state.game === 'dcs'
  const filled = state.roles.filter((r) => r.input.trim() !== '').length
  const outDir = isDcs ? state.config.dcsMissionsDir : state.config.il2MissionsDir
  const output = state.output && state.output.game === state.plan?.game ? state.output : null

  return (
    <LangContext.Provider value={t}>
      <div className="wrap">
        <header className="topbar">
          <div className="brand">
            ✈ <span className="brand-ai">AI</span>rborne
          </div>
          <div className="gamesel">
            <button className={!isDcs ? 'gamesel-btn active' : 'gamesel-btn'} onClick={() => switchGame('il2')} disabled={busy !== ''}>
              IL-2 Korea
            </button>
            <button className={isDcs ? 'gamesel-btn active' : 'gamesel-btn'} onClick={() => switchGame('dcs')} disabled={busy !== ''}>
              DCS World
            </button>
          </div>
          <div className="cfg">
            {state.config.hasKey ? '🔑' : '⚠'} {state.config.model || t.noModel} · {state.config.base} ·{' '}
            {t.tokensPerPlan(state.config.maxTokens.toLocaleString(t.locale))}
          </div>
          <div className="langsel" title={t.languageTitle}>
            {(['de', 'en'] as Lang[]).map((l) => (
              <button key={l} className={lang === l ? 'langsel-btn active' : 'langsel-btn'} onClick={() => switchLanguage(l)} disabled={busy !== ''}>
                {l.toUpperCase()}
              </button>
            ))}
          </div>
          <button
            className={activeTab === 'settings' ? 'stagebtn active' : 'stagebtn'}
            onClick={() => setActiveTab(activeTab === 'settings' ? 'input' : 'settings')}
            disabled={busy !== ''}
            title={t.settingsTitle}
          >
            {t.settings}
          </button>
          <button onClick={reset} disabled={busy !== ''} title={t.newProjectTitle}>{t.newProject}</button>
          <button onClick={save} disabled={busy !== ''}>{t.save}</button>
          <button onClick={load} disabled={busy !== ''}>{t.load}</button>
        </header>

        {error && <div className="error">{error}</div>}

        <nav className="flow">
          <div className="stage">
            <span className="stage-label">{t.stageInput}</span>
            <div className={activeTab === 'input' ? 'segment' : 'segment dim'}>
              <button className={mode === 'quick' ? 'seg active' : 'seg'} onClick={() => selectMode('quick')}>
                {t.modeQuick}
              </button>
              <button className={mode === 'steps' ? 'seg active' : 'seg'} onClick={() => selectMode('steps')}>
                {t.modeSteps}{filled > 0 ? ` (${filled}/5)` : ''}
              </button>
              <button className={mode === 'challenge' ? 'seg active' : 'seg'} onClick={() => selectMode('challenge')} title={t.modeChallengeTitle}>
                {t.modeChallenge}
              </button>
              {isDcs && (
                <button className={mode === 'prefab' ? 'seg active' : 'seg'} onClick={() => selectMode('prefab')} title={t.modePrefabsTitle}>
                  {t.modePrefabs}{state.prefabs.length > 0 ? ` (${state.prefabs.length})` : ''}
                </button>
              )}
            </div>
          </div>
          <div className="arrow">→</div>
          <div className="stage">
            <span className="stage-label">{t.stageResult}</span>
            <button className={activeTab === 'plan' ? 'stagebtn active' : 'stagebtn'} onClick={() => setActiveTab('plan')}>
              {t.planExport}
              {state.plan && <span className="badge">{output ? t.badgeExported : state.plan.challenge ? t.badgeChallenge : t.badgeReady}</span>}
            </button>
          </div>
          <div className="arrow">→</div>
          <div className="stage">
            <span className="stage-label">{t.stageEdit}</span>
            <button className={activeTab === 'edit' ? 'stagebtn active' : 'stagebtn'} onClick={() => setActiveTab('edit')} title={t.editMissionTitle}>
              {t.editMission}
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
                <h3>{t.merge}</h3>
                <div className="hint" style={{ lineHeight: 1.6 }}>
                  {t.mergeHint}
                </div>
                <button className="primary" onClick={merge} disabled={busy !== '' || filled === 0}>
                  {busy === 'merge' ? t.merging : t.mergeButton}
                </button>
              </div>
            </div>
          </section>
        )}

        {activeTab === 'input' && mode === 'quick' && (
          <section className="pane">
            <div className="pane-col">
              <h3>{t.quickHeading}</h3>
              <textarea
                className="io"
                value={quickInput}
                onChange={(e) => setQuickInput(e.target.value)}
                placeholder={t.quickPlaceholder}
              />
              <button className="primary" onClick={quick} disabled={busy !== '' || quickInput.trim() === ''}>
                {busy === 'quick' ? t.quickBusy : t.quickButton}
              </button>
              <div className="hint" style={{ fontSize: 13, lineHeight: 1.6 }}>
                {t.quickHint}
              </div>
            </div>
          </section>
        )}

        {activeTab === 'input' && mode === 'challenge' && (
          <ChallengePane game={state.game} aircraft={state.aircraft?.[state.game] || []} maps={state.maps || []} busy={busy} onStart={challenge} />
        )}

        {activeTab === 'edit' && <MissionEditor busy={busy} hasOutput={!!output} run={run} />}

        {activeTab === 'settings' && (
          <SettingsPane config={state.config} busy={busy} lang={lang} onSaveMaxTokens={saveMaxTokens} onSetLanguage={switchLanguage} />
        )}

        {activeTab === 'input' && mode === 'prefab' && isDcs && (
          <section className="pane">
            <div className="pane-col">
              <h3>{t.prefabHeading}</h3>
              <textarea
                className="io short"
                value={prefabInput}
                onChange={(e) => setPrefabInput(e.target.value)}
                placeholder={t.prefabPlaceholder}
              />
              <button className="primary" onClick={buildPrefab} disabled={busy !== '' || prefabInput.trim() === ''}>
                {busy === 'prefab' ? t.prefabBusy : t.prefabButton}
              </button>
              <div className="row" style={{ alignItems: 'center', marginTop: 16 }}>
                <h3 style={{ margin: 0, flex: 1 }}>{t.prefabDraft}</h3>
                <button onClick={savePrefab} disabled={busy !== '' || prefabDraft.trim() === ''}>
                  {busy === 'prefab-save' ? t.saving : t.saveToLibrary}
                </button>
              </div>
              <textarea
                className="io result mono"
                value={prefabDraft}
                onChange={(e) => setPrefabDraft(e.target.value)}
                placeholder={t.prefabDraftPlaceholder}
                spellCheck={false}
              />
              <div className="hint" style={{ lineHeight: 1.5 }}>
                {t.prefabDraftHint}
              </div>
            </div>
            <div className="pane-col narrow">
              <div className="row" style={{ alignItems: 'center', marginTop: 0 }}>
                <h3 style={{ margin: 0, flex: 1 }}>{t.library(state.prefabs.length)}</h3>
                <button onClick={openPrefabFolder} disabled={busy !== ''} title={t.prefabFolderTitle}>{t.folder}</button>
              </div>
              {state.prefabs.length === 0 && <div className="hint">{t.libraryEmpty}</div>}
              <div className="prefab-list">
                {state.prefabs.map((p) => (
                  <div className={placement.id === p.id ? 'prefab-card active' : 'prefab-card'} key={p.id} onClick={() => setPlacement({ ...placement, id: p.id })}>
                    <div className="prefab-title">{p.name}</div>
                    <div className="hint">{prefabSummary(p)}{p.country ? ` · ${p.country}` : ''}</div>
                    {p.description && <div className="prefab-desc">{p.description}</div>}
                    <div className="row" style={{ marginTop: 6 }}>
                      <button className="small" onClick={(e) => { e.stopPropagation(); loadPrefabIntoEditor(p) }} disabled={busy !== ''}>{t.edit}</button>
                      <button className="small" onClick={(e) => { e.stopPropagation(); if (confirm(t.deletePrefabConfirm(p.name))) deletePrefab(p.id) }} disabled={busy !== ''}>{t.delete}</button>
                    </div>
                  </div>
                ))}
              </div>

              <h3 style={{ marginTop: 16 }}>{t.insertIntoPlan}</h3>
              {!state.plan && <div className="hint">{t.insertNoPlan}</div>}
              <div className="form-grid">
                <label>Prefab</label>
                <select value={placement.id} onChange={(e) => setPlacement({ ...placement, id: e.target.value })}>
                  <option value="">{t.choose}</option>
                  {state.prefabs.map((p) => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
                <label>{t.side}</label>
                <select value={placement.side} onChange={(e) => setPlacement({ ...placement, side: e.target.value })}>
                  <option value="friendly">{t.sideFriendly}</option>
                  <option value="enemy">{t.sideEnemy}</option>
                </select>
                <label>Lat / Lon</label>
                <div className="row" style={{ marginTop: 0 }}>
                  <input value={placement.lat} onChange={(e) => setPlacement({ ...placement, lat: e.target.value })} placeholder="35.40" />
                  <input value={placement.lon} onChange={(e) => setPlacement({ ...placement, lon: e.target.value })} placeholder="35.95" />
                </div>
                <label>Heading °</label>
                <input value={placement.heading} onChange={(e) => setPlacement({ ...placement, heading: e.target.value })} placeholder="0" />
                <label>Country</label>
                <input value={placement.country} onChange={(e) => setPlacement({ ...placement, country: e.target.value })} placeholder={t.countryFromPrefab} />
              </div>
              <button className="primary" onClick={placePrefab} disabled={busy !== '' || !state.plan || !placement.id}>
                {busy === 'prefab-place' ? t.placing : t.placeButton}
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
              <h3>{t.missionPlan}</h3>
              {!state.plan && <div className="hint">{t.noPlanYet}</div>}
              {state.plan && (state.issues || []).length > 0 && (
                <ul className="issues">
                  {state.issues.map((i, n) => (
                    <li key={n}>⚠ {i}</li>
                  ))}
                </ul>
              )}
              {state.plan && (state.issues || []).length === 0 && <div className="ok">{t.planValid}</div>}
              {state.plan && state.repair && (
                <details className="repair">
                  <summary>
                    {t.repairSummary(state.repair.found.length)}
                    {state.repair.repaired
                      ? t.repairFixed(state.repair.found.length - state.repair.remaining.length, state.repair.remaining.length)
                      : t.repairKept}
                  </summary>
                  <ul className="issues">
                    {state.repair.found.map((f, i) => (
                      <li key={i}>{state.repair!.repaired && !state.repair!.remaining.includes(f) ? '✓' : '⚠'} {f}</li>
                    ))}
                  </ul>
                  {state.repair.note && <div className="hint">{state.repair.note}</div>}
                  <div className="hint">{t.repairHint}</div>
                </details>
              )}
              {state.plan && (
                <div className="hint">
                  {t.game}: <b>{state.plan.game === 'dcs' ? 'DCS World' : 'IL-2 Korea'}</b> · {t.map}: {state.plan.map} · {state.plan.date} {state.plan.time}
                  {state.plan.challenge && <> · <b>{t.modeChallenge}</b> {t.challengeHidden}</>}
                </div>
              )}
              {state.plan && <MediaPanel game={state.plan.game} media={state.plan.media} busy={busy} onChange={setMedia} />}

              <h3 style={{ marginTop: 20 }}>{t.export}</h3>
              <div className="hint">{t.targetFolder}: {outDir || t.notSetEnv}</div>
              <button
                className="primary"
                onClick={generate}
                disabled={busy !== '' || !state.plan || planDirty}
                title={planDirty ? t.applyJsonFirst : ''}
              >
                {busy === 'generate' ? t.generating : isDcs ? t.generateDcs : t.generateIl2}
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
                    <button onClick={openFolder} disabled={busy !== ''}>{t.openFolder}</button>
                    {output.game === 'il2' && (
                      <button onClick={openEditor} disabled={busy !== ''} title={t.startIl2EditorTitle}>
                        {t.startIl2Editor}
                      </button>
                    )}
                  </div>
                  <div className="hint" style={{ lineHeight: 1.5 }}>
                    {output.game === 'il2' ? t.il2AfterExport : t.dcsAfterExport}
                  </div>
                </div>
              )}
              <div className="hint" style={{ marginTop: 16 }}>{t.project}: {state.projectPath}</div>
              <div className="hint">{t.log}: {state.config.logPath || t.notInitialized}</div>
            </div>
            <div className="pane-col">
              <div className="row" style={{ alignItems: 'center' }}>
                <h3 style={{ margin: 0, flex: 1 }}>JSON {planDirty ? t.changed : ''}</h3>
                <button onClick={applyPlan} disabled={busy !== '' || !planDirty}>
                  {busy === 'apply' ? t.checking : t.applyChanges}
                </button>
              </div>
              <textarea
                className="io result mono"
                value={planDraft}
                onChange={(e) => {
                  setPlanDraft(e.target.value)
                  setPlanDirty(e.target.value !== state.planJson)
                }}
                placeholder={t.noPlanPlaceholder}
                spellCheck={false}
              />
            </div>
          </section>
        )}
      </div>
    </LangContext.Provider>
  )
}

export default App
