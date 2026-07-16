<template>
  <div class="donut-card">
    <div class="chart-header">
      <div class="chart-title-row">
        <div class="chart-title">{{ title }}</div>
        <slot name="header-action" />
      </div>
      <div v-if="subtitle" class="chart-subtitle">{{ subtitle }}</div>
    </div>
    <div v-if="hasData" class="donut-wrap">
      <Doughnut :data="chartData" :options="chartOptions" />
    </div>
    <div v-else class="donut-empty">{{ emptyText || '—' }}</div>
    <div v-if="hasData" class="donut-legend">
      <div v-for="(item, i) in legendItems" :key="i" class="legend-row">
        <div class="legend-left">
          <div class="legend-dot" :style="{ background: item.color }"></div>
          <span class="legend-label" :title="item.label">{{ item.label }}</span>
        </div>
        <div class="legend-val">{{ formatValue(item.value) }} <span class="legend-pct">{{ item.pct }}%</span></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Doughnut } from 'vue-chartjs'
import {
  Chart as ChartJS,
  ArcElement,
  Tooltip,
} from 'chart.js'
import { formatCostDisplay, formatCostWhole } from '../../composables/useFormatCost'

ChartJS.register(ArcElement, Tooltip)

export interface DonutSlice {
  label: string
  value: number
  color: string
}

const props = defineProps<{
  title: string
  subtitle?: string
  slices: DonutSlice[]
  emptyText?: string
  /** Round legend/tooltip amounts to whole dollars. */
  whole?: boolean
}>()

function formatValue(value: number): string {
  return props.whole ? formatCostWhole(value) : formatCostDisplay(value)
}

const total = computed(() => props.slices.reduce((s, x) => s + x.value, 0))
const hasData = computed(() => total.value > 0)

const legendItems = computed(() =>
  props.slices.map(s => ({
    ...s,
    pct: total.value > 0 ? Math.round((s.value / total.value) * 100) : 0,
  })),
)

const chartData = computed(() => ({
  labels: props.slices.map(s => s.label),
  datasets: [{
    data: props.slices.map(s => s.value),
    backgroundColor: props.slices.map(s => s.color),
    borderColor: '#0a0a09',
    borderWidth: 3,
    hoverBorderColor: '#0a0a09',
  }],
}))

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '72%',
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
      titleFont: { family: 'DM Sans', size: 11 },
      padding: 10,
      callbacks: {
        label: (ctx: any) => {
          const t = ctx.dataset.data.reduce((a: number, b: number) => a + b, 0)
          const pct = t > 0 ? Math.round((ctx.parsed / t) * 100) : 0
          const amount = props.whole
            ? '$' + Math.round(ctx.parsed).toLocaleString('en-US')
            : '$' + ctx.parsed.toFixed(2)
          return `  ${amount} (${pct}%)`
        },
      },
    },
  },
}))
</script>

<style scoped>
.donut-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  padding: var(--space-6);
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 320ms;
  display: flex;
  flex-direction: column;
}
.chart-header {
  margin-bottom: var(--space-5);
}
.chart-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}
.chart-title {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}
.chart-subtitle {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10.5px;
  color: var(--text-disabled);
  margin-top: 4px;
}
.donut-wrap {
  height: 150px;
  position: relative;
  margin-bottom: var(--space-5);
}
.donut-empty {
  height: 150px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-disabled);
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  margin-bottom: var(--space-5);
}
.donut-legend {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
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
.legend-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
.legend-pct {
  color: var(--text-disabled);
  font-size: 10px;
}
</style>
