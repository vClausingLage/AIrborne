import { plan as planNS } from '../wailsjs/go/models'
import { PickImage, PickImages } from '../wailsjs/go/main/App'
import { useT } from './i18n'

interface Props {
  game: string
  media: planNS.Media | null | undefined
  busy: string
  onChange: (m: planNS.Media) => void
}

function baseName(p: string): string {
  const i = Math.max(p.lastIndexOf('\\'), p.lastIndexOf('/'))
  return i >= 0 ? p.slice(i + 1) : p
}

// Content that never goes through the LLM: briefing picture (both games)
// and kneeboard pages (DCS). Files stay where they are; the generator
// copies them into the mission on export.
export default function MediaPanel({ game, media, busy, onChange }: Props) {
  const t = useT()
  const isDcs = game === 'dcs'
  const m: planNS.Media = {
    briefingImage: media?.briefingImage || '',
    kneeboards: media?.kneeboards || [],
    kneeboardBriefing: !!media?.kneeboardBriefing,
  } as planNS.Media
  const disabled = busy !== ''

  const pickBriefing = async () => {
    const p = await PickImage(t.pickBriefingDialog)
    if (p) onChange({ ...m, briefingImage: p } as planNS.Media)
  }
  const addKneeboards = async () => {
    const ps = await PickImages(t.pickKneeboardDialog)
    if (ps.length) onChange({ ...m, kneeboards: [...(m.kneeboards || []), ...ps] } as planNS.Media)
  }
  const removeKneeboard = (i: number) =>
    onChange({ ...m, kneeboards: (m.kneeboards || []).filter((_, n) => n !== i) } as planNS.Media)

  return (
    <div className="media">
      <h3 style={{ marginTop: 20 }}>{t.mediaHeading}</h3>
      <div className="media-row">
        <span className="media-label">{t.briefingImage}</span>
        {m.briefingImage ? (
          <span className="media-file" title={m.briefingImage}>
            {baseName(m.briefingImage)}
          </span>
        ) : (
          <span className="hint" style={{ margin: 0 }}>{t.none}</span>
        )}
        <button className="small" onClick={pickBriefing} disabled={disabled}>{t.pick}</button>
        {m.briefingImage && (
          <button className="small" onClick={() => onChange({ ...m, briefingImage: '' } as planNS.Media)} disabled={disabled}>
            ✕
          </button>
        )}
      </div>
      <div className="hint">
        {isDcs
          ? t.briefingImageHintDcs
          : t.briefingImageHintIl2}
      </div>
      {isDcs && (
        <>
          <div className="media-row" style={{ marginTop: 8 }}>
            <span className="media-label">Kneeboard</span>
            <button className="small" onClick={addKneeboards} disabled={disabled}>{t.addPages}</button>
          </div>
          <label className="check">
            <input
              type="checkbox"
              checked={m.kneeboardBriefing}
              disabled={disabled}
              onChange={(e) => onChange({ ...m, kneeboardBriefing: e.target.checked } as planNS.Media)}
            />
            {t.kneeboardBriefing}
          </label>
          {(m.kneeboards || []).length > 0 && (
            <ul className="media-list">
              {(m.kneeboards || []).map((p, i) => (
                <li key={p + i}>
                  <span className="media-file" title={p}>{baseName(p)}</span>
                  <button className="small" onClick={() => removeKneeboard(i)} disabled={disabled}>✕</button>
                </li>
              ))}
            </ul>
          )}
          <div className="hint">{t.kneeboardHint}</div>
        </>
      )}
    </div>
  )
}
