import { useCallback, useEffect, useState } from 'react'
import { fetchSummary, type SummaryResponse } from './api/summary'
import './App.css'

const nf = new Intl.NumberFormat()

function formatInstant(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

function Dashboard({ data }: { data: SummaryResponse }) {
  const stats: { label: string; value: string }[] = [
    { label: 'Event rows', value: nf.format(data.event_rows) },
    { label: 'Unique events', value: nf.format(data.unique_events) },
    { label: 'Unique repos', value: nf.format(data.unique_repos) },
    { label: 'Unique actors', value: nf.format(data.unique_actors) },
    { label: 'First event', value: formatInstant(data.first_event_at) },
    { label: 'Last event', value: formatInstant(data.last_event_at) },
  ]

  return (
    <section className="dashboard" aria-labelledby="dash-title">
      <h2 id="dash-title" className="dashboard__title">
        Summary
      </h2>
      <p className="dashboard__hint">
        Stats from <code>raw_github_events</code> via{' '}
        <code>GET /summary</code>
      </p>
      <ul className="stat-grid">
        {stats.map(({ label, value }) => (
          <li key={label} className="stat-card">
            <span className="stat-card__label">{label}</span>
            <span className="stat-card__value">{value}</span>
          </li>
        ))}
      </ul>
    </section>
  )
}

export default function App() {
  const [data, setData] = useState<SummaryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setData(await fetchSummary())
    } catch (e) {
      setData(null)
      setError(e instanceof Error ? e.message : 'Failed to load summary')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <div className="app">
      <header className="header">
        <h1 className="header__title">GitHub Intel</h1>
        <p className="header__subtitle">Dashboard</p>
      </header>

      <main className="main">
        {loading && <p className="state state--muted">Loading summary…</p>}
        {error && (
          <div className="state state--error" role="alert">
            <p>{error}</p>
            <button type="button" className="btn" onClick={() => void load()}>
              Retry
            </button>
          </div>
        )}
        {!loading && !error && data && <Dashboard data={data} />}
      </main>
    </div>
  )
}
