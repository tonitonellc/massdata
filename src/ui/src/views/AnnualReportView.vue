<script setup>
import Chart from 'chart.js/auto'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { chartColors } from '../utils/chartUtils'
import { getApiUrl } from '../utils/env'
import { formatCurrency, formatNumber } from '../utils/format'

// ─── Year lists ───────────────────────────────────────────────────────────
const CY = new Date().getFullYear()
const ALL_SPENDING_YEARS = Array.from({ length: CY - 2009 }, (_, i) => String(2010 + i))
const ALL_REVENUE_YEARS  = Array.from({ length: CY - 2009 }, (_, i) => String(2010 + i))

const selectedSpendingYears = ref(ALL_SPENDING_YEARS.slice(-3))
const selectedRevenueYears  = ref(ALL_REVENUE_YEARS.slice(-3))

// ─── State ────────────────────────────────────────────────────────────────
const loading  = ref(false)
const error    = ref('')
const spendingData = ref([])
const revenueData  = ref([])
const showSpending = ref(true)
const showRevenue  = ref(true)

// ─── Chart refs ───────────────────────────────────────────────────────────
const spendingTotalRef = ref(null)
const spendingDeptRef  = ref(null)
const revenueTotalRef  = ref(null)
const revenueDeptRef   = ref(null)
const charts = {}

// ─── Per-dataset colors (cycle if >6 years selected) ─────────────────────
const YEAR_COLORS = [
  chartColors.primary, chartColors.blue,   chartColors.teal,
  chartColors.orange,  chartColors.green,  chartColors.red,
  chartColors.yellow,  chartColors.purple,
]
const color = (i, offset = 0) => YEAR_COLORS[(i + offset) % YEAR_COLORS.length]

// ─── Aggregation helpers ──────────────────────────────────────────────────
function topDepts(results, n = 8) {
  const totals = new Map()
  results.forEach(yr => yr.by_dept.forEach(d =>
    totals.set(d.dept, (totals.get(d.dept) || 0) + d.total)
  ))
  return Array.from(totals.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, n)
    .map(([dept]) => dept)
}

function yoyChange(results) {
  const out = []
  for (let i = 1; i < results.length; i++) {
    const prev = results[i - 1].total, curr = results[i].total
    out.push(prev > 0 ? ((curr - prev) / prev) * 100 : null)
  }
  return out
}

const spendingTopDepts = computed(() => topDepts(spendingData.value))
const revenueTopDepts  = computed(() => topDepts(revenueData.value))
const spendingYoY      = computed(() => yoyChange(spendingData.value))
const revenueYoY       = computed(() => yoyChange(revenueData.value))

// ─── Table sort ───────────────────────────────────────────────────────────
const spendingSort = reactive({ field: null, dir: 'asc' })
const revenueSort  = reactive({ field: null, dir: 'asc' })

function tableSortBy(sortState, field) {
  if (sortState.field === field) sortState.dir = sortState.dir === 'asc' ? 'desc' : 'asc'
  else { sortState.field = field; sortState.dir = 'asc' }
}
function sortIndicator(sortState, field) {
  return sortState.field !== field ? '' : sortState.dir === 'asc' ? '↑' : '↓'
}
function sortedDepts(depts, results, sortState) {
  const { field, dir } = sortState
  if (!field) return depts
  return [...depts].sort((a, b) => {
    let valA, valB
    if (field === 'dept') { valA = a.toLowerCase(); valB = b.toLowerCase() }
    else {
      const yr = results.find(r => r.year === field)
      valA = yr?.by_dept.find(d => d.dept === a)?.total ?? 0
      valB = yr?.by_dept.find(d => d.dept === b)?.total ?? 0
    }
    if (valA < valB) return dir === 'asc' ? -1 : 1
    if (valA > valB) return dir === 'asc' ? 1 : -1
    return 0
  })
}

const sortedSpendingDepts = computed(() => sortedDepts(spendingTopDepts.value, spendingData.value, spendingSort))
const sortedRevenueDepts  = computed(() => sortedDepts(revenueTopDepts.value, revenueData.value, revenueSort))

// ─── Fetch ────────────────────────────────────────────────────────────────
const loadReport = async () => {
  if (!selectedSpendingYears.value.length && !selectedRevenueYears.value.length) return
  loading.value = true
  error.value = ''
  spendingData.value = []
  revenueData.value  = []

  const params = new URLSearchParams()
  if (selectedSpendingYears.value.length) params.set('spending_years', selectedSpendingYears.value.join(','))
  if (selectedRevenueYears.value.length)  params.set('revenue_years',  selectedRevenueYears.value.join(','))

  try {
    const resp = await fetch(getApiUrl(`/api/mass-annual?${params}`))
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = await resp.json()
    if (data.error) throw new Error(data.error)
    spendingData.value = (data.spending || []).sort((a, b) => a.year.localeCompare(b.year))
    revenueData.value  = (data.revenue  || []).sort((a, b) => a.year.localeCompare(b.year))
  } catch (e) {
    error.value = `Error loading annual data: ${e.message}`
  } finally {
    loading.value = false
  }
}

// ─── Chart helpers ────────────────────────────────────────────────────────
function isDark()  { return document.body.classList.contains('dark-mode') }
function textCol() { return isDark() ? '#94a3b8' : '#4a5568' }
function gridCol() { return isDark() ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)' }

function yDollarTick(v) {
  const a = Math.abs(v)
  if (a >= 1e9) return '$' + formatNumber(+(v / 1e9).toFixed(1)) + 'B'
  if (a >= 1e6) return '$' + formatNumber(+(v / 1e6).toFixed(1)) + 'M'
  return '$' + formatNumber(Math.round(v / 1000)) + 'k'
}

function truncate(s, n = 22) { return s && s.length > n ? s.slice(0, n) + '…' : (s || '') }

const baseOpts = () => ({
  responsive: true, maintainAspectRatio: false,
  scales: {
    y: { beginAtZero: true, ticks: { color: textCol(), callback: yDollarTick }, grid: { color: gridCol() } },
    x: { ticks: { color: textCol() } },
  },
})

function buildTotalChart(el, results, offset) {
  if (!el?.getContext('2d') || !results.length) return null
  return new Chart(el, {
    type: 'bar',
    data: {
      labels: results.map(r => `FY${r.year}`),
      datasets: [{ data: results.map(r => r.total), backgroundColor: results.map((_, i) => color(i, offset)), borderRadius: 4 }],
    },
    options: {
      ...baseOpts(),
      plugins: {
        legend: { display: false },
        tooltip: { callbacks: { label: ctx => ` $${formatCurrency(ctx.parsed.y)}` } },
      },
    },
  })
}

function buildDeptChart(el, results, depts, offset) {
  if (!el?.getContext('2d') || !depts.length) return null
  const tc = textCol()
  return new Chart(el, {
    type: 'bar',
    data: {
      labels: depts.map(d => truncate(d)),
      datasets: results.map((r, i) => ({
        label: `FY${r.year}`,
        data: depts.map(dept => r.by_dept.find(d => d.dept === dept)?.total ?? 0),
        backgroundColor: color(i, offset),
        borderRadius: 3,
      })),
    },
    options: {
      ...baseOpts(),
      scales: {
        y: { beginAtZero: true, ticks: { color: textCol(), callback: yDollarTick }, grid: { color: gridCol() } },
        x: { ticks: { color: textCol(), maxRotation: 40, font: { size: 11 } } },
      },
      plugins: {
        legend: { labels: { color: tc, boxWidth: 12 } },
        tooltip: {
          callbacks: {
            title: items => depts[items[0].dataIndex],
            label:  ctx  => ` ${ctx.dataset.label}: $${formatCurrency(ctx.parsed.y)}`,
          },
        },
      },
    },
  })
}

const renderCharts = async () => {
  await nextTick()
  Object.values(charts).forEach(c => c?.destroy())

  if (spendingData.value.length) {
    charts.spendingTotal = buildTotalChart(spendingTotalRef.value, spendingData.value, 0)
    charts.spendingDept  = buildDeptChart(spendingDeptRef.value, spendingData.value, spendingTopDepts.value, 0)
  }
  if (revenueData.value.length) {
    charts.revenueTotal = buildTotalChart(revenueTotalRef.value, revenueData.value, 3)
    charts.revenueDept  = buildDeptChart(revenueDeptRef.value, revenueData.value, revenueTopDepts.value, 3)
  }
}

watch([spendingData, revenueData], renderCharts)

// ─── Year selector helpers ────────────────────────────────────────────────
// Note: in Vue 3 templates, refs auto-unwrap, so we use explicit handlers
// per list rather than passing the ref as a parameter.
const onAddSpending = (e) => {
  const yr = e.target.value
  if (yr && !selectedSpendingYears.value.includes(yr) && selectedSpendingYears.value.length < 6)
    selectedSpendingYears.value = [...selectedSpendingYears.value, yr].sort()
  e.target.value = ''
}
const onAddRevenue = (e) => {
  const yr = e.target.value
  if (yr && !selectedRevenueYears.value.includes(yr) && selectedRevenueYears.value.length < 6)
    selectedRevenueYears.value = [...selectedRevenueYears.value, yr].sort()
  e.target.value = ''
}
const onRemoveSpending = (yr) => {
  if (selectedSpendingYears.value.length > 1)
    selectedSpendingYears.value = selectedSpendingYears.value.filter(y => y !== yr)
}
const onRemoveRevenue = (yr) => {
  if (selectedRevenueYears.value.length > 1)
    selectedRevenueYears.value = selectedRevenueYears.value.filter(y => y !== yr)
}

// ─── Lifecycle ────────────────────────────────────────────────────────────
onMounted(loadReport)
onBeforeUnmount(() => Object.values(charts).forEach(c => c?.destroy()))
</script>

<template>
  <div class="data-explorer">
    <div class="explorer-header">
      <h1>Annual Reports</h1>
      <p class="subtitle">Comparing Commonwealth of Massachusetts spending and revenue across years.</p>
    </div>

    <!-- Year selectors -->
    <div class="filters">
      <div class="filter-group">
        <label>Spending Fiscal Years:</label>
        <div class="multi-select-container">
          <div class="selected-items">
            <span v-if="!selectedSpendingYears.length" class="placeholder">Select years…</span>
            <span v-for="yr in selectedSpendingYears" :key="yr" class="selected-chip">
              FY{{ yr }}
              <button @click="onRemoveSpending(yr)" class="remove-chip">&times;</button>
            </span>
          </div>
          <select @change="onAddSpending" class="multi-select">
            <option value="">Add year…</option>
            <option v-for="yr in ALL_SPENDING_YEARS" :key="yr" :value="yr"
              :disabled="selectedSpendingYears.includes(yr)">FY{{ yr }}</option>
          </select>
        </div>
      </div>

      <div class="filter-group">
        <label>Revenue Fiscal Years:</label>
        <div class="multi-select-container">
          <div class="selected-items">
            <span v-if="!selectedRevenueYears.length" class="placeholder">Select years…</span>
            <span v-for="yr in selectedRevenueYears" :key="yr" class="selected-chip">
              FY{{ yr }}
              <button @click="onRemoveRevenue(yr)" class="remove-chip">&times;</button>
            </span>
          </div>
          <select @change="onAddRevenue" class="multi-select">
            <option value="">Add year…</option>
            <option v-for="yr in ALL_REVENUE_YEARS" :key="yr" :value="yr"
              :disabled="selectedRevenueYears.includes(yr)">FY{{ yr }}</option>
          </select>
        </div>
      </div>

      <div class="button-group">
        <button class="search-btn" @click="loadReport" :disabled="loading">
          {{ loading ? 'Loading…' : 'Load Report' }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading"><div class="spinner"></div></div>
    <div v-else-if="error" class="error"><p>{{ error }}</p></div>

    <template v-else-if="spendingData.length || revenueData.length">

      <!-- ── SPENDING ──────────────────────────────────────────────────────── -->
      <div class="section-header" @click="showSpending = !showSpending">
        <span class="chevron">{{ showSpending ? '▼' : '▶' }}</span>
        <h2>Annual Spending Overview</h2>
      </div>
      <section v-show="showSpending" class="report-section">
        <template v-if="spendingData.length">
          <!-- per-year totals -->
          <div class="stats-grid">
            <div v-for="r in spendingData" :key="r.year" class="stat-card">
              <div class="stat-label">FY{{ r.year }} Total Spending</div>
              <div class="stat-value">${{ formatCurrency(r.total) }}</div>
              <div class="stat-sub">{{ formatNumber(r.count) }} transactions</div>
            </div>
          </div>

          <!-- YoY change -->
          <div v-if="spendingData.length > 1" class="stats-grid yoy-grid">
            <div v-for="(pct, i) in spendingYoY" :key="i" class="stat-card">
              <div class="stat-label">FY{{ spendingData[i].year }} → FY{{ spendingData[i + 1].year }}</div>
              <div class="stat-value" :class="pct !== null && pct >= 0 ? 'change-positive' : 'change-negative'">
                {{ pct !== null ? (pct >= 0 ? '+' : '') + pct.toFixed(1) + '%' : 'N/A' }}
              </div>
              <div class="stat-sub">year-over-year</div>
            </div>
          </div>

          <!-- charts -->
          <div class="charts-grid">
            <div class="chart-card">
              <h3>Total Spending by Year</h3>
              <div class="chart-wrap"><canvas ref="spendingTotalRef"></canvas></div>
            </div>
            <div class="chart-card">
              <h3>Top Departments by Spending</h3>
              <div class="chart-wrap"><canvas ref="spendingDeptRef"></canvas></div>
            </div>
          </div>

          <!-- breakdown table -->
          <div class="report-table-scroll">
            <table class="data-table">
              <thead>
                <tr>
                  <th @click="tableSortBy(spendingSort, 'dept')" style="cursor:pointer">
                    Department {{ sortIndicator(spendingSort, 'dept') }}
                  </th>
                  <th v-for="r in spendingData" :key="r.year"
                    @click="tableSortBy(spendingSort, r.year)"
                    style="text-align:right; cursor:pointer">
                    FY{{ r.year }} {{ sortIndicator(spendingSort, r.year) }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="dept in sortedSpendingDepts" :key="dept">
                  <td data-label="Department">{{ dept }}</td>
                  <td v-for="r in spendingData" :key="r.year" :data-label="'FY' + r.year"
                    style="text-align:right; font-variant-numeric: tabular-nums">
                    ${{ formatCurrency(r.by_dept.find(d => d.dept === dept)?.total ?? 0) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </section>

      <!-- ── REVENUE ───────────────────────────────────────────────────────── -->
      <div class="section-header" @click="showRevenue = !showRevenue">
        <span class="chevron">{{ showRevenue ? '▼' : '▶' }}</span>
        <h2>Annual Revenue Overview</h2>
      </div>
      <section v-show="showRevenue" class="report-section">
        <template v-if="revenueData.length">
          <div class="stats-grid">
            <div v-for="r in revenueData" :key="r.year" class="stat-card">
              <div class="stat-label">FY{{ r.year }} Total Revenue</div>
              <div class="stat-value">${{ formatCurrency(r.total) }}</div>
              <div class="stat-sub">{{ formatNumber(r.count) }} records</div>
            </div>
          </div>

          <div v-if="revenueData.length > 1" class="stats-grid yoy-grid">
            <div v-for="(pct, i) in revenueYoY" :key="i" class="stat-card">
              <div class="stat-label">FY{{ revenueData[i].year }} → FY{{ revenueData[i + 1].year }}</div>
              <div class="stat-value" :class="pct !== null && pct >= 0 ? 'change-positive' : 'change-negative'">
                {{ pct !== null ? (pct >= 0 ? '+' : '') + pct.toFixed(1) + '%' : 'N/A' }}
              </div>
              <div class="stat-sub">year-over-year</div>
            </div>
          </div>

          <div class="charts-grid">
            <div class="chart-card">
              <h3>Total Revenue by Year</h3>
              <div class="chart-wrap"><canvas ref="revenueTotalRef"></canvas></div>
            </div>
            <div class="chart-card">
              <h3>Top Departments by Revenue</h3>
              <div class="chart-wrap"><canvas ref="revenueDeptRef"></canvas></div>
            </div>
          </div>

          <div class="report-table-scroll">
            <table class="data-table">
              <thead>
                <tr>
                  <th @click="tableSortBy(revenueSort, 'dept')" style="cursor:pointer">
                    Department {{ sortIndicator(revenueSort, 'dept') }}
                  </th>
                  <th v-for="r in revenueData" :key="r.year"
                    @click="tableSortBy(revenueSort, r.year)"
                    style="text-align:right; cursor:pointer">
                    FY{{ r.year }} {{ sortIndicator(revenueSort, r.year) }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="dept in sortedRevenueDepts" :key="dept">
                  <td data-label="Department">{{ dept }}</td>
                  <td v-for="r in revenueData" :key="r.year" :data-label="'FY' + r.year"
                    style="text-align:right; font-variant-numeric: tabular-nums">
                    ${{ formatCurrency(r.by_dept.find(d => d.dept === dept)?.total ?? 0) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </section>

    </template>

    <div v-else-if="!loading" class="empty-state">
      <p>Select fiscal years above and click Load Report to generate the annual comparison.</p>
    </div>

  </div>
</template>

<style scoped>
/* Year chip selector */
.multi-select-container {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  min-width: 220px;
}

.selected-items {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  min-height: 36px;
  align-items: center;
}

.placeholder { color: #9ca3af; font-size: 0.875rem; }

.selected-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  background: #667eea;
  color: white;
  padding: 0.25rem 0.6rem;
  border-radius: 20px;
  font-size: 0.8rem;
  font-weight: 600;
}

.remove-chip {
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.8);
  cursor: pointer;
  font-size: 1rem;
  line-height: 1;
  padding: 0;
  display: flex;
  align-items: center;
}
.remove-chip:hover { color: white; }

.multi-select {
  padding: 0.5rem 0.75rem;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 0.9rem;
  background: white;
  cursor: pointer;
}

/* Report sections */
.report-section { margin-bottom: 2.5rem; }

.stat-sub {
  font-size: 0.75rem;
  color: #6c757d;
  margin-top: 0.3rem;
}

.yoy-grid {
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  margin-top: -0.75rem;
  margin-bottom: 1.5rem;
}

.change-positive { color: #27ae60 !important; }
.change-negative { color: #e74c3c !important; }

/* Chart wrap overrides the global .chart-card canvas height */
.chart-wrap {
  position: relative;
  height: 300px;
}
.chart-wrap canvas {
  width: 100% !important;
  height: 100% !important;
}

/* Dept comparison table horizontal scroll */
.report-table-scroll {
  overflow-x: auto;
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 8px;
  margin-bottom: 1.5rem;
}

/* Dark mode */
:global(body.dark-mode) .stat-sub { color: #64748b; }
:global(body.dark-mode) .multi-select { background: #1a202c; color: #e2e8f0; border-color: #4a5568; }
:global(body.dark-mode) .report-table-scroll { background: #1e2a3a; border-color: #4a5568; }

@media (max-width: 768px) {
  .multi-select-container { min-width: 100%; }
  .net-grid { grid-template-columns: 1fr; }
}
</style>
