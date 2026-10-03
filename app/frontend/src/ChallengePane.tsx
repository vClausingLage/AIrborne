import { useState } from 'react'
import { pipeline } from '../wailsjs/go/models'
import { useT } from './i18n'

interface Props {
  game: string
  aircraft: pipeline.AircraftOption[]
  maps: string[]
  busy: string
  onStart: (aircraft: string, mapHint: string) => void
}

const OTHER = '__other__'

// Herausforderung: the player picks only the aircraft (and optionally a DCS
// map); the LLM designs a challenging scenario whose enemy details never
// reach the briefing.
export default function ChallengePane({ game, aircraft, maps, busy, onStart }: Props) {
  const t = useT()
  const [choice, setChoice] = useState('')
  const [custom, setCustom] = useState('')
  const [map, setMap] = useState('')
  const isDcs = game === 'dcs'
  const type = choice === OTHER ? custom.trim() : choice
  const red = aircraft.filter((a) => a.side === 'red')
  const blue = aircraft.filter((a) => a.side === 'blue')
  const any = aircraft.filter((a) => a.side !== 'red' && a.side !== 'blue')

  const group = (label: string, list: pipeline.AircraftOption[]) =>
    list.length > 0 && (
      <optgroup label={label}>
        {list.map((a) => (
          <option key={a.type} value={a.type}>
            {a.label}
          </option>
        ))}
      </optgroup>
    )

  return (
    <section className="pane">
      <div className="pane-col narrow">
        <h3>{t.challengeHeading}</h3>
        <div className="form-grid">
          <label>{t.aircraft}</label>
          <select value={choice} onChange={(e) => setChoice(e.target.value)}>
            <option value="">{t.choose}</option>
            {group(isDcs ? t.groupBlueDcs : t.groupBlueIl2, blue)}
            {group(isDcs ? t.groupRedDcs : t.groupRedIl2, red)}
            {group(t.groupAny, any)}
            {isDcs && <option value={OTHER}>{t.otherModule}</option>}
          </select>
          {choice === OTHER && (
            <>
              <label>{t.dcsType}</label>
              <input value={custom} onChange={(e) => setCustom(e.target.value)} placeholder={t.dcsTypePlaceholder} />
            </>
          )}
          {isDcs && (
            <>
              <label>{t.map}</label>
              <select value={map} onChange={(e) => setMap(e.target.value)}>
                <option value="">{t.aiChooses}</option>
                {maps.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            </>
          )}
        </div>
        <button className="primary" onClick={() => onStart(type, map)} disabled={busy !== '' || type === ''}>
          {busy === 'challenge' ? t.challengeBusy : t.challengeButton}
        </button>
      </div>
      <div className="pane-col">
        <h3>{t.howItWorks}</h3>
        <div className="hint" style={{ fontSize: 13, lineHeight: 1.7, wordBreak: 'normal' }}>
          {t.challengeExplain}
          {isDcs && (
            <>
              <br />
              <br />
              {t.challengeExplainDcs}
            </>
          )}
          <br />
          <br />
          {t.challengeExplainAfter}
        </div>
      </div>
    </section>
  )
}
