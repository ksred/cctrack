import type { Summary, SessionsResponse, Session, DailySpend, Settings, RatesResponse, ProjectSummary, ProjectMonthly, ModelSummary, HeatmapCell, RequestRecord } from './types'

const BASE = '/api/v1'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}

export async function fetchSummary(): Promise<Summary> {
  return get<Summary>('/summary')
}

export async function fetchSessions(
  limit = 25,
  offset = 0,
  sort = 'cost',
  dir = 'desc'
): Promise<SessionsResponse> {
  return get<SessionsResponse>(`/sessions?limit=${limit}&offset=${offset}&sort=${sort}&dir=${dir}`)
}

export async function fetchSession(id: string): Promise<Session> {
  return get<Session>(`/sessions/${id}`)
}

export async function fetchRecent(n = 10): Promise<Session[]> {
  return get<Session[]>(`/recent?n=${n}`)
}

export async function fetchDaily(days = 30): Promise<DailySpend[]> {
  return get<DailySpend[]>(`/daily?days=${days}`)
}

export async function fetchSettings(): Promise<Settings> {
  return get<Settings>('/settings')
}

export async function updateSettings(data: Partial<Settings>): Promise<Settings> {
  const res = await fetch(`${BASE}/settings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}

export async function fetchProjects(): Promise<ProjectSummary[]> {
  return get<ProjectSummary[]>('/projects')
}

export async function fetchProjectMonthly(): Promise<ProjectMonthly[]> {
  return get<ProjectMonthly[]>('/projects/monthly')
}

export async function fetchRates(): Promise<RatesResponse> {
  return get<RatesResponse>('/rates')
}

export async function fetchModels(): Promise<ModelSummary[]> {
  return get<ModelSummary[]>('/models')
}

export async function fetchHeatmap(): Promise<HeatmapCell[]> {
  return get<HeatmapCell[]>('/heatmap')
}

export async function fetchSessionRequests(sessionId: string): Promise<RequestRecord[]> {
  return get<RequestRecord[]>(`/sessions/${sessionId}/requests`)
}
