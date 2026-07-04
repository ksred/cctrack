<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Overview</h1>
      <div class="page-date">{{ currentDate }}</div>
    </div>

    <WindowBars
      v-if="store.summary"
      :fiveHour="store.summary.window_5h"
      :sevenDay="store.summary.window_7d"
    />

    <div class="stat-grid" v-if="store.summary">
      <StatCard
        label="5h Window"
        :value="store.summary.window_5h?.cost ?? 0"
        :tokens="store.summary.window_5h?.tokens ?? 0"
        :highlight="true"
        :trendPct="window5hTrend"
        trendLabel="prev 5h"
        :prevAmount="store.summary.window_5h?.prev_cost"
      />
      <StatCard
        :label="todayLabel"
        :value="store.summary.today?.cost ?? 0"
        :tokens="store.summary.today?.tokens ?? 0"
        :trendPct="dayTrend"
        :prevName="yesterdayLabel"
        :prevAmount="store.summary.trends?.prev_day_cost"
      />
      <StatCard
        label="7d Window"
        :value="store.summary.window_7d?.cost ?? 0"
        :tokens="store.summary.window_7d?.tokens ?? 0"
        :trendPct="window7dTrend"
        trendLabel="prev 7d"
        :prevAmount="store.summary.window_7d?.prev_cost"
      />
      <StatCard
        :label="monthLabel"
        :value="store.summary.month?.cost ?? 0"
        :tokens="store.summary.month?.tokens ?? 0"
        :budget="store.summary.budget"
        :trendPct="monthTrend"
        :prevName="prevMonthName"
        :prevAmount="store.summary.trends?.prev_month_cost"
      />
    </div>

    <div class="charts-row" :class="{ 'charts-row-single': !store.summary }">
      <DailySpendChart />
      <StatCard
        v-if="store.summary"
        class="projected-slot"
        label="Projected"
        :value="store.summary.projected"
        subtext="est. this month"
      />
    </div>

    <!-- Model breakdown + two donuts (cost-by-type, projects-by-spend) -->
    <div class="insights-row" v-if="models.length || costBreakdownSlices.length">
      <ModelBreakdown :models="models">
        <template #header-action>
          <TimeRangeSelect v-model="modelsRange" />
        </template>
      </ModelBreakdown>
      <Donut
        title="Cost Breakdown"
        :slices="costBreakdownSlices"
      >
        <template #header-action>
          <TimeRangeSelect v-model="costRange" />
        </template>
      </Donut>
      <Donut
        title="Spend by Project"
        :slices="projectSpendSlices"
        emptyText="No spend in this range"
      >
        <template #header-action>
          <TimeRangeSelect v-model="projectsRange" />
        </template>
      </Donut>
    </div>

    <!-- Activity heatmap on its own row so the wider canvas reads clearly -->
    <div class="heatmap-row" v-if="heatmap.length">
      <ActivityHeatmap :cells="heatmap" />
    </div>

    <div class="section-header" v-if="recentGroups.length">
      <div class="section-title">Recent Sessions</div>
      <router-link class="view-all" to="/sessions">View all →</router-link>
    </div>

    <div class="sessions-table-wrap" v-if="recentGroups.length">
      <table>
        <thead>
          <tr>
            <th style="width:40px"></th>
            <th>Project</th>
            <th>Started</th>
            <th>Last Active</th>
            <th class="right">Tokens</th>
            <th class="right">Cost</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="group in recentGroups" :key="group.project">
            <ProjectGroupRow
              :group="group"
              :expanded="sessionsStore.expanded.has(group.project)"
              @toggle="sessionsStore.toggleExpand"
            />
            <template v-if="sessionsStore.expanded.has(group.project)">
              <tr v-if="sessionsStore.childLoading.has(group.project)" class="loading-row">
                <td></td>
                <td colspan="5">Loading sessions…</td>
              </tr>
              <SessionRow
                v-for="(session, i) in (sessionsStore.childSessions.get(group.project) || [])"
                :key="session.id"
                :session="session"
                :rank="i + 1"
                show-started
                subordinate
                @select="openSession"
              />
            </template>
          </template>
        </tbody>
      </table>
    </div>

    <div class="section-header top-section" v-if="store.topSessions.length">
      <div class="section-title">Most Expensive Sessions</div>
    </div>

    <div class="sessions-table-wrap" v-if="store.topSessions.length">
      <table>
        <thead>
          <tr>
            <th style="width:40px">#</th>
            <th>Session</th>
            <th>Last Active</th>
            <th class="right">Tokens</th>
            <th class="right">Cost</th>
          </tr>
        </thead>
        <tbody>
          <SessionRow
            v-for="(session, i) in store.topSessions"
            :key="session.id"
            :session="session"
            :rank="i + 1"
            @select="openSession"
          />
        </tbody>
      </table>
    </div>

    <SlideOver :open="!!selectedSession" @close="selectedSession = null">
      <SessionDetail :session="selectedSession" />
    </SlideOver>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useDashboardStore } from '../stores/dashboard'
import { useSessionsStore } from '../stores/sessions'
import StatCard from '../components/primitives/StatCard.vue'
import DailySpendChart from '../components/charts/DailySpendChart.vue'
import Donut from '../components/charts/Donut.vue'
import ModelBreakdown from '../components/charts/ModelBreakdown.vue'
import ActivityHeatmap from '../components/charts/ActivityHeatmap.vue'
import WindowBars from '../components/charts/WindowBars.vue'
import SessionRow from '../components/domain/SessionRow.vue'
import ProjectGroupRow from '../components/domain/ProjectGroupRow.vue'
import SessionDetail from '../components/domain/SessionDetail.vue'
import SlideOver from '../components/primitives/SlideOver.vue'
import TimeRangeSelect, { type TimeRange } from '../components/primitives/TimeRangeSelect.vue'
import type { Session, ModelSummary, HeatmapCell, ProjectMonthly, CostBreakdown } from '../types'
import { fetchSession, fetchModels, fetchHeatmap, fetchProjectsSpend, fetchCostBreakdown } from '../api'

const store = useDashboardStore()
const sessionsStore = useSessionsStore()
const selectedSession = ref<Session | null>(null)

// Top N most-recently-active projects for the Overview preview. The full
// list lives at /sessions; here we just show enough to scan at a glance.
const recentGroups = computed(() => sessionsStore.groups.slice(0, 8))
const models = ref<ModelSummary[]>([])
const heatmap = ref<HeatmapCell[]>([])
const projectSpend = ref<ProjectMonthly[]>([])
const costBreakdown = ref<CostBreakdown | null>(null)

// Per-card time range — independent so the user can ask different
// questions of each chart without one resetting the others.
const modelsRange = ref<TimeRange>('30d')
const costRange = ref<TimeRange>('30d')
const projectsRange = ref<TimeRange>('30d')

const currentDate = computed(() => {
  const d = new Date()
  return d.toLocaleDateString('en-GB', {
    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric'
  })
})

function trendPct(current: number, previous: number): number | null {
  if (previous <= 0) return null
  return Math.round(((current - previous) / previous) * 100)
}

const window5hTrend = computed(() => {
  const w = store.summary?.window_5h
  if (!w) return null
  return trendPct(w.cost, w.prev_cost)
})

const dayTrend = computed(() => {
  if (!store.summary?.trends) return null
  return trendPct(store.summary.today?.cost ?? 0, store.summary.trends.prev_day_cost)
})

const window7dTrend = computed(() => {
  const w = store.summary?.window_7d
  if (!w) return null
  return trendPct(w.cost, w.prev_cost)
})

const monthTrend = computed(() => {
  if (!store.summary?.trends) return null
  return trendPct(store.summary.month?.cost ?? 0, store.summary.trends.prev_month_cost)
})

// Previous calendar month's full name (e.g. "April"). Setting day=1 first
// avoids the JavaScript month-arithmetic trap where Mar 31 - 1 month = Mar 3.
const prevMonthName = computed(() => {
  const d = new Date()
  d.setDate(1)
  d.setMonth(d.getMonth() - 1)
  return d.toLocaleDateString('en-GB', { month: 'long' })
})

const prevMonthLabel = computed(() => {
  const d = new Date()
  d.setDate(1)
  d.setMonth(d.getMonth() - 1)
  return d.toLocaleDateString('en-GB', { month: 'long', year: 'numeric' })
})

const todayLabel = computed(() =>
  new Date().toLocaleDateString('en-GB', { day: 'numeric', month: 'short' }),
)

const monthLabel = computed(() =>
  new Date().toLocaleDateString('en-GB', { month: 'long' }),
)

// Slices for the two Overview donuts.
const costBreakdownSlices = computed(() => {
  const cb = costBreakdown.value
  if (!cb) return []
  return [
    { label: 'Input', value: cb.input_cost, color: '#f59e0b' },
    { label: 'Output', value: cb.output_cost, color: '#fbbf24' },
    { label: 'Cache Read', value: cb.cache_read_cost, color: '#78716c' },
    { label: 'Cache Write', value: cb.cache_write_cost, color: '#44403c' },
  ]
})

// Project palette for the second donut. Same set as ModelBreakdown so the
// dashboard reads as one color story; cycles if there are more than 8 projects.
const projectColors = ['#f59e0b', '#0ea5e9', '#10b981', '#a78bfa', '#fbbf24', '#ec4899', '#78716c', '#44403c']

// Send the full project name through; the legend ellipsizes via CSS and
// surfaces the full string in a native title-attribute tooltip, while
// chart.js shows it in full when hovering a donut slice. Truncating at
// the data layer would lose the hover affordance.
const projectSpendSlices = computed(() =>
  projectSpend.value.slice(0, 8).map((p, i) => ({
    label: p.project || '(no project)',
    value: p.cost,
    color: projectColors[i % projectColors.length],
  })),
)

const yesterdayLabel = computed(() => {
  const d = new Date()
  d.setDate(d.getDate() - 1)
  return d.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })
})

async function openSession(id: string) {
  selectedSession.value = await fetchSession(id)
}

async function loadModels() {
  models.value = (await fetchModels(modelsRange.value)) || []
}
async function loadCostBreakdown() {
  costBreakdown.value = await fetchCostBreakdown(costRange.value)
}
async function loadProjectSpend() {
  projectSpend.value = (await fetchProjectsSpend(projectsRange.value)) || []
}

// Refetch each card independently when its own range changes — the others
// don't need to re-render, so isolating the watchers avoids redundant
// network requests.
watch(modelsRange, loadModels)
watch(costRange, loadCostBreakdown)
watch(projectsRange, loadProjectSpend)

async function loadHeatmap() {
  heatmap.value = (await fetchHeatmap()) || []
}

onMounted(() => {
  // Always ensure summary is present — websocket may arrive late or load()
  // may have skipped summary when a sibling endpoint failed.
  if (!store.summary) store.refreshSummary().catch(() => {})
  if (!store.loaded) store.load()
  // Reset any date filter the user may have set on the Sessions tab — the
  // Overview preview is a global "what have I been working on" view, not a
  // filtered slice. setDateFilter triggers loadGroups internally.
  sessionsStore.setDateFilter('')
  Promise.all([loadModels(), loadCostBreakdown(), loadProjectSpend(), loadHeatmap()])
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
.page-date {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-tertiary);
  padding-bottom: 4px;
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  margin-bottom: var(--space-8);
}

.charts-row {
  display: grid;
  grid-template-columns: 1fr 300px;
  gap: var(--space-5);
  margin-bottom: var(--space-8);
  align-items: stretch;
}
.charts-row-single {
  grid-template-columns: 1fr;
}
/* The Projected card sits in the right slot of charts-row where the donut
   used to live. Stretching its background to match DailySpendChart's height
   keeps the visual rhythm of the row. */
.charts-row .projected-slot {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.insights-row {
  display: grid;
  /* minmax(0, 1fr) instead of 1fr: the default 1fr is minmax(auto, 1fr),
     where 'auto' refuses to shrink below the content's min-content width.
     Long project names in the legend would otherwise widen the third
     track and push the card past the viewport. minmax(0,1fr) allows the
     track to shrink so the inner text-overflow:ellipsis can kick in. */
  grid-template-columns: 340px minmax(0, 1fr) minmax(0, 1fr);
  gap: var(--space-5);
  margin-bottom: var(--space-8);
  align-items: stretch;
}

.heatmap-row {
  margin-bottom: var(--space-8);
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 360ms;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 360ms;
}
.section-title {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}
.view-all {
  font-size: 12px;
  color: var(--amber-500);
  text-decoration: none;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: color 150ms;
}
.view-all:hover { color: var(--amber-300); }

.sessions-table-wrap {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  overflow: hidden;
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 400ms;
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

.top-section {
  margin-top: var(--space-8);
  animation-delay: 440ms;
}

tr.loading-row td {
  padding: var(--space-4) var(--space-5);
  color: var(--text-tertiary);
  font-size: 12px;
}
</style>
