<template>
  <div class="chart-card">
    <div class="chart-header">
      <div class="chart-title">Daily Spend — Last {{ windowLabel }}</div>
      <div class="chart-controls">
        <select v-model.number="windowDays" @change="reload" class="window-select" aria-label="Time range">
          <option v-for="opt in windowOptions" :key="opt.days" :value="opt.days">{{ opt.label }}</option>
        </select>
        <div class="chart-meta">{{ totalStr }} total</div>
      </div>
    </div>
    <div class="chart-canvas-wrap">
      <Bar v-if="chartData" :data="chartData" :options="chartOptions" />
      <div v-else class="chart-empty">{{ loadError || 'No spend in this range' }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Tooltip,
} from 'chart.js'
import type { DailySpend } from '../../types'
import { formatCostDisplay } from '../../composables/useFormatCost'
import { fetchDaily } from '../../api'

const router = useRouter()

ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip)

const windowOptions = [
  { days: 7, label: 'Week' },
  { days: 30, label: 'Month' },
  { days: 60, label: '2 Months' },
  { days: 90, label: 'Quarter' },
  { days: 180, label: '6 Months' },
  { days: 365, label: 'Year' },
]

const windowDays = ref(30)
const data = ref<DailySpend[]>([])
const loadError = ref('')

const windowLabel = computed(
  () => windowOptions.find(o => o.days === windowDays.value)?.label ?? `${windowDays.value} Days`,
)

async function reload() {
  loadError.value = ''
  try {
    data.value = (await fetchDaily(windowDays.value)) ?? []
  } catch {
    data.value = []
    loadError.value = 'Failed to load daily spend'
  }
}

onMounted(reload)

const totalStr = computed(() => {
  const total = data.value.reduce((sum, d) => sum + d.cost, 0)
  return formatCostDisplay(total)
})

const chartData = computed(() => {
  if (!data.value.length) return null

  const labels = data.value.map((d, i) => {
    if (i === data.value.length - 1) return 'Today'
    // d.date is "YYYY-MM-DD" representing a local calendar day. `new Date(s)`
    // parses bare dates as UTC midnight, which then renders as the previous
    // day in any timezone west of UTC. Append T00:00:00 to force local-zone
    // parsing so the label matches what the backend bucketed.
    const date = new Date(d.date + 'T00:00:00')
    return date.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })
  })

  const values = data.value.map(d => d.cost)
  const colors = values.map((_, i) =>
    i === values.length - 1 ? 'rgba(251,191,36,1)' : 'rgba(245,158,11,0.55)'
  )

  return {
    labels,
    datasets: [{
      data: values,
      backgroundColor: colors,
      borderColor: 'transparent',
      borderWidth: 0,
      borderRadius: 0,
    }],
  }
})

function handleBarClick(_event: any, elements: any[]) {
  if (!elements.length) return
  const idx = elements[0].index
  const point = data.value[idx]
  if (!point?.date) return
  router.push({ path: '/sessions', query: { date: point.date } })
}

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  onClick: handleBarClick,
  onHover: (event: any, elements: any[]) => {
    const target = event?.native?.target
    if (target) target.style.cursor = elements.length ? 'pointer' : 'default'
  },
  animation: {
    duration: 700,
    easing: 'easeOutQuart' as const,
    delay: (ctx: any) => ctx.dataIndex * 18,
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
        label: (ctx: any) => ' $' + ctx.parsed.y.toFixed(4),
      },
    },
  },
  scales: {
    x: {
      grid: { color: 'transparent' },
      border: { color: '#1e1e1b' },
      ticks: {
        color: '#5a5855',
        font: { family: 'DM Sans', size: 10 },
        maxRotation: 0,
        maxTicksLimit: 8,
      },
    },
    y: {
      grid: { color: '#1e1e1b' },
      border: { color: 'transparent' },
      ticks: {
        color: '#5a5855',
        font: { family: 'JetBrains Mono', size: 10 },
        callback: (v: any) => '$' + Number(v).toFixed(2),
        maxTicksLimit: 5,
      },
    },
  },
}
</script>

<style scoped>
.chart-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  padding: var(--space-6);
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 280ms;
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
.chart-controls {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}
.window-select {
  background: var(--bg-subtle);
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  padding: 4px 8px;
  cursor: pointer;
  transition: border-color 120ms, color 120ms;
}
.window-select:hover {
  border-color: var(--amber-500);
  color: var(--text-primary);
}
.window-select:focus {
  outline: none;
  border-color: var(--amber-500);
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
.chart-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-disabled);
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
}
</style>
