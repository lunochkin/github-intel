const base = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

export type SummaryResponse = {
  event_rows: number
  unique_events: number
  unique_repos: number
  unique_actors: number
  first_event_at?: string
  last_event_at?: string
}

export type SummaryErrorBody = {
  error: string
}

export async function fetchSummary(signal?: AbortSignal): Promise<SummaryResponse> {
  const res = await fetch(`${base}/summary`, {
    signal,
    headers: { Accept: 'application/json' },
  })
  const text = await res.text()
  let body: unknown
  try {
    body = text ? JSON.parse(text) : {}
  } catch {
    throw new Error('Invalid JSON from /summary')
  }
  if (!res.ok) {
    const err = body as SummaryErrorBody
    throw new Error(err?.error ?? `HTTP ${res.status}`)
  }
  return body as SummaryResponse
}
