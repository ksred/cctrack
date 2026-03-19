<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Overview</h1>
      <div class="page-date">
        {{ currentDate }}<span v-if="lastUpdated"> — {{ lastUpdated }}</span>
      </div>
    </div>

    <div class="stat-grid" v-if="store.summary">
      <StatCard
        label="Today"
        :value="store.summary.today.cost"
        :tokens="store.summary.today.tokens"
        :highlight="true"
        :budget="store.summary.budget"
        :trendPct="dayTrend"
        trendLabel="yesterday"
      />
      <StatCard
        label="This Week"
        :value="store.summary.week.cost"
        :tokens="store.summary.week.tokens"
        :trendPct="weekTrend"
        trendLabel="prev week"
      />
      <StatCard
        label="This Month"
        :value="store.summary.month.cost"
        :tokens="store.summary.month.tokens"
        :trendPct="monthTrend"
        trendLabel="prev month"
      />
      <StatCard
        label="Projected"
        :value="store.summary.projected"
        subtext="est. this month"
      />
    </div>

    <div class="charts-row" v-if="store.summary">
      <DailySpendChart :data="store.daily" />
      <TokenDonut
        v-if="store.summary.cost_breakdown"
        :inputCost="store.summary.cost_breakdown.input_cost"
        :outputCost="store.summary.cost_breakdown.output_cost"
        :cacheReadCost="store.summary.cost_breakdown.cache_read_cost"
        :cacheWriteCost="store.summary.cost_breakdown.cache_write_cost"
      />
    </div>

    <!-- Model Breakdown + Heatmap row -->
    <div class="insights-row" v-if="models.length || heatmap.length">
      <ModelBreakdown :models="models" />
      <ActivityHeatmap :cells="heatmap" />
    </div>

    <div class="section-header" v-if="store.recentSessions.length">
      <div class="section-title">Recent Sessions</div>
      <router-link class="view-all" to="/sessions">View all →</router-link>
    </div>

    <div class="sessions-table-wrap" v-if="store.recentSessions.length">
      <table>
        <thead>
          <tr>
            <th style="width:40px">#</th>
            <th>Session</th>
            <th>Model</th>
            <th>Last Active</th>
            <th class="right">Tokens</th>
            <th class="right">Cost</th>
          </tr>
        </thead>
        <tbody>
          <SessionRow
            v-for="(session, i) in store.recentSessions"
            :key="session.id"
            :session="session"
            :rank="i + 1"
            @select="openSession"
          />
        </tbody>
      </table>
    </div>

    <div class="section-header top-section" v-if="store.topSessions.length">
      <div class="section-title">Most Expensive</div>
    </div>

    <div class="sessions-table-wrap" v-if="store.topSessions.length">
      <table>
        <thead>
          <tr>
            <th style="width:40px">#</th>
            <th>Session</th>
            <th>Model</th>
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
import { ref, onMounted, computed } from 'vue'
import { useDashboardStore } from '../stores/dashboard'
import StatCard from '../components/primitives/StatCard.vue'
import DailySpendChart from '../components/charts/DailySpendChart.vue'
import TokenDonut from '../components/charts/TokenDonut.vue'
import ModelBreakdown from '../components/charts/ModelBreakdown.vue'
import ActivityHeatmap from '../components/charts/ActivityHeatmap.vue'
import SessionRow from '../components/domain/SessionRow.vue'
import SessionDetail from '../components/domain/SessionDetail.vue'
import SlideOver from '../components/primitives/SlideOver.vue'
import type { Session, ModelSummary, HeatmapCell } from '../types'
import { fetchSession, fetchModels, fetchHeatmap } from '../api'

const store = useDashboardStore()
const selectedSession = ref<Session | null>(null)
const models = ref<ModelSummary[]>([])
const heatmap = ref<HeatmapCell[]>([])

const currentDate = computed(() => {
  const d = new Date()
  return d.toLocaleDateString('en-GB', {
    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric'
  })
})

const lastUpdated = computed(() => {
  const ts = store.lastUpdated
  if (!ts) return null
  const time = ts.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
  // Extract timezone abbreviation from Date.toString(), e.g. "(AEST)" or "(Australian Eastern Daylight Time)"
  const tzMatch = ts.toString().match(/\((.+)\)/)
  let tz = ''
  if (tzMatch) {
    const tzName = tzMatch[1]
    tz = tzName.includes(' ')
      ? tzName.split(' ').map(w => w[0]).join('')
      : tzName
  }
  return `Last updated: ${time}${tz ? ' ' + tz : ''}`
})

function trendPct(current: number, previous: number): number | null {
  if (previous <= 0) return null
  return Math.round(((current - previous) / previous) * 100)
}

const dayTrend = computed(() => {
  if (!store.summary?.trends) return null
  return trendPct(store.summary.today.cost, store.summary.trends.prev_day_cost)
})

const weekTrend = computed(() => {
  if (!store.summary?.trends) return null
  return trendPct(store.summary.week.cost, store.summary.trends.prev_week_cost)
})

const monthTrend = computed(() => {
  if (!store.summary?.trends) return null
  return trendPct(store.summary.month.cost, store.summary.trends.prev_month_cost)
})

async function openSession(id: string) {
  selectedSession.value = await fetchSession(id)
}

onMounted(async () => {
  if (!store.loaded) store.load()
  const [m, h] = await Promise.all([fetchModels(), fetchHeatmap()])
  models.value = m || []
  heatmap.value = h || []
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
}

.insights-row {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: var(--space-5);
  margin-bottom: var(--space-8);
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
</style>
