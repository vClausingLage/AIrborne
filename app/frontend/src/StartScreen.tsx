import { Lang, useT } from './i18n'

interface Props {
  lang: Lang
  busy: string
  error: string
  // game of the autosaved project, if it already has a plan
  currentGame: string
  onPick: (game: string) => void
  onSetLanguage: (l: Lang) => void
}

// First screen: the simulator is chosen here once; the rest of the UI has no
// game switch.
export default function StartScreen({ lang, busy, error, currentGame, onPick, onSetLanguage }: Props) {
  const t = useT()
  const games = [
    { id: 'dcs', tag: t.startDcsTag, title: 'DCS World', desc: t.startDcsDesc, features: t.startDcsFeatures },
    { id: 'il2', tag: t.startIl2Tag, title: 'IL-2 Sturmovik: Korea', desc: t.startIl2Desc, features: t.startIl2Features },
  ]

  return (
    <div className="start">
      <header className="start-hero">
        <div className="langsel" title={t.languageTitle}>
          {(['de', 'en'] as Lang[]).map((l) => (
            <button key={l} className={lang === l ? 'langsel-btn active' : 'langsel-btn'} onClick={() => onSetLanguage(l)} disabled={busy !== ''}>
              {l.toUpperCase()}
            </button>
          ))}
        </div>
        <div className="start-logo">
          ✈ <span className="brand-ai">AI</span>rborne
        </div>
        <h1 className="start-title">{t.startTitle}</h1>
        <p className="start-subtitle">{t.startSubtitle}</p>
      </header>

      <main className="start-body">
        {error && <div className="error">{error}</div>}
        <div className="game-cards">
          {games.map((g) => (
            <button key={g.id} className="game-card" onClick={() => onPick(g.id)} disabled={busy !== ''}>
              <span className="game-card-tag">{g.tag}</span>
              <span className="game-card-title">{g.title}</span>
              <span className="game-card-desc">{g.desc}</span>
              <ul className="game-card-features">
                {g.features.map((f) => (
                  <li key={f}>{f}</li>
                ))}
              </ul>
              <span className="game-card-footer">
                {currentGame === g.id && <span className="game-card-current">● {t.startCurrentProject}</span>}
                <span className="game-card-cta">{t.startCta} →</span>
              </span>
            </button>
          ))}
        </div>
      </main>
    </div>
  )
}
