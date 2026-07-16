<template>
  <tr
    class="group-row"
    :class="{ 'nested-group': depth > 0 }"
    @click="$emit('toggle', group.project)"
  >
    <td class="chevron-cell">
      <span class="chevron" :class="{ open: expanded }">▸</span>
    </td>
    <td>
      <div class="group-name" :style="namePad">
        {{ label }}
        <span class="session-pill">{{ group.session_count }} {{ group.session_count === 1 ? 'session' : 'sessions' }}</span>
      </div>
    </td>
    <td class="time-cell">{{ formatDate(group.started_at) }}</td>
    <td class="time-cell">{{ formatDate(group.last_activity) }}</td>
    <td class="token-cell">{{ formatTokens(group.total_tokens) }}</td>
    <td class="cost-cell">{{ formatCostDisplay(group.total_cost) }}</td>
  </tr>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ProjectGroup } from '../../types'
import { formatCostDisplay, formatTokens, formatDate } from '../../composables/useFormatCost'

const props = defineProps<{
  group: ProjectGroup
  expanded: boolean
  /** Override label (e.g. worktree suffix when nested). */
  displayName?: string
  /** Nesting depth under a parent project (0 = root). */
  depth?: number
}>()

defineEmits<{ toggle: [project: string] }>()

const depth = computed(() => props.depth ?? 0)
const label = computed(() => props.displayName ?? (props.group.project || '(no project)'))
const namePad = computed(() =>
  depth.value > 0 ? { paddingLeft: `${depth.value * 18}px` } : undefined,
)
</script>

<style scoped>
tr.group-row {
  border-bottom: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: background 100ms;
  user-select: none;
}
tr.group-row:hover { background: var(--bg-elevated); }
tr.group-row.nested-group {
  background: rgba(255, 255, 255, 0.012);
}
tr.group-row.nested-group .group-name {
  color: var(--text-secondary);
  font-weight: 400;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
}

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

.group-name {
  color: var(--text-primary);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.session-pill {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10.5px;
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
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  text-align: right;
}
</style>
