<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Sessions</h1>
      <div class="page-meta">
        {{ store.total }} {{ pluralizeProjects(store.total) }}{{ store.dateFilter ? ' on ' + dateFilterLabel : '' }}
      </div>
    </div>

    <div v-if="store.dateFilter" class="filter-bar">
      <span class="filter-label">Filtered by date:</span>
      <span class="filter-value">{{ dateFilterLabel }}</span>
      <button class="filter-clear" @click="clearFilter" aria-label="Clear filter">×</button>
    </div>

    <div v-if="store.error" class="error-banner" role="alert">
      <span class="error-label">{{ errorLabel }}</span>
      <span class="error-detail">{{ store.error }}</span>
      <button class="error-retry" @click="retry">Retry</button>
    </div>

    <div class="sessions-table-wrap">
      <table>
        <thead>
          <tr>
            <th style="width:40px"></th>
            <th class="sortable" @click="store.setSort('project')">
              Project
              <span v-if="store.sortBy === 'project'" class="sort-arrow">{{ store.sortDir === 'desc' ? '↓' : '↑' }}</span>
            </th>
            <th class="sortable" @click="store.setSort('started')">
              Started
              <span v-if="store.sortBy === 'started'" class="sort-arrow">{{ store.sortDir === 'desc' ? '↓' : '↑' }}</span>
            </th>
            <th class="sortable" @click="store.setSort('date')">
              Last Active
              <span v-if="store.sortBy === 'date'" class="sort-arrow">{{ store.sortDir === 'desc' ? '↓' : '↑' }}</span>
            </th>
            <th class="right sortable" @click="store.setSort('tokens')">
              Tokens
              <span v-if="store.sortBy === 'tokens'" class="sort-arrow">{{ store.sortDir === 'desc' ? '↓' : '↑' }}</span>
            </th>
            <th class="right sortable" @click="store.setSort('cost')">
              Cost
              <span v-if="store.sortBy === 'cost'" class="sort-arrow">{{ store.sortDir === 'desc' ? '↓' : '↑' }}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <ProjectTreeRows
            :families="nestedFamilies"
            :expanded="store.expanded"
            :child-sessions="store.childSessions"
            :child-loading="store.childLoading"
            @toggle="store.toggleExpand"
            @select="store.selectSession"
          />
          <tr v-if="!store.loading && !store.error && !store.groups.length" class="empty-row">
            <td></td>
            <td colspan="5">No sessions{{ store.dateFilter ? ' on ' + dateFilterLabel : '' }}.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <SlideOver :open="!!store.selectedSession" @close="store.clearSelection()">
      <SessionDetail :session="store.selectedSession" />
    </SlideOver>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSessionsStore } from '../stores/sessions'
import { nestProjectGroups, sortFamilies } from '../composables/useWorktreeNesting'
import ProjectTreeRows from '../components/domain/ProjectTreeRows.vue'
import SessionDetail from '../components/domain/SessionDetail.vue'
import SlideOver from '../components/primitives/SlideOver.vue'

const store = useSessionsStore()
const route = useRoute()
const router = useRouter()

const nestedFamilies = computed(() =>
  sortFamilies(nestProjectGroups(store.groups), store.sortBy, store.sortDir),
)

// URL query is the source of truth for the date filter so a bookmarked
// /sessions?date=2026-05-03 reproduces the filtered view.
function syncFromRoute() {
  const date = (route.query.date as string) || ''
  if (date !== store.dateFilter) {
    store.setDateFilter(date)
  } else if (!store.groups.length) {
    store.loadGroups()
  }
}

onMounted(syncFromRoute)
watch(() => route.query.date, syncFromRoute)

function clearFilter() {
  router.replace({ query: { ...route.query, date: undefined } })
}

function retry() {
  store.loadGroups()
}

const errorLabel = computed(() => {
  switch (store.errorKind) {
    case 'invalid_date': return 'Invalid date'
    case 'network': return 'Network unreachable'
    default: return 'Failed to load'
  }
})

function pluralizeProjects(n: number) {
  return n === 1 ? 'project' : 'projects'
}

const dateFilterLabel = computed(() => {
  if (!store.dateFilter) return ''
  // Same local-midnight parsing trick used in DailySpendChart — bare YYYY-MM-DD
  // would otherwise be interpreted as UTC midnight and shift a day west.
  const d = new Date(store.dateFilter + 'T00:00:00')
  return d.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' })
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: var(--space-8);
  animation: fadeSlideUp 0.4s ease both;
}
.page-title {
  font-family: 'Bebas Neue', sans-serif;
  font-size: 36px;
  letter-spacing: 0.04em;
  color: var(--text-primary);
  line-height: 1;
}
.page-meta {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-tertiary);
  padding-bottom: 4px;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  margin-bottom: var(--space-4);
  background: rgba(245, 158, 11, 0.06);
  border: 1px solid rgba(245, 158, 11, 0.2);
  font-size: 12px;
  animation: fadeSlideUp 0.3s ease both;
}
.filter-label {
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-size: 10.5px;
}
.filter-value {
  color: var(--amber-400);
  font-family: 'JetBrains Mono', monospace;
}
.filter-clear {
  margin-left: auto;
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  padding: 0 var(--space-2);
  transition: color 120ms;
}
.filter-clear:hover { color: var(--text-primary); }

.error-banner {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  margin-bottom: var(--space-4);
  background: rgba(239, 68, 68, 0.06);
  border: 1px solid rgba(239, 68, 68, 0.25);
  font-size: 12px;
  animation: fadeSlideUp 0.3s ease both;
}
.error-label {
  color: rgba(239, 68, 68, 0.95);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-size: 10.5px;
  font-weight: 500;
}
.error-detail {
  color: var(--text-tertiary);
  font-family: 'JetBrains Mono', monospace;
  font-size: 11.5px;
}
.error-retry {
  margin-left: auto;
  background: transparent;
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
  font-size: 11px;
  padding: 4px 10px;
  cursor: pointer;
  font-family: 'JetBrains Mono', monospace;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  transition: border-color 120ms, color 120ms;
}
.error-retry:hover {
  border-color: var(--amber-500);
  color: var(--text-primary);
}

.sessions-table-wrap {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  overflow: hidden;
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 100ms;
}
table { width: 100%; font-size: 13px; }
thead th {
  padding: var(--space-3) var(--space-5);
  text-align: left;
  font-size: 10.5px;
  font-weight: 500;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  border-bottom: 1px solid var(--border-subtle);
  white-space: nowrap;
}
thead th.right { text-align: right; }
thead th.sortable {
  cursor: pointer;
  user-select: none;
  transition: color 150ms;
}
thead th.sortable:hover { color: var(--text-secondary); }
.sort-arrow {
  color: var(--amber-500);
  margin-left: 4px;
}

tr.loading-row td,
tr.empty-row td {
  padding: var(--space-4) var(--space-5);
  color: var(--text-tertiary);
  font-size: 12px;
}
</style>
