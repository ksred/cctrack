<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Projects</h1>
      <div class="page-meta">{{ families.length }} projects</div>
    </div>

    <!-- Cost by project bar chart -->
    <div class="charts-row" v-if="families.length">
      <div class="chart-card">
        <div class="chart-header">
          <div class="chart-title">Cost by Project</div>
          <div class="chart-meta">{{ formatCostDisplay(totalCost) }} total</div>
        </div>
        <div class="chart-canvas-wrap bar-chart" :style="{ height: barChartHeight + 'px' }">
          <Bar v-if="projectBarData" :data="projectBarData" :options="projectBarOptions" />
        </div>
      </div>

      <div class="chart-card">
        <div class="chart-header">
          <div class="chart-title">Share of Spend</div>
        </div>
        <div class="chart-canvas-wrap tall">
          <Doughnut v-if="projectDonutData" :data="projectDonutData" :options="donutOptions" />
        </div>
        <div class="donut-legend">
          <div v-for="(f, i) in topFamiliesForLegend" :key="f.id" class="legend-row">
            <div class="legend-left">
              <div class="legend-dot" :style="{ background: projectColors[i] }"></div>
              <span>{{ f.displayName }}</span>
            </div>
            <div class="legend-val">{{ formatCostDisplay(f.rollup.total_cost) }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Monthly cost per project stacked bar -->
    <div class="chart-card full-width" v-if="monthlyData.length || monthlyLoading">
      <div class="chart-header">
        <div class="chart-title">Monthly Spend by Project</div>
        <div v-if="monthlyLoading" class="chart-meta">Loading…</div>
      </div>
      <div class="chart-canvas-wrap tall">
        <Bar v-if="monthlyChartData" :data="monthlyChartData" :options="monthlyBarOptions" />
        <div v-else-if="monthlyLoading" class="chart-placeholder">Aggregating monthly spend…</div>
      </div>
    </div>

    <!-- Project table -->
    <div class="section-header">
      <div class="section-title">All Projects</div>
    </div>

    <div class="sessions-table-wrap" v-if="families.length">
      <table>
        <thead>
          <tr>
            <th style="width:40px">#</th>
            <th>Project</th>
            <th class="right">Sessions</th>
            <th class="right">Tokens</th>
            <th>Last Active</th>
            <th class="right">Cost</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="(family, i) in families" :key="family.id">
            <tr class="family-row">
              <td class="rank" :class="{ top: i === 0 }">{{ i + 1 }}</td>
              <td>
                <div class="family-name">
                  {{ family.displayName }}
                  <span v-if="family.members.length > 1" class="member-pill">{{ family.members.length }} trees</span>
                </div>
              </td>
              <td class="mono right">{{ family.rollup.session_count }}</td>
              <td class="mono right dim">{{ formatTokens(family.rollup.total_tokens) }}</td>
              <td class="mono dim">{{ formatDate(family.rollup.last_activity) }}</td>
              <td class="cost-cell" :class="{ top: i === 0 }">{{ formatCostDisplay(family.rollup.total_cost) }}</td>
            </tr>
            <tr
              v-for="node in visibleMembers(family)"
              :key="node.group.project"
              class="member-row"
            >
              <td class="rank"></td>
              <td class="member-name">{{ node.displayName }}</td>
              <td class="mono right">{{ node.group.session_count }}</td>
              <td class="mono right dim">{{ formatTokens(node.group.total_tokens) }}</td>
              <td class="mono dim">{{ formatDate(node.group.last_activity) }}</td>
              <td class="cost-cell dim-cost">{{ formatCostDisplay(node.group.total_cost) }}</td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Bar, Doughnut } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  ArcElement,
  Tooltip,
  Legend,
} from 'chart.js'
import type { ProjectSummary, ProjectMonthly, ProjectGroup } from '../types'
import type { ProjectFamily } from '../composables/useWorktreeNesting'
import { fetchProjects, fetchProjectMonthly } from '../api'
import { formatCostDisplay, formatTokens, formatDate } from '../composables/useFormatCost'
import {
  nestProjectGroups,
  sortFamilies,
  familyLookup,
} from '../composables/useWorktreeNesting'

ChartJS.register(CategoryScale, LinearScale, BarElement, ArcElement, Tooltip, Legend)

const projects = ref<ProjectSummary[]>([])
const monthlyData = ref<ProjectMonthly[]>([])
const monthlyLoading = ref(false)

const projectColors = [
  '#f59e0b', '#fbbf24', '#fcd34d', '#d97706',
  '#92400e', '#78716c', '#57534e', '#44403c',
  '#a8a29e', '#6b7280', '#4b5563', '#374151',
]

function toGroup(p: ProjectSummary): ProjectGroup {
  return {
    project: p.project,
    session_count: p.session_count,
    total_cost: p.total_cost,
    total_tokens: p.total_tokens,
    started_at: '',
    last_activity: p.last_activity,
  }
}

const families = computed(() =>
  sortFamilies(nestProjectGroups(projects.value.map(toGroup)), 'cost', 'desc'),
)

const byProjectFamily = computed(() => familyLookup(families.value))

/** Multi-member families show main + worktrees; singles are header-only. */
function visibleMembers(family: ProjectFamily) {
  return family.members.length > 1 ? family.members : []
}

const totalCost = computed(() =>
  families.value.reduce((sum, f) => sum + f.rollup.total_cost, 0),
)

const topFamiliesForLegend = computed(() => families.value.slice(0, 8))

// Horizontal bar chart: cost by family
const BAR_PROJECT_LIMIT = 20
/** Vertical room per bar so labels stay ~11px-readable (not crushed into the canvas). */
const BAR_ROW_PX = 22

const barChartHeight = computed(() => {
  const n = Math.min(families.value.length, BAR_PROJECT_LIMIT)
  // Axis padding + per-row allotment; floor keeps short lists from looking sparse.
  return Math.max(300, n * BAR_ROW_PX + 48)
})

const projectBarData = computed(() => {
  if (!families.value.length) return null
  const top = families.value.slice(0, BAR_PROJECT_LIMIT)
  return {
    labels: top.map(f => f.displayName),
    datasets: [{
      data: top.map(f => f.rollup.total_cost),
      backgroundColor: top.map((_, i) => projectColors[i % projectColors.length]),
      borderColor: 'transparent',
      borderWidth: 0,
      borderRadius: 0,
      barPercentage: 0.7,
      categoryPercentage: 0.85,
    }],
  }
})

const projectBarOptions = {
  responsive: true,
  maintainAspectRatio: false,
  indexAxis: 'y' as const,
  animation: { duration: 700, easing: 'easeOutQuart' as const },
  layout: {
    padding: { top: 4, bottom: 4 },
  },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: '#1a1a18',
      borderColor: '#2a2a27',
      borderWidth: 1,
      titleColor: '#8c8a84',
      bodyColor: '#f59e0b',
      bodyFont: { family: 'JetBrains Mono', size: 13 },
      titleFont: { family: 'DM Sans', size: 11 },
      padding: 12,
      callbacks: {
        label: (ctx: any) => ' $' + ctx.parsed.x.toFixed(2),
      },
    },
  },
  scales: {
    x: {
      grid: { color: '#1e1e1b' },
      border: { color: 'transparent' },
      ticks: {
        color: '#5a5855',
        font: { family: 'JetBrains Mono', size: 10 },
        callback: (v: any) => '$' + Number(v).toFixed(0),
      },
    },
    y: {
      grid: { color: 'transparent' },
      border: { color: '#1e1e1b' },
      ticks: {
        color: '#8c8a84',
        font: { family: 'DM Sans', size: 11 },
        autoSkip: false,
      },
    },
  },
}

// Donut chart: share of total spend by family
const projectDonutData = computed(() => {
  if (!families.value.length) return null
  const top = families.value.slice(0, 8)
  const otherCost = families.value.slice(8).reduce((s, f) => s + f.rollup.total_cost, 0)
  const labels = top.map(f => f.displayName)
  const data = top.map(f => f.rollup.total_cost)
  if (otherCost > 0) {
    labels.push('Other')
    data.push(otherCost)
  }
  return {
    labels,
    datasets: [{
      data,
      backgroundColor: [...projectColors.slice(0, top.length), '#292524'],
      borderColor: '#0a0a09',
      borderWidth: 3,
    }],
  }
})

const donutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: '68%',
  animation: { duration: 800, easing: 'easeOutQuart' as const },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: '#1a1a18',
      borderColor: '#2a2a27',
      borderWidth: 1,
      titleColor: '#8c8a84',
      bodyColor: '#f0ede8',
      bodyFont: { family: 'JetBrains Mono', size: 12 },
      padding: 10,
      callbacks: {
        label: (ctx: any) => {
          const total = ctx.dataset.data.reduce((a: number, b: number) => a + b, 0)
          const pct = total > 0 ? Math.round((ctx.parsed / total) * 100) : 0
          return ` $${ctx.parsed.toFixed(2)} (${pct}%)`
        },
      },
    },
  },
}

// Monthly stacked bar — costs rolled up to family display names
const monthlyChartData = computed(() => {
  if (!monthlyData.value.length || !families.value.length) return null

  const months = [...new Set(monthlyData.value.map(d => d.month))].sort()
  const topFamilies = families.value.slice(0, 6)
  const topIds = new Set(topFamilies.map(f => f.id))
  const lookup = byProjectFamily.value

  // Aggregate raw monthly rows into family-month costs
  const familyMonthCost = new Map<string, number>()
  for (const d of monthlyData.value) {
    const family = lookup.get(d.project)
    const key = family
      ? `${family.id}\0${d.month}`
      : `orphan:${d.project}\0${d.month}`
    familyMonthCost.set(key, (familyMonthCost.get(key) || 0) + d.cost)
  }

  const datasets = topFamilies.map((family, i) => ({
    label: family.displayName,
    data: months.map(month => familyMonthCost.get(`${family.id}\0${month}`) || 0),
    backgroundColor: projectColors[i % projectColors.length],
    borderColor: 'transparent',
    borderWidth: 0,
  }))

  // "Other" = families outside the top 6 (plus any orphan rows)
  const hasOther = families.value.length > 6 ||
    monthlyData.value.some(d => !lookup.has(d.project))
  if (hasOther) {
    datasets.push({
      label: 'Other',
      data: months.map(month => {
        let sum = 0
        for (const [key, cost] of familyMonthCost) {
          const [fid, m] = key.split('\0')
          if (m !== month) continue
          if (topIds.has(fid)) continue
          sum += cost
        }
        return sum
      }),
      backgroundColor: '#292524',
      borderColor: 'transparent',
      borderWidth: 0,
    })
  }

  return {
    labels: months.map(m => {
      const [y, mo] = m.split('-')
      const d = new Date(Number(y), Number(mo) - 1)
      return d.toLocaleDateString('en-GB', { month: 'short', year: '2-digit' })
    }),
    datasets,
  }
})

const monthlyBarOptions = {
  responsive: true,
  maintainAspectRatio: false,
  animation: { duration: 700, easing: 'easeOutQuart' as const },
  plugins: {
    legend: {
      display: true,
      position: 'bottom' as const,
      labels: {
        color: '#8c8a84',
        font: { family: 'DM Sans', size: 11 },
        boxWidth: 8,
        boxHeight: 8,
        padding: 16,
      },
    },
    tooltip: {
      backgroundColor: '#1a1a18',
      borderColor: '#2a2a27',
      borderWidth: 1,
      titleColor: '#8c8a84',
      bodyColor: '#f0ede8',
      bodyFont: { family: 'JetBrains Mono', size: 12 },
      padding: 12,
      callbacks: {
        label: (ctx: any) => ` ${ctx.dataset.label}: $${ctx.parsed.y.toFixed(2)}`,
      },
    },
  },
  scales: {
    x: {
      stacked: true,
      grid: { color: 'transparent' },
      border: { color: '#1e1e1b' },
      ticks: {
        color: '#5a5855',
        font: { family: 'DM Sans', size: 11 },
      },
    },
    y: {
      stacked: true,
      grid: { color: '#1e1e1b' },
      border: { color: 'transparent' },
      ticks: {
        color: '#5a5855',
        font: { family: 'JetBrains Mono', size: 10 },
        callback: (v: any) => '$' + Number(v).toFixed(0),
      },
    },
  },
}

onMounted(async () => {
  // Table + bar/donut only need the cheap sessions rollup — paint those
  // first. Monthly spend walks the requests table (heavier) and fills in
  // after so the page doesn't sit blank waiting on it.
  projects.value = (await fetchProjects()) || []
  monthlyLoading.value = true
  try {
    monthlyData.value = (await fetchProjectMonthly()) || []
  } finally {
    monthlyLoading.value = false
  }
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

.charts-row {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: var(--space-5);
  margin-bottom: var(--space-6);
}

.chart-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  padding: var(--space-6);
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 100ms;
}
.chart-card.full-width {
  margin-bottom: var(--space-8);
  animation-delay: 200ms;
}
.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-5);
}
.chart-title {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}
.chart-meta {
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-tertiary);
}
.chart-canvas-wrap {
  height: 180px;
  position: relative;
}
.chart-canvas-wrap.tall {
  height: 260px;
}
.chart-canvas-wrap.bar-chart {
  /* Height set inline from barChartHeight so row count keeps labels readable. */
  min-height: 300px;
}
.chart-placeholder {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-disabled);
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
}

.donut-legend {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-top: var(--space-4);
}
.legend-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  gap: var(--space-3);
}
.legend-left {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
  min-width: 0;
}
.legend-left span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
  letter-spacing: 0.04em;
}
.legend-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
.legend-val {
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 300ms;
}
.section-title {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}

.sessions-table-wrap {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  overflow: hidden;
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 350ms;
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

tbody tr {
  border-bottom: 1px solid var(--border-subtle);
  transition: background 100ms;
}
tbody tr:last-child { border-bottom: none; }
tbody tr:hover { background: var(--bg-elevated); }

td {
  padding: var(--space-4) var(--space-5);
  color: var(--text-secondary);
  vertical-align: middle;
}
td.right { text-align: right; }
.rank {
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-disabled);
  width: 32px;
  text-align: right;
  padding-right: var(--space-2);
}
.rank.top { color: var(--amber-500); }

tr.family-row {
  background: rgba(245, 158, 11, 0.03);
  border-bottom-color: var(--border-default);
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
.member-pill {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10.5px;
  font-weight: 400;
  letter-spacing: 0;
  color: var(--text-tertiary);
  background: var(--bg-subtle);
  padding: 2px 6px;
  border: 1px solid var(--border-subtle);
}

tr.member-row {
  background: rgba(255, 255, 255, 0.012);
}
.member-name {
  padding-left: calc(var(--space-5) + 18px);
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-secondary);
}

.mono {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
}
.dim { color: var(--text-tertiary); }
.cost-cell {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  text-align: right;
}
.cost-cell.top { color: var(--amber-400); }
.cost-cell.dim-cost {
  color: var(--text-secondary);
  font-weight: 400;
}
</style>
