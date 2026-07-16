<template>
  <tr
    :class="{ 'active-session': isActive, 'subordinate-row': subordinate }"
    :style="subordinatePad"
    @click="$emit('select', session.id)"
  >
    <td class="rank">{{ rank }}</td>
    <td>
      <div class="session-name">
        <span class="name-text">{{ displayName }}</span>
        <span class="model-pill">{{ formatModel(session.model) }}</span>
        <span v-if="isActive" class="live-badge">Live</span>
      </div>
    </td>
    <td v-if="showStarted" class="time-cell">{{ formatDate(session.started_at) }}</td>
    <td class="time-cell">{{ formatDate(session.last_activity) }}</td>
    <td class="token-cell">{{ formatTokens(totalTokens) }}</td>
    <td class="cost-cell" :class="{ top: rank === 1 }">{{ formatCostDisplay(session.total_cost) }}</td>
  </tr>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Session } from '../../types'
import { formatCostDisplay, formatTokens, formatModel, formatDate } from '../../composables/useFormatCost'

const props = defineProps<{
  session: Session
  rank: number
  isActive?: boolean
  showStarted?: boolean
  subordinate?: boolean
  /** Parent project nesting depth — extra indent for sessions under worktrees. */
  depth?: number
}>()

defineEmits<{ select: [id: string] }>()

// When this row is rendered as a child inside a project group, the project
// name is redundant — surface the slug/id instead so adjacent rows are
// distinguishable.
const displayName = computed(() => {
  const s = props.session
  if (props.subordinate) {
    return s.slug || s.id.slice(0, 8)
  }
  return s.project || s.slug || s.id.slice(0, 8)
})

const subordinatePad = computed(() => {
  if (!props.subordinate) return undefined
  const extra = (props.depth ?? 0) * 18
  // space-10 (40px) is the base subordinate indent from CSS; add nesting on top.
  return { '--session-indent': `${40 + extra}px` } as Record<string, string>
})

const totalTokens = computed(() =>
  props.session.total_input + props.session.total_output +
  props.session.total_cache_read + props.session.total_cache_write
)
</script>

<style scoped>
tr {
  border-bottom: 1px solid var(--border-subtle);
  transition: background 100ms;
  cursor: pointer;
  position: relative;
}
tr:last-child { border-bottom: none; }
tr:hover { background: var(--bg-elevated); }
tr.active-session {
  animation: pulse-row 2.4s ease-in-out infinite;
}
tr.active-session td:first-child {
  border-left: 2px solid var(--amber-500);
}
tr.subordinate-row {
  background: rgba(255, 255, 255, 0.012);
}
tr.subordinate-row td:nth-child(2) {
  padding-left: var(--session-indent, var(--space-10));
}
tr.subordinate-row .session-name {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-secondary);
}

td {
  padding: var(--space-4) var(--space-5);
  color: var(--text-secondary);
  vertical-align: middle;
  font-size: 13px;
}

.rank {
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-disabled);
  width: 32px;
  text-align: right;
  padding-right: var(--space-2);
}
tr:first-child .rank { color: var(--amber-500); }

.session-name {
  color: var(--text-primary);
  font-weight: 400;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.model-pill {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10.5px;
  color: var(--text-tertiary);
  background: var(--bg-subtle);
  padding: 2px 6px;
  border: 1px solid var(--border-subtle);
}
.live-badge {
  font-family: 'JetBrains Mono', monospace;
  font-size: 9px;
  font-weight: 500;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--bg-base);
  background: var(--status-live);
  padding: 2px 5px;
  line-height: 1.4;
}

.time-cell {
  font-family: 'JetBrains Mono', monospace;
  font-size: 11.5px;
  color: var(--text-tertiary);
}
.token-cell {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-tertiary);
  text-align: right;
}
.cost-cell {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  text-align: right;
}
.cost-cell.top { color: var(--amber-400); }
</style>
