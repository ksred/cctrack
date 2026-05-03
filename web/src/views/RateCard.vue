<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Rate Card</h1>
      <div class="page-meta">
        {{ version || '—' }}<span v-if="updated"> · updated {{ updated }}</span>
      </div>
    </div>

    <div class="rate-table-wrap" v-if="rates.length">
      <table>
        <thead>
          <tr>
            <th>Model</th>
            <th>Released</th>
            <th class="right">Input</th>
            <th class="right">Output</th>
            <th class="right">Cache Read</th>
            <th class="right">Cache Write 5m</th>
            <th class="right">Cache Write 1h</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="rate in rates" :key="rate.Family">
            <td class="model-name">{{ rate.Family }}</td>
            <td class="released">{{ rate.Released || '—' }}</td>
            <td class="price right">${{ rate.InputPerMToken.toFixed(2) }}</td>
            <td class="price right">${{ rate.OutputPerMToken.toFixed(2) }}</td>
            <td class="price right">${{ rate.CacheReadPerMToken.toFixed(2) }}</td>
            <td class="price right">${{ rate.CacheWrite5mPerMToken.toFixed(2) }}</td>
            <td class="price right">${{ rate.CacheWrite1hPerMToken.toFixed(2) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <p class="rate-note">
      All prices per million tokens. Rates are bundled with the binary — update cctrack to get the latest rates.
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { ModelRate } from '../types'
import { fetchRates } from '../api'

const rates = ref<ModelRate[]>([])
const version = ref('')
const updated = ref('')

onMounted(async () => {
  const r = await fetchRates()
  rates.value = r.rates
  version.value = r.version
  updated.value = r.updated
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

.rate-table-wrap {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  overflow: hidden;
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 100ms;
}
table { width: 100%; font-size: 13px; }
thead th {
  padding: var(--space-4) var(--space-5);
  text-align: left;
  font-size: 10.5px;
  font-weight: 500;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  border-bottom: 1px solid var(--border-subtle);
}
thead th.right { text-align: right; }
tbody tr {
  border-bottom: 1px solid var(--border-subtle);
}
tbody tr:last-child { border-bottom: none; }
td {
  padding: var(--space-4) var(--space-5);
  color: var(--text-secondary);
}
td.right { text-align: right; }
.model-name {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-primary);
}
.released {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-tertiary);
  white-space: nowrap;
}
.price {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
}
.rate-note {
  margin-top: var(--space-6);
  font-size: 13px;
  color: var(--text-tertiary);
}
</style>
