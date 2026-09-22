import { defineStore } from 'pinia'
import { ref } from 'vue'

import {
  type DashboardReport,
  type MovementSummaryReport,
  type ReportFilter,
  type ValuationReport,
  type VelocityReport,
  getDashboardReport,
  getMovementSummaryReport,
  getValuationReport,
  getVelocityReport,
} from '@/api/reports'

export const useReportsStore = defineStore('reports', () => {
  const dashboard = ref<DashboardReport | null>(null)
  const valuation = ref<ValuationReport | null>(null)
  const movementSummary = ref<MovementSummaryReport | null>(null)
  const velocity = ref<VelocityReport | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Reports are read together on the dashboard, so one flag and one error
  // message cover the whole screen rather than four independent spinners.
  async function fetchAll(filter: ReportFilter = {}) {
    loading.value = true
    error.value = null
    try {
      const [dashboardResult, valuationResult, summaryResult, velocityResult] = await Promise.all([
        getDashboardReport(filter),
        getValuationReport({ ...filter, limit: filter.limit ?? 50 }),
        getMovementSummaryReport(filter),
        getVelocityReport({ ...filter, limit: 8 }),
      ])
      dashboard.value = dashboardResult
      valuation.value = valuationResult
      movementSummary.value = summaryResult
      velocity.value = velocityResult
    } catch (e: any) {
      error.value = e.message || 'Failed to load reports'
    } finally {
      loading.value = false
    }
  }

  return { dashboard, valuation, movementSummary, velocity, loading, error, fetchAll }
})
