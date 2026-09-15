import { useState } from 'react'
import { missionfile } from '../wailsjs/go/models'
import {
  PickMissionFile,
  OpenMission,
  OpenGeneratedMission,
  MissionClose,
  MissionRead,
  MissionWrite,
  MissionSetMeta,
  MissionAddKneeboards,
  MissionAddBriefingText,
  MissionSetBriefingImage,
  MissionRemove,
  MissionSave,
  PickMissionSavePath,
  OpenMissionFolder,
  PickImage,
  PickImages,
} from '../wailsjs/go/main/App'

interface Props {
  busy: string
  hasOutput: boolean
  run: (label: string, fn: () => Promise<void>) => Promise<void>
}

const BIG = 2 * 1024 * 1024

function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

// Opens an existing .miz / .Mission and edits it in place: metadata form,
// pictures/kneeboards, and the raw text of every text entry. Nothing else in
// the mission is touched (see internal/missionfile).
export default function MissionEditor({ busy, hasOutput, run }: Props) {
  const [doc, setDoc] = useState<missionfile.Doc | null>(null)
  const [meta, setMeta] = useState<missionfile.Meta | null>(null)
  const [entry, setEntry] = useState('')
  const [text, setText] = useState('')
  const [textDirty, setTextDirty] = useState(false)
  const [kbTitle, setKbTitle] = useState('')
  const [kbBody, setKbBody] = useState('')
  const [pathInput, setPathInput] = useState('')
  const disabled = busy !== ''

  const apply = (d: missionfile.Doc) => {
    setDoc(d)
    setMeta({ ...d.meta, title: { ...d.meta.title }, briefing: { ...d.meta.briefing } } as missionfile.Meta)
  }
  const openPath = (p: string) =>
    run('mission-open', async () => {
      apply(await OpenMission(p))
      setEntry('')
      setText('')
      setTextDirty(false)
    })
  const open = () =>
    run('mission-open', async () => {
      const p = await PickMissionFile()
      if (p) await openPath(p)
    })
  const openGenerated = () =>
    run('mission-open', async () => {
      apply(await OpenGeneratedMission())
      setEntry('')
      setText('')
      setTextDirty(false)
    })
  const close = () => {
    if (doc?.dirty && !confirm('Ungespeicherte Änderungen verwerfen?')) return
    MissionClose()
    setDoc(null)
    setMeta(null)
    setEntry('')
    setText('')
  }
  const selectEntry = (name: string) =>
    run('mission-read', async () => {
      if (textDirty && !confirm('Textänderungen an der aktuellen Datei verwerfen?')) return
      setEntry(name)
      setText(await MissionRead(name))
      setTextDirty(false)
    })
  const saveText = () =>
    run('mission-write', async () => {
      apply(await MissionWrite(entry, text))
      setTextDirty(false)
    })
  const saveMeta = () => meta && run('mission-meta', async () => apply(await MissionSetMeta(meta)))
  const setBriefingImage = () =>
    run('mission-media', async () => {
      const p = await PickImage('Briefing-Bild wählen (PNG/JPG)')
      if (p) apply(await MissionSetBriefingImage(p))
    })
  const addKneeboards = () =>
    run('mission-media', async () => {
      const ps = await PickImages('Kneeboard-Seiten wählen (PNG/JPG)')
      if (ps.length) apply(await MissionAddKneeboards(ps))
    })
  const addTextPage = () =>
    run('mission-media', async () => {
      apply(await MissionAddBriefingText(kbTitle, kbBody))
      setKbBody('')
    })
  const remove = (name: string) =>
    run('mission-media', async () => {
      if (!confirm(`„${name}“ aus der Mission entfernen?`)) return
      apply(await MissionRemove(name))
      if (entry === name) {
        setEntry('')
        setText('')
        setTextDirty(false)
      }
    })
  const save = () => run('mission-save', async () => apply(await MissionSave('')))
  const saveAs = () =>
    run('mission-save', async () => {
      const p = await PickMissionSavePath()
      if (p) apply(await MissionSave(p))
    })
  const openFolder = () => run('open', () => OpenMissionFolder())

  const metaDirty =
    !!doc &&
    !!meta &&
    JSON.stringify(meta) !== JSON.stringify(doc.meta)

  if (!doc || !meta) {
    return (
      <section className="pane">
        <div className="pane-col narrow">
          <h3>🛠 Mission bearbeiten</h3>
          <button className="primary" onClick={open} disabled={disabled}>
            {busy === 'mission-open' ? 'Öffnet…' : 'Mission öffnen (.miz / .Mission)'}
          </button>
          {hasOutput && (
            <button onClick={openGenerated} disabled={disabled} style={{ marginTop: 8 }}>
              Zuletzt exportierte Mission öffnen
            </button>
          )}
          <div className="row" style={{ alignItems: 'center' }}>
            <input
              className="media-input"
              style={{ margin: 0, flex: 1 }}
              value={pathInput}
              placeholder="…oder Pfad einfügen"
              onChange={(e) => setPathInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && pathInput.trim()) openPath(pathInput.trim().replace(/^"|"$/g, ''))
              }}
            />
            <button onClick={() => openPath(pathInput.trim().replace(/^"|"$/g, ''))} disabled={disabled || pathInput.trim() === ''}>
              Öffnen
            </button>
          </div>
          <div className="hint" style={{ marginTop: 16, lineHeight: 1.6, wordBreak: 'normal' }}>
            Öffnet bestehende Missionen ohne KI: <b>DCS .miz</b> (ZIP mit mission/options/dictionary…) oder <b>IL-2 .Mission</b>{' '}
            samt Sprachdateien. Titel, Briefing, Datum/Zeit lassen sich im Formular ändern, Briefing-Bild und Kneeboards anhängen und
            jede Textdatei direkt bearbeiten. Alles andere bleibt Byte für Byte erhalten; beim ersten Speichern wird eine{' '}
            <code>.bak</code>-Kopie angelegt.
          </div>
        </div>
        <div className="pane-col" />
      </section>
    )
  }

  const isDcs = doc.game === 'dcs'
  const bigEntry = doc.entries.find((e) => e.name === entry)?.size || 0

  return (
    <section className="pane">
      <div className="pane-col narrow">
        <div className="row" style={{ alignItems: 'center', marginTop: 0 }}>
          <h3 style={{ margin: 0, flex: 1 }}>
            {isDcs ? 'DCS' : 'IL-2'} · {doc.name}
            {doc.dirty ? ' (geändert)' : ''}
          </h3>
          <button className="small" onClick={close} disabled={disabled}>Schließen</button>
        </div>
        <div className="hint" title={doc.path}>{doc.path}</div>
        {doc.protected && <ul className="issues"><li>⚠ Geschützte Mission – nur Bilder/Kneeboards änderbar</li></ul>}
        {doc.notes.length > 0 && (
          <ul className="issues">
            {doc.notes.map((n, i) => (
              <li key={i}>ℹ {n}</li>
            ))}
          </ul>
        )}
        <div className="row">
          <button className="primary" style={{ marginTop: 0 }} onClick={save} disabled={disabled || !doc.dirty || textDirty || metaDirty}
            title={textDirty || metaDirty ? 'Erst Text-/Formularänderungen übernehmen' : ''}>
            {busy === 'mission-save' ? 'Speichert…' : 'Speichern'}
          </button>
          <button onClick={saveAs} disabled={disabled || textDirty || metaDirty}>Speichern unter…</button>
          <button onClick={openFolder} disabled={disabled}>Ordner</button>
        </div>

        {!doc.protected && (
          <>
            <div className="row" style={{ alignItems: 'center', marginTop: 16 }}>
              <h3 style={{ margin: 0, flex: 1 }}>Metadaten{metaDirty ? ' (geändert)' : ''}</h3>
              <button className="small" onClick={saveMeta} disabled={disabled || !metaDirty}>Übernehmen</button>
            </div>
            <div className="form-grid">
              <label>Titel{isDcs ? '' : ' DE'}</label>
              <input value={meta.title.de} onChange={(e) => setMeta({ ...meta, title: { ...meta.title, de: e.target.value } } as missionfile.Meta)} />
              {!isDcs && (
                <>
                  <label>Titel EN</label>
                  <input value={meta.title.en} onChange={(e) => setMeta({ ...meta, title: { ...meta.title, en: e.target.value } } as missionfile.Meta)} />
                  <label>Autor</label>
                  <input value={meta.author} onChange={(e) => setMeta({ ...meta, author: e.target.value } as missionfile.Meta)} />
                </>
              )}
              <label>Datum</label>
              <input value={meta.date} placeholder="YYYY-MM-DD" onChange={(e) => setMeta({ ...meta, date: e.target.value } as missionfile.Meta)} />
              <label>Zeit</label>
              <input value={meta.time} placeholder="HH:MM:SS" onChange={(e) => setMeta({ ...meta, time: e.target.value } as missionfile.Meta)} />
            </div>
            <label className="hint">Briefing{isDcs ? '' : ' DE'}</label>
            <textarea
              className="io short"
              value={meta.briefing.de}
              onChange={(e) => setMeta({ ...meta, briefing: { ...meta.briefing, de: e.target.value } } as missionfile.Meta)}
            />
            {!isDcs && (
              <>
                <label className="hint">Briefing EN</label>
                <textarea
                  className="io short"
                  value={meta.briefing.en}
                  onChange={(e) => setMeta({ ...meta, briefing: { ...meta.briefing, en: e.target.value } } as missionfile.Meta)}
                />
              </>
            )}
          </>
        )}

        <h3 style={{ marginTop: 16 }}>Medien</h3>
        <div className="media-row">
          <span className="media-label">Briefing-Bild</span>
          {doc.briefingImages.length === 0 && <span className="hint" style={{ margin: 0 }}>(keins)</span>}
          <button className="small" onClick={setBriefingImage} disabled={disabled || doc.protected}>Wählen</button>
        </div>
        {doc.briefingImages.length > 0 && (
          <ul className="media-list">
            {doc.briefingImages.map((n) => (
              <li key={n}>
                <span className="media-file" title={n}>{n}</span>
                <button className="small" onClick={() => remove(n)} disabled={disabled}>✕</button>
              </li>
            ))}
          </ul>
        )}
        {isDcs && (
          <>
            <div className="media-row" style={{ marginTop: 8 }}>
              <span className="media-label">Kneeboard</span>
              <button className="small" onClick={addKneeboards} disabled={disabled}>Bilder hinzufügen</button>
            </div>
            {doc.kneeboards.length > 0 && (
              <ul className="media-list">
                {doc.kneeboards.map((n) => (
                  <li key={n}>
                    <span className="media-file" title={n}>{n.replace('KNEEBOARD/IMAGES/', '')}</span>
                    <button className="small" onClick={() => remove(n)} disabled={disabled}>✕</button>
                  </li>
                ))}
              </ul>
            )}
            <div className="hint">Textseite erzeugen (z. B. Briefing, Funkplan):</div>
            <input className="media-input" value={kbTitle} placeholder="Überschrift" onChange={(e) => setKbTitle(e.target.value)} />
            <textarea className="io short" value={kbBody} placeholder="Text der Kneeboard-Seite…" onChange={(e) => setKbBody(e.target.value)} />
            <div className="row">
              <button className="small" onClick={() => { setKbTitle(meta.title.de); setKbBody(meta.briefing.de) }} disabled={disabled || doc.protected}>
                Briefing übernehmen
              </button>
              <button className="small" onClick={addTextPage} disabled={disabled || kbBody.trim() === ''}>Als Seite hinzufügen</button>
            </div>
          </>
        )}
      </div>

      <div className="pane-col narrow">
        <h3>Dateien ({doc.entries.length})</h3>
        <div className="entry-list">
          {doc.entries.map((e) => (
            <div
              key={e.name}
              className={(entry === e.name ? 'entry active' : 'entry') + (e.text ? '' : ' binary')}
              onClick={() => e.text && selectEntry(e.name)}
              title={e.text ? 'Als Text bearbeiten' : 'Binärdatei'}
            >
              <span className="entry-name">{e.name}</span>
              <span className="entry-size">{fmtSize(e.size)}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="pane-col">
        <div className="row" style={{ alignItems: 'center', marginTop: 0 }}>
          <h3 style={{ margin: 0, flex: 1 }}>
            {entry || 'Text'} {textDirty ? '(geändert)' : ''}
          </h3>
          <button onClick={saveText} disabled={disabled || !textDirty}>
            {busy === 'mission-write' ? 'Übernimmt…' : 'Änderungen übernehmen'}
          </button>
        </div>
        {bigEntry > BIG && <div className="hint">Große Datei ({fmtSize(bigEntry)}) – der Editor kann träge reagieren.</div>}
        <textarea
          className="io result mono"
          value={text}
          onChange={(e) => {
            setText(e.target.value)
            setTextDirty(true)
          }}
          placeholder="(links eine Textdatei wählen)"
          spellCheck={false}
          disabled={!entry}
        />
      </div>
    </section>
  )
}
