import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Summary, Session, WsEvent } from '../types'
import { fetchSummary, fetchRecent } from '../api'

export const useDashboardStore = defineStore('dashboard', () => {
  const summary = ref<Summary | null>(null)
  const recentSessions = ref<Session[]>([])
  const loaded = ref(false)

  // Daily spend is owned by DailySpendChart itself so it can refetch when the
  // user changes the time-range dropdown without going through the store.
  async function load() {
    // Summary is fetched independently so a failure in recent sessions
    // can't leave the dashboard meters blank — refreshSummary already follows
    // this pattern; load() should too.
    try {
      summary.value = await fetchSummary()
    } catch {
      // Keep any websocket-provided summary if the REST call fails.
    }
    try {
      recentSessions.value = (await fetchRecent(10)) || []
    } catch {
      recentSessions.value = []
    }
    loaded.value = true
  }

  // Summary-only refresh used by surfaces (e.g. the window-bar re-sync
  // button) that just need the bucket.state honest-state classification
  // to redraw after a backend status change. Narrower than load() so an
  // unrelated endpoint failure can't fail the refresh, and so we don't
  // refetch recent sessions on every manual sync.
  async function refreshSummary() {
    summary.value = await fetchSummary()
  }

  function applyEvent(event: WsEvent) {
    switch (event.type) {
      case 'summary.updated':
        if (event.payload) {
          const p = typeof event.payload === 'string'
            ? JSON.parse(event.payload)
            : event.payload
          if (p && typeof p === 'object') {
            summary.value = {
              ...(summary.value ?? {}),
              ...p,
              window_5h: p.window_5h ?? summary.value?.window_5h,
              window_7d: p.window_7d ?? summary.value?.window_7d,
              today: p.today ?? summary.value?.today,
              month: p.month ?? summary.value?.month,
            } as Summary
          }
        }
        break

      case 'session.updated':
        if (event.payload) {
          const rIdx = recentSessions.value.findIndex(s => s.id === event.payload.id)
          if (rIdx >= 0) {
            recentSessions.value[rIdx] = event.payload
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

  return { summary, recentSessions, loaded, load, refreshSummary, applyEvent }
})
