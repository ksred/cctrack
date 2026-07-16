export function formatCostDisplay(value: number): string {
  if (value < 0.01) return '$' + value.toFixed(4)
  return '$' + value.toFixed(2)
}

/** Whole-dollar display for larger aggregates (7d, monthly, projected, etc.). */
export function formatCostWhole(value: number): string {
  return '$' + Math.round(value).toLocaleString('en-US')
}

export function formatCostPrecise(value: number): string {
  return '$' + value.toFixed(4)
}

export function formatTokens(n: number): string {
  if (n < 1000) return String(n)
  if (n < 1_000_000) return (n / 1000).toFixed(1) + 'K'
  return (n / 1_000_000).toFixed(1) + 'M'
}

export function formatTokensRaw(n: number): string {
  return n.toLocaleString()
}

// Render a canonical model id (e.g. "claude-opus-4-7-20251020") as
// "Opus 4.7": strip the trailing date suffix and reuse the family
// formatter so version digits join with dots, not spaces.
export function formatModel(model: string): string {
  return formatFamily(model.replace(/-\d{8}$/, ''))
}

// Render a canonical Family string (e.g. "claude-opus-4-7") as a human-readable
// "Opus 4.7" — capitalize the model name and join the remaining numeric segments
// with dots so version numbers read naturally.
export function formatFamily(family: string): string {
  if (!family) return ''
  const stripped = family.replace(/^claude-/, '').replace(/^anthropic-/, '')
  const segments = stripped.split('-')
  if (!segments.length || !segments[0]) return family
  const name = segments[0].charAt(0).toUpperCase() + segments[0].slice(1)
  const versionParts = segments.slice(1).filter(s => /^\d+$/.test(s))
  return versionParts.length ? `${name} ${versionParts.join('.')}` : name
}

export function formatDate(iso: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  const now = new Date()
  const isToday = d.toDateString() === now.toDateString()
  if (isToday) {
    return d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })
}
