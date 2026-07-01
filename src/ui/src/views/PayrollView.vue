<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useCollapsibleSections, useHelpModal, useSortable, useUrlSync } from '../composables/composables'
import { chartColors, makeBarChart, makeDoughnutChart, makeStackedBarChart } from '../utils/chartUtils'
import { getApiUrl } from '../utils/env'
import { formatCurrency, formatDate, formatNumber } from '../utils/format'
import { setHashParams } from '../utils/urlFilters'

const CURRENT_YEAR = new Date().getFullYear()
const payrollYears = Array.from({ length: CURRENT_YEAR - 2010 }, (_, i) => CURRENT_YEAR - i)

const records = ref([])
const loading = ref(false)
const error = ref('')
const totalRecords = ref(0)
const totalAmount = ref(0)

const year = ref(CURRENT_YEAR)
const page = ref(0)
const limit = ref(25)
const searchLast = ref('')
const searchFirst = ref('')
const searchDept = ref('')
const searchTitle = ref('')
const searchType = ref('')
const searchBargaining = ref('')
const sortDir = ref('desc')
const sortField = ref('service_end_date')

const { sortField: clientSortField, sortDirection, sortBy, getSortIndicator, sortRecords } = useSortable()
const helpModal = useHelpModal()

const readUrlParams = useUrlSync(
  () => ({
    year: year.value !== CURRENT_YEAR ? String(year.value) : undefined,
    last: searchLast.value || undefined,
    first: searchFirst.value || undefined,
    dept: searchDept.value || undefined,
    title: searchTitle.value || undefined,
    type: searchType.value || undefined,
    bargaining: searchBargaining.value || undefined,
    limit: limit.value !== 25 ? String(limit.value) : undefined,
  }),
  [year, searchLast, searchFirst, searchDept, searchTitle, searchType, searchBargaining, limit]
)
const { showCharts, showTable } = useCollapsibleSections()

const deptBarRef = ref(null)
const titleBarRef = ref(null)
const typeBarRef = ref(null)
const bargainingBarRef = ref(null)
const typeDoughnutRef = ref(null)
const baseOvertimeRef = ref(null)
let deptChart, titleChart, typeChart, bargainingChart, typeDoughnutChart, baseOvertimeChart

const totalPages = computed(() => Math.ceil(totalRecords.value / limit.value))
const currentPage = computed(() => page.value + 1)
const startRecord = computed(() => totalRecords.value === 0 ? 0 : page.value * limit.value + 1)
const endRecord = computed(() => Math.min(page.value * limit.value + records.value.length, totalRecords.value))
const averageAmount = computed(() => totalRecords.value > 0 ? totalAmount.value / totalRecords.value : 0)

const pageOptions = computed(() => {
  const pages = []
  for (let i = 1; i <= totalPages.value; i++) pages.push(i)
  return pages
})

const sortedRecords = computed(() => {
  if (!clientSortField.value) return records.value
  return sortRecords(records.value, clientSortField.value)
})

const aggregateByField = (field) => {
  const map = new Map()
  records.value.forEach(r => {
    const key = r[field] || 'Unknown'
    map.set(key, (map.get(key) || 0) + (r.pay_total_actual || 0))
  })
  return Array.from(map.entries())
    .map(([label, amount]) => ({ label, amount }))
    .filter(x => x.amount > 0)
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 10)
}

const topDepts = computed(() => aggregateByField('department_division'))
const topTitles = computed(() => aggregateByField('position_title'))
const topTypes = computed(() => aggregateByField('position_type'))
const topBargaining = computed(() => aggregateByField('bargaining_group_title'))

const baseOvertimeByDept = computed(() => {
  const map = new Map()
  records.value.forEach(r => {
    const key = r.department_division || 'Unknown'
    const entry = map.get(key) || { base: 0, overtime: 0 }
    entry.base += r.pay_base_actual || 0
    entry.overtime += r.pay_overtime_actual || 0
    map.set(key, entry)
  })
  return Array.from(map.entries())
    .map(([label, v]) => ({ label, ...v, total: v.base + v.overtime }))
    .filter(d => d.total > 0)
    .sort((a, b) => b.total - a.total)
    .slice(0, 8)
})

const renderCharts = async () => {
  if (loading.value || records.value.length === 0) return
  await nextTick()
  if (deptChart) deptChart.destroy()
  if (titleChart) titleChart.destroy()
  if (typeChart) typeChart.destroy()
  if (bargainingChart) bargainingChart.destroy()
  if (typeDoughnutChart) typeDoughnutChart.destroy()
  if (baseOvertimeChart) baseOvertimeChart.destroy()
  deptChart       = makeBarChart(deptBarRef.value,       topDepts.value,      chartColors.primary, { valueLabel: 'Total Pay' })
  titleChart      = makeBarChart(titleBarRef.value,      topTitles.value,     chartColors.blue,    { valueLabel: 'Total Pay' })
  typeChart       = makeBarChart(typeBarRef.value,        topTypes.value,      chartColors.teal,    { valueLabel: 'Total Pay' })
  bargainingChart = makeBarChart(bargainingBarRef.value,  topBargaining.value, chartColors.purple,  { valueLabel: 'Total Pay' })
  typeDoughnutChart = makeDoughnutChart(typeDoughnutRef.value, topTypes.value, { top: 6 })
  baseOvertimeChart = makeStackedBarChart(
    baseOvertimeRef.value,
    baseOvertimeByDept.value.map(d => d.label),
    [
      { label: 'Base Pay', data: baseOvertimeByDept.value.map(d => d.base) },
      { label: 'Overtime', data: baseOvertimeByDept.value.map(d => d.overtime) },
    ],
    [chartColors.primary, chartColors.red]
  )
}

watch([records, loading], ([r, l]) => { if (!l && r?.length > 0) renderCharts() })

const fetchPayroll = async () => {
  loading.value = true
  error.value = ''
  const params = new URLSearchParams({
    year: String(year.value),
    page: String(page.value),
    limit: String(limit.value),
    sort: sortDir.value,
    sort_field: sortField.value,
  })
  if (searchLast.value) params.set('last', searchLast.value)
  if (searchFirst.value) params.set('first', searchFirst.value)
  if (searchDept.value) params.set('dept', searchDept.value)
  if (searchTitle.value) params.set('title', searchTitle.value)
  if (searchType.value) params.set('type', searchType.value)
  if (searchBargaining.value) params.set('bargaining', searchBargaining.value)
  try {
    const resp = await fetch(getApiUrl(`/api/mass-payroll?${params}`))
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = await resp.json()
    if (data.error) throw new Error(data.error)
    records.value = data.data ?? []
    totalRecords.value = data.count ?? 0
    totalAmount.value = data.total_amount ?? 0
  } catch (err) {
    console.error(err)
    error.value = `Error fetching payroll data: ${err.message}`
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { page.value = 0; fetchPayroll() }
const handleYearChange = () => { page.value = 0; fetchPayroll() }

const goToPage = (p) => { page.value = p - 1; fetchPayroll() }
const firstPage = () => goToPage(1)
const previousPage = () => goToPage(currentPage.value - 1)
const nextPage = () => goToPage(currentPage.value + 1)
const lastPage = () => goToPage(totalPages.value)
const onPageSelect = (e) => goToPage(Number(e.target.value))
const onLimitChange = () => { page.value = 0; fetchPayroll() }

const resetFilters = () => {
  searchLast.value = ''
  searchFirst.value = ''
  searchDept.value = ''
  searchTitle.value = ''
  searchType.value = ''
  searchBargaining.value = ''
  year.value = CURRENT_YEAR
  limit.value = 25
  page.value = 0
  sortDir.value = 'desc'
  sortField.value = 'service_end_date'
  setHashParams({})
  fetchPayroll()
}

const drillDown = (field, value) => {
  const map = {
    name_last:             searchLast,
    department_division:   searchDept,
    position_title:        searchTitle,
    position_type:         searchType,
    bargaining_group_title: searchBargaining,
  }
  if (!map[field] || !value) return
  map[field].value = value
  page.value = 0
  fetchPayroll()
}

onMounted(() => {
  readUrlParams(p => {
    if (p.year) year.value = Number(p.year)
    if (p.last) searchLast.value = p.last
    if (p.first) searchFirst.value = p.first
    if (p.dept) searchDept.value = p.dept
    if (p.title) searchTitle.value = p.title
    if (p.type) searchType.value = p.type
    if (p.bargaining) searchBargaining.value = p.bargaining
    if (p.limit) limit.value = Number(p.limit)
  })
  fetchPayroll()
})
</script>

<template>
  <div class="data-explorer">
    <div class="explorer-header">
      <h1>State Payroll</h1>
      <p class="subtitle">Employee compensation records for the Commonwealth of Massachusetts</p>
      <button
        @click="helpModal.openHelpModal('State Payroll', 'https://cthrupayroll.mass.gov/#!/year/2026/', 'Employee payroll records for Commonwealth of Massachusetts employees, sourced from the CTHRU platform.')"
        class="help-btn"
      >?</button>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">Total Records in Search</div>
        <div class="stat-value">{{ formatNumber(totalRecords) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Records on This Page</div>
        <div class="stat-value">{{ formatNumber(records.length) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Total Pay in Search</div>
        <div class="stat-value">${{ formatCurrency(totalAmount) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Average Pay per Record</div>
        <div class="stat-value">${{ formatCurrency(averageAmount) }}</div>
      </div>
    </div>

    <div class="filters">
      <div class="filter-group">
        <label>Records per page:</label>
        <select v-model.number="limit" @change="onLimitChange">
          <option :value="10">10</option>
          <option :value="25">25</option>
          <option :value="50">50</option>
          <option :value="100">100</option>
          <option :value="250">250</option>
          <option :value="500">500</option>
          <option :value="750">750</option>
          <option :value="1000">1000</option>
        </select>
      </div>
      <div class="filter-group">
        <label>Year:</label>
        <select v-model.number="year" @change="handleYearChange">
          <option v-for="y in payrollYears" :key="y" :value="y">{{ y }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>Last Name:</label>
        <input type="text" v-model="searchLast" @keyup.enter="handleSearch" placeholder="Last name...">
      </div>
      <div class="filter-group">
        <label>First Name:</label>
        <input type="text" v-model="searchFirst" @keyup.enter="handleSearch" placeholder="First name...">
      </div>
      <div class="filter-group">
        <label>Department / Division:</label>
        <input type="text" v-model="searchDept" @keyup.enter="handleSearch" placeholder="Department...">
      </div>
      <div class="filter-group">
        <label>Position Title:</label>
        <input type="text" v-model="searchTitle" @keyup.enter="handleSearch" placeholder="Title...">
      </div>
      <div class="filter-group">
        <label>Position Type:</label>
        <input type="text" v-model="searchType" @keyup.enter="handleSearch" placeholder="Full Time, Part Time...">
      </div>
      <div class="filter-group">
        <label>Bargaining Group:</label>
        <input type="text" v-model="searchBargaining" @keyup.enter="handleSearch" placeholder="Bargaining group...">
      </div>
      <div class="filter-group">
        <label>Sort by:</label>
        <select v-model="sortField" @change="handleSearch">
          <option value="service_end_date">Service End Date</option>
          <option value="year">Year</option>
          <option value="pay_total_actual">Total Pay</option>
          <option value="pay_base_actual">Base Pay</option>
          <option value="pay_overtime_actual">Overtime Pay</option>
          <option value="annual_rate">Annual Rate</option>
          <option value="name_last">Last Name</option>
          <option value="name_first">First Name</option>
          <option value="department_division">Department</option>
          <option value="position_title">Title</option>
          <option value="position_type">Position Type</option>
          <option value="bargaining_group_title">Bargaining Group</option>
        </select>
      </div>

      <div class="filter-group">
        <label>Direction:</label>
        <select v-model="sortDir" @change="handleSearch">
          <option value="desc">↓ Descending</option>
          <option value="asc">↑ Ascending</option>
        </select>
      </div>

      <div class="button-group">
        <button @click="handleSearch" class="search-btn">Search</button>
        <button @click="resetFilters" class="reset-btn">Reset</button>
      </div>
    </div>

    <div v-if="loading" class="loading"><div class="spinner"></div></div>
    <div v-else-if="error" class="error"><p>{{ error }}</p></div>

    <div v-else-if="records.length > 0" class="content-container">
      <div class="section-header" @click="showCharts = !showCharts">
        <span class="chevron">{{ showCharts ? '▼' : '▶' }}</span>
        <h2>Charts & Analysis</h2>
      </div>
      <div v-show="showCharts" class="charts-grid">
        <div class="chart-card">
          <h3>Top Departments by Total Pay (This Page)</h3>
          <canvas ref="deptBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Top Position Titles by Total Pay (This Page)</h3>
          <canvas ref="titleBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Top Position Types by Total Pay (This Page)</h3>
          <canvas ref="typeBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Top Bargaining Groups by Total Pay (This Page)</h3>
          <canvas ref="bargainingBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Position Type Distribution (This Page)</h3>
          <canvas ref="typeDoughnutRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Base vs Overtime Pay by Department (This Page)</h3>
          <canvas ref="baseOvertimeRef"></canvas>
        </div>
      </div>

      <div class="section-header" @click="showTable = !showTable">
        <span class="chevron">{{ showTable ? '▼' : '▶' }}</span>
        <h2>Records Table</h2>
      </div>
      <div v-show="showTable">
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th @click="sortBy('name_last')" style="cursor: pointer;">Last {{ getSortIndicator('name_last') }}</th>
                <th @click="sortBy('name_first')" style="cursor: pointer;">First {{ getSortIndicator('name_first') }}</th>
                <th @click="sortBy('department_division')" style="cursor: pointer;">Department / Division {{ getSortIndicator('department_division') }}</th>
                <th @click="sortBy('position_title')" style="cursor: pointer;">Title {{ getSortIndicator('position_title') }}</th>
                <th @click="sortBy('position_type')" style="cursor: pointer;">Type {{ getSortIndicator('position_type') }}</th>
                <th @click="sortBy('pay_total_actual')" style="cursor: pointer;">Total Pay {{ getSortIndicator('pay_total_actual') }}</th>
                <th @click="sortBy('pay_base_actual')" style="cursor: pointer;">Base {{ getSortIndicator('pay_base_actual') }}</th>
                <th @click="sortBy('pay_overtime_actual')" style="cursor: pointer;">Overtime {{ getSortIndicator('pay_overtime_actual') }}</th>
                <th @click="sortBy('annual_rate')" style="cursor: pointer;">Annual Rate {{ getSortIndicator('annual_rate') }}</th>
                <th @click="sortBy('service_end_date')" style="cursor: pointer;">Service End {{ getSortIndicator('service_end_date') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="record in sortedRecords" :key="record.id">
                <td data-label="Last">
                  <span class="drilldown" @click="drillDown('name_last', record.name_last)">{{ record.name_last }}</span>
                </td>
                <td data-label="First">{{ record.name_first }}</td>
                <td data-label="Department / Division">
                  <span class="drilldown" @click="drillDown('department_division', record.department_division)">{{ record.department_division }}</span>
                </td>
                <td data-label="Title">
                  <span class="drilldown" @click="drillDown('position_title', record.position_title)">{{ record.position_title }}</span>
                </td>
                <td data-label="Type">
                  <span class="drilldown" @click="drillDown('position_type', record.position_type)">{{ record.position_type }}</span>
                </td>
                <td data-label="Total Pay" class="total-cost">${{ formatCurrency(record.pay_total_actual) }}</td>
                <td data-label="Base">${{ formatCurrency(record.pay_base_actual) }}</td>
                <td data-label="Overtime">${{ formatCurrency(record.pay_overtime_actual) }}</td>
                <td data-label="Annual Rate">${{ formatCurrency(record.annual_rate) }}</td>
                <td data-label="Service End">{{ formatDate(record.service_end_date) }}</td>
              </tr>
            </tbody>
          </table>

          <div class="pagination" v-if="totalRecords > 0">
            <button :disabled="currentPage === 1" @click="firstPage" class="page-btn">« First</button>
            <button :disabled="currentPage === 1" @click="previousPage" class="page-btn">‹ Previous</button>
            <span class="pagination-info">
              Showing {{ formatNumber(startRecord) }}–{{ formatNumber(endRecord) }} of {{ formatNumber(totalRecords) }} records
            </span>
            <select :value="currentPage" @change="onPageSelect" class="page-select">
              <option v-for="p in pageOptions" :key="p" :value="p">Page {{ p }} of {{ totalPages }}</option>
            </select>
            <button :disabled="currentPage === totalPages" @click="nextPage" class="page-btn">Next ›</button>
            <button :disabled="currentPage === totalPages" @click="lastPage" class="page-btn">Last »</button>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <p>No records found matching your filters.</p>
    </div>

    <div v-if="helpModal.showModal.value" class="modal-overlay" @click="helpModal.closeModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>{{ helpModal.modalContent.value.title }}</h2>
          <button @click="helpModal.closeModal" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <p>{{ helpModal.modalContent.value.description }}</p>
          <a :href="helpModal.modalContent.value.url" target="_blank" class="dataset-link">View Dataset →</a>
        </div>
      </div>
    </div>
  </div>
</template>
