<template>
  <div class="stat-card" :class="{ highlight }">
    <div class="stat-label">{{ label }}</div>
    <div class="stat-value">{{ formattedValue }}</div>
    <div class="stat-sub">
      <span v-if="tokens !== undefined">{{ formatTokens(tokens) }} tokens</span>
      <span v-else-if="subtext" style="color: var(--text-tertiary)">{{ subtext }}</span>
    </div>
    <div v-if="trendPct !== undefined && trendPct !== null" class="stat-trend" :class="trendClass">
      <span class="trend-arrow">{{ trendPct > 0 ? '↑' : trendPct < 0 ? '↓' : '→' }}</span>
      <span>
        {{ Math.abs(trendPct) }}% vs {{ prevName || trendLabel }}<template v-if="prevAmount !== undefined"> · {{ formatPrev(prevAmount) }}</template>
      </span>
    </div>
    <div v-if="budget && budget > 0" class="budget-bar-wrap">
      <div class="budget-bar-fill" :style="{ width: budgetPct + '%', background: budgetColor }"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, toRef } from 'vue'
import { useCountUp } from '../../composables/useCountUp'
import { formatTokens, formatCostDisplay, formatCostWhole } from '../../composables/useFormatCost'

const props = defineProps<{
  label: string
  value: number
  tokens?: number
  highlight?: boolean
  budget?: number
  subtext?: string
  trendPct?: number | null
  trendLabel?: string
  // Optional: when present, render `(prevName · prevAmount)` after the trend
  // percentage so the user sees what they're being compared against.
  prevName?: string
  prevAmount?: number
  /** Round to whole dollars (no cents) — for larger aggregates. */
  whole?: boolean
}>()

const targetValue = toRef(props, 'value')
const animated = useCountUp(targetValue)

function formatPrev(amount: number): string {
  return props.whole ? formatCostWhole(amount) : formatCostDisplay(amount)
}

const formattedValue = computed(() => {
  // Target value drives the format; useCountUp animates `animated.value`
  // through small intermediate numbers that we don't want to render as
  // $0.0000 for the entire animation when the target is 0.
  if (props.whole) {
    if (props.value === 0) return '$0'
    return formatCostWhole(animated.value)
  }
  if (props.value === 0) return '$0.00'
  const v = animated.value
  if (v < 0.01) return '$' + v.toFixed(4)
  return '$' + v.toFixed(2)
})

const trendClass = computed(() => {
  if (props.trendPct === undefined || props.trendPct === null) return ''
  if (props.trendPct > 10) return 'trend-up'
  if (props.trendPct < -10) return 'trend-down'
  return 'trend-flat'
})

const budgetPct = computed(() => {
  if (!props.budget || props.budget <= 0) return 0
  return Math.min((props.value / props.budget) * 100, 100)
})

const budgetColor = computed(() => {
  if (budgetPct.value >= 100) return 'var(--cost-high)'
  if (budgetPct.value >= 80) return 'var(--cost-mid)'
  return 'var(--amber-500)'
})
</script>

<style scoped>
.stat-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  padding: var(--space-6);
  position: relative;
  overflow: hidden;
  animation: fadeSlideUp 0.45s ease both;
}
.stat-card:nth-child(1) { animation-delay: 40ms; }
.stat-card:nth-child(2) { animation-delay: 100ms; }
.stat-card:nth-child(3) { animation-delay: 160ms; }
.stat-card:nth-child(4) { animation-delay: 220ms; }

.stat-card.highlight::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--amber-500);
}
.stat-card.highlight {
  background: linear-gradient(135deg, var(--amber-glow-sm), transparent 60%);
}

.stat-label {
  font-size: 10.5px;
  font-weight: 500;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  margin-bottom: var(--space-3);
}
.stat-value {
  font-family: 'Bebas Neue', sans-serif;
  font-size: 64px;
  line-height: 0.9;
  color: var(--text-primary);
  letter-spacing: 0.01em;
  margin-bottom: var(--space-3);
}
.stat-card.highlight .stat-value {
  color: var(--amber-400);
}
.stat-sub {
  font-family: 'JetBrains Mono', monospace;
  font-size: 11.5px;
  color: var(--text-tertiary);
}
.stat-sub span {
  color: var(--text-secondary);
}
.stat-trend {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10.5px;
  margin-top: var(--space-2);
  color: var(--text-tertiary);
}
.stat-trend.trend-up { color: var(--cost-high); }
.stat-trend.trend-down { color: #4ade80; }
.stat-trend.trend-flat { color: var(--text-tertiary); }
.trend-arrow {
  margin-right: 3px;
}
.budget-bar-wrap {
  margin-top: var(--space-4);
  height: 2px;
  background: var(--bg-subtle);
  position: relative;
  overflow: hidden;
}
.budget-bar-fill {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  transition: width 600ms cubic-bezier(0.16, 1, 0.3, 1);
}
</style>
