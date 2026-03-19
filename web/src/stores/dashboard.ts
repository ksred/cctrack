import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Summary, Session, DailySpend, WsEvent } from '../types'
import { fetchSummary, fetchDaily, fetchRecent, fetchSessions } from '../api'

export const useDashboardStore = defineStore('dashboard', () => {
  const summary = ref<Summary | null>(null)
  const daily = ref<DailySpend[]>([])
  const recentSessions = ref<Session[]>([])
  const topSessions = ref<Session[]>([])
  const loaded = ref(false)
  const lastUpdated = ref<Date | null>(null)

  async function load() {
    const [s, d, recent, top] = await Promise.all([
      fetchSummary(),
      fetchDaily(30),
      fetchRecent(10),
      fetchSessions(5, 0, 'cost', 'desc'),
    ])
    summary.value = s
    daily.value = d
    recentSessions.value = recent || []
    topSessions.value = top.sessions || []
    loaded.value = true
    lastUpdated.value = new Date()
  }

  function applyEvent(event: WsEvent) {
    switch (event.type) {
      case 'summary.updated':
        if (event.payload) {
          summary.value = {
            ...summary.value,
            ...event.payload,
          } as Summary
          lastUpdated.value = new Date()
        }
        break

      case 'session.updated':
        if (event.payload) {
          // Update in recent sessions
          const rIdx = recentSessions.value.findIndex(s => s.id === event.payload.id)
          if (rIdx >= 0) {
            recentSessions.value[rIdx] = event.payload
          }
          // Update in top sessions
          const tIdx = topSessions.value.findIndex(s => s.id === event.payload.id)
          if (tIdx >= 0) {
            topSessions.value[tIdx] = event.payload
          }
        }
        break

      case 'session.created':
        if (event.payload) {
          recentSessions.value.unshift(event.payload)
          if (recentSessions.value.length > 10) {
            recentSessions.value.pop()
          }
        }
        break

      case 'ping':
        break
    }
  }

  return { summary, daily, recentSessions, topSessions, loaded, lastUpdated, load, applyEvent }
})
