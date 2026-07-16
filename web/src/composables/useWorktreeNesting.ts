import type { ProjectGroup } from '../types'

/** One expandable project row (main working tree or a worktree). */
export interface NestedProjectGroup {
  group: ProjectGroup
  /** Label shown in the table (e.g. `main`, `worktrees-team2`). */
  displayName: string
  depth: number
}

/**
 * Top-level family: rolls up a main project and all of its worktrees.
 * Header shows consolidated cost; members keep their own lines and sessions.
 */
export interface ProjectFamily {
  /** Stable id for collapse state (`family:<root-project>`). */
  id: string
  /** Uppercase label without common path preambles. */
  displayName: string
  rollup: ProjectGroup
  members: NestedProjectGroup[]
}

const WORKTREE_MARKER = '-worktrees-'

/** Path prefixes stripped before uppercasing the family header. */
const FAMILY_PREAMBLES = ['webstorm-projects-']

/**
 * Find the best parent project for a worktree name among `candidates`.
 * Prefers the longest matching prefix of the path segment before `-worktrees-`.
 *
 * e.g. `…-lexigraf-web-worktrees-team2` → `…-lexigraf-web`
 *      `…-agni-books-scriptorium-worktrees-team3` → `…-agni-books`
 */
export function findWorktreeParent(name: string, candidates: string[]): string | null {
  const idx = name.indexOf(WORKTREE_MARKER)
  if (idx < 0) return null
  const before = name.slice(0, idx)
  let best: string | null = null
  for (const c of candidates) {
    if (c === name) continue
    if (c === before || before.startsWith(c + '-')) {
      if (!best || c.length > best.length) best = c
    }
  }
  return best
}

/** Strip known path preambles and uppercase for the family header. */
export function familyDisplayName(project: string): string {
  let s = project || ''
  const lower = s.toLowerCase()
  for (const p of FAMILY_PREAMBLES) {
    if (lower.startsWith(p)) {
      s = s.slice(p.length)
      break
    }
  }
  return (s || '(no project)').toUpperCase()
}

function rollupGroups(groups: ProjectGroup[], rootName: string): ProjectGroup {
  let session_count = 0
  let total_cost = 0
  let total_tokens = 0
  let started_at = ''
  let last_activity = ''
  for (const g of groups) {
    session_count += g.session_count
    total_cost += g.total_cost
    total_tokens += g.total_tokens
    if (g.started_at && (!started_at || g.started_at < started_at)) started_at = g.started_at
    if (g.last_activity && (!last_activity || g.last_activity > last_activity)) {
      last_activity = g.last_activity
    }
  }
  return {
    project: rootName,
    session_count,
    total_cost,
    total_tokens,
    started_at,
    last_activity,
  }
}

/**
 * Group a flat project list into families: each main working tree plus its
 * worktrees become siblings under one consolidated header. Order of families
 * follows the order of root projects in `groups`; orphan worktrees (no parent
 * present) become their own single-member family.
 */
export function nestProjectGroups(groups: ProjectGroup[]): ProjectFamily[] {
  if (!groups.length) return []

  const names = groups.map(g => g.project)
  const parentOf = new Map<string, string>()
  for (const name of names) {
    const parent = findWorktreeParent(name, names)
    if (parent) parentOf.set(name, parent)
  }

  const childNamesOf = new Map<string, string[]>()
  for (const [child, parent] of parentOf) {
    const list = childNamesOf.get(parent) ?? []
    list.push(child)
    childNamesOf.set(parent, list)
  }

  const roots = groups.filter(g => !parentOf.has(g.project))

  return roots.map(root => {
    const worktreeNames = childNamesOf.get(root.project) ?? []
    // Preserve original list order among worktree siblings.
    const worktrees = groups.filter(x => worktreeNames.includes(x.project))
    const memberGroups = [root, ...worktrees]

    const members: NestedProjectGroup[] = memberGroups.map((g, i) => ({
      group: g,
      displayName: i === 0
        ? 'main'
        : g.project.slice(root.project.length + 1),
      depth: 1,
    }))

    return {
      id: `family:${root.project}`,
      displayName: familyDisplayName(root.project),
      rollup: rollupGroups(memberGroups, root.project),
      members,
    }
  })
}

const familySortKeys: Record<string, keyof ProjectGroup> = {
  cost: 'total_cost',
  date: 'last_activity',
  started: 'started_at',
  tokens: 'total_tokens',
  project: 'project',
}

/** Sort families (and members within) by the Sessions table column. */
export function sortFamilies(
  families: ProjectFamily[],
  sortBy: string,
  sortDir: 'asc' | 'desc',
): ProjectFamily[] {
  const key = familySortKeys[sortBy] ?? 'last_activity'
  const factor = sortDir === 'desc' ? -1 : 1

  function cmp(a: string | number, b: string | number) {
    if (a === b) return 0
    return a < b ? -1 * factor : 1 * factor
  }

  return families
    .map(f => {
      const main = f.members.find(m => m.displayName === 'main')
      const rest = f.members.filter(m => m.displayName !== 'main')
      rest.sort((a, b) =>
        cmp(a.group[key] as string | number, b.group[key] as string | number),
      )
      return {
        ...f,
        members: main ? [main, ...rest] : rest,
      }
    })
    .sort((a, b) => {
      // Prefer rollup for cost/tokens/dates; project sort uses the display name.
      if (sortBy === 'project') return cmp(a.displayName, b.displayName)
      return cmp(a.rollup[key] as string | number, b.rollup[key] as string | number)
    })
}

/** Map raw project name → family for chart/monthly rollups. */
export function familyLookup(families: ProjectFamily[]): Map<string, ProjectFamily> {
  const map = new Map<string, ProjectFamily>()
  for (const f of families) {
    for (const m of f.members) {
      map.set(m.group.project, f)
    }
  }
  return map
}
