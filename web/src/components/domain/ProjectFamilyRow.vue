<template>
  <tr class="family-row" @click="$emit('toggle', familyId)">
    <td class="chevron-cell">
      <span class="chevron" :class="{ open: expanded }">▸</span>
    </td>
    <td>
      <div class="family-name">
        {{ displayName }}
        <span class="session-pill">{{ rollup.session_count }} {{ rollup.session_count === 1 ? 'session' : 'sessions' }}</span>
      </div>
    </td>
    <td class="time-cell">{{ formatDate(rollup.started_at) }}</td>
    <td class="time-cell">{{ formatDate(rollup.last_activity) }}</td>
    <td class="token-cell">{{ formatTokens(rollup.total_tokens) }}</td>
    <td class="cost-cell">{{ formatCostDisplay(rollup.total_cost) }}</td>
  </tr>
</template>

<script setup lang="ts">
import type { ProjectGroup } from '../../types'
import { formatCostDisplay, formatTokens, formatDate } from '../../composables/useFormatCost'

defineProps<{
  familyId: string
  displayName: string
  rollup: ProjectGroup
  expanded: boolean
}>()

defineEmits<{ toggle: [familyId: string] }>()
</script>

<style scoped>
tr.family-row {
  border-bottom: 1px solid var(--border-default);
  cursor: pointer;
  transition: background 100ms;
  user-select: none;
  background: rgba(245, 158, 11, 0.03);
}
tr.family-row:hover { background: rgba(245, 158, 11, 0.06); }

td {
  padding: var(--space-4) var(--space-5);
  color: var(--text-secondary);
  vertical-align: middle;
  font-size: 13px;
}

.chevron-cell {
  width: 32px;
  padding-right: 0;
}
.chevron {
  display: inline-block;
  font-size: 11px;
  color: var(--text-tertiary);
  transition: transform 150ms ease;
}
.chevron.open {
  transform: rotate(90deg);
  color: var(--amber-500);
}

.family-name {
  color: var(--text-primary);
  font-weight: 600;
  font-size: 12px;
  letter-spacing: 0.1em;
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.session-pill {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10.5px;
  font-weight: 400;
  letter-spacing: 0;
  color: var(--text-tertiary);
  background: var(--bg-subtle);
  padding: 2px 6px;
  border: 1px solid var(--border-subtle);
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
  font-size: 14px;
  font-weight: 600;
  color: var(--amber-400);
  text-align: right;
}
</style>
