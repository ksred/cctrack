export interface SpendBucket {
  cost: number
  tokens: number
}

export interface CostBreakdown {
  input_cost: number
  output_cost: number
  cache_read_cost: number
  cache_write_cost: number
}

export interface Trends {
  prev_day_cost: number
  prev_week_cost: number
  prev_month_cost: number
}

export interface Summary {
  today: SpendBucket
  week: SpendBucket
  month: SpendBucket
  projected: number
  tokens: {
    input: number
    output: number
    cache_read: number
    cache_write: number
  }
  cost_breakdown: CostBreakdown
  trends: Trends
  budget: number
}

export interface ModelSummary {
  model: string
  family: string
  session_count: number
  total_cost: number
  total_tokens: number
}

export interface HeatmapCell {
  day: number
  hour: number
  cost: number
}

export interface RequestRecord {
  request_id: string
  session_id: string
  timestamp: string
  model: string
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_5m_tokens: number
  cache_write_1h_tokens: number
  cache_write_tokens: number // derived: 5m + 1h
  cost: number
}

export interface Session {
  id: string
  project: string
  slug: string
  model: string
  started_at: string
  last_activity: string
  total_input: number
  total_output: number
  total_cache_read: number
  total_cache_write_5m: number
  total_cache_write_1h: number
  total_cache_write: number // derived: 5m + 1h
  total_cost: number
}

export interface DailySpend {
  date: string
  cost: number
}

export interface SessionsResponse {
  sessions: Session[]
  total: number
  limit: number
  offset: number
}

export interface Settings {
  log_dir: string
  db_path: string
  port: number
  monthly_budget_usd: number
  open_browser_on_serve: boolean
}

export interface ModelRate {
  Family: string
  Released: string
  InputPerMToken: number
  OutputPerMToken: number
  CacheReadPerMToken: number
  CacheWrite5mPerMToken: number
  CacheWrite1hPerMToken: number
}

export interface RatesResponse {
  version: string
  updated: string
  rates: ModelRate[]
}

export interface ProjectSummary {
  project: string
  session_count: number
  total_cost: number
  total_tokens: number
  total_input: number
  total_output: number
  total_cache_read: number
  total_cache_write: number
  last_activity: string
}

export interface ProjectMonthly {
  project: string
  month: string
  cost: number
}

export interface WsEvent {
  type: 'session.updated' | 'session.created' | 'summary.updated' | 'ping'
  payload: any
}

export type ConnectionStatus = 'connected' | 'reconnecting' | 'offline'
