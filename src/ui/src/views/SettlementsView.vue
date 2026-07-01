<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useCollapsibleSections, useHelpModal, useSortable, useUrlSync } from '../composables/composables'
import { aggregateByDay, chartColors, makeBarChart, makeLineChart } from '../utils/chartUtils'
import { getApiUrl } from '../utils/env'
import { formatCurrency, formatDate, formatNumber } from '../utils/format'
import { setHashParams } from '../utils/urlFilters'

const CURRENT_YEAR = new Date().getFullYear()
const settlementYears = Array.from({ length: CURRENT_YEAR - 2013 }, (_, i) => String(2014 + i))

const records = ref([])
const loading = ref(false)
const error = ref('')
const totalRecords = ref(0)
const totalAmount = ref(0)

const year = ref(String(CURRENT_YEAR))
const page = ref(0)
const limit = ref(25)
const searchPayee = ref('')
const searchDept = ref('')
const filterQuarter = ref('')
const fromDate = ref('')
const toDate = ref('')
const sortDir = ref('desc')
const sortField = ref('payment_date')

const { sortField: clientSortField, sortBy, getSortIndicator, sortRecords } = useSortable()
const helpModal = useHelpModal()

const readUrlParams = useUrlSync(
  () => ({
    year: year.value !== String(CURRENT_YEAR) ? year.value : undefined,
    payee: searchPayee.value || undefined,
    dept: searchDept.value || undefined,
    quarter: filterQuarter.value || undefined,
    from: fromDate.value || undefined,
    to: toDate.value || undefined,
    limit: limit.value !== 25 ? String(limit.value) : undefined,
  }),
  [year, searchPayee, searchDept, filterQuarter, fromDate, toDate, limit]
)
const { showCharts, showTable } = useCollapsibleSections()

const payeeBarRef   = ref(null)
const deptBarRef    = ref(null)
const quarterBarRef = ref(null)
const trendRef      = ref(null)
let payeeChart, deptChart, quarterChart, trendChart

const totalPages  = computed(() => Math.ceil(totalRecords.value / limit.value))
const currentPage = computed(() => page.value + 1)
const startRecord = computed(() => totalRecords.value === 0 ? 0 : page.value * limit.value + 1)
const endRecord   = computed(() => Math.min(page.value * limit.value + records.value.length, totalRecords.value))
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
    map.set(key, (map.get(key) || 0) + (r.line_amount || 0))
  })
  return Array.from(map.entries())
    .map(([label, amount]) => ({ label, amount }))
    .filter(x => x.amount > 0)
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 10)
}

const QUARTER_ORDER = ['First', 'Second', 'Third', 'Fourth']
const topPayees     = computed(() => aggregateByField('payee_name'))
const topDepts      = computed(() => aggregateByField('dept_paid_on_behalf_of'))
const quarterData   = computed(() => {
  const map = new Map()
  records.value.forEach(r => {
    const q = r.quarter || 'Unknown'
    map.set(q, (map.get(q) || 0) + (r.line_amount || 0))
  })
  return QUARTER_ORDER
    .filter(q => map.has(q))
    .map(q => ({ label: q, amount: map.get(q) }))
})
const dailySettlements = computed(() => aggregateByDay(records.value, 'payment_date', 'line_amount'))

const renderCharts = async () => {
  if (loading.value || records.value.length === 0) return
  await nextTick()

  if (payeeChart)   payeeChart.destroy()
  if (deptChart)    deptChart.destroy()
  if (quarterChart) quarterChart.destroy()
  if (trendChart)   trendChart.destroy()

  payeeChart   = makeBarChart(payeeBarRef.value,   topPayees.value,   chartColors.primary, { valueLabel: 'Amount' })
  deptChart    = makeBarChart(deptBarRef.value,    topDepts.value,    chartColors.teal,    { valueLabel: 'Amount' })
  quarterChart = makeBarChart(quarterBarRef.value, quarterData.value, chartColors.orange,  { valueLabel: 'Amount' })
  trendChart   = makeLineChart(trendRef.value,     dailySettlements.value, chartColors.blue, { valueLabel: 'Amount' })
}

watch([records, loading], ([newRecords, isLoading]) => {
  if (!isLoading && newRecords?.length > 0) renderCharts()
})

const fetchSettlements = async () => {
  loading.value = true
  error.value = ''

  const params = new URLSearchParams({
    year:       year.value,
    page:       String(page.value),
    limit:      String(limit.value),
    sort:       sortDir.value,
    sort_field: sortField.value,
  })
  if (searchPayee.value)  params.set('payee',   searchPayee.value)
  if (searchDept.value)   params.set('dept',    searchDept.value)
  if (filterQuarter.value) params.set('quarter', filterQuarter.value)
  if (fromDate.value)     params.set('from',    fromDate.value)
  if (toDate.value)       params.set('to',      toDate.value)

  try {
    const resp = await fetch(getApiUrl(`/api/mass-settlements?${params}`))
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = await resp.json()
    if (data.error) throw new Error(data.error)
    records.value      = data.data ?? []
    totalRecords.value = data.count ?? 0
    totalAmount.value  = data.total_amount ?? 0
  } catch (err) {
    console.error(err)
    error.value = `Error fetching settlements data: ${err.message}`
  } finally {
    loading.value = false
  }
}

const handleSearch   = () => { page.value = 0; fetchSettlements() }
const handleYearChange = () => { page.value = 0; fetchSettlements() }
const goToPage       = (p) => { page.value = p - 1; fetchSettlements() }
const firstPage      = () => goToPage(1)
const previousPage   = () => goToPage(currentPage.value - 1)
const nextPage       = () => goToPage(currentPage.value + 1)
const lastPage       = () => goToPage(totalPages.value)
const onPageSelect   = (e) => goToPage(Number(e.target.value))
const onLimitChange  = () => { page.value = 0; fetchSettlements() }

const drillDown = (field, value) => {
  const map = {
    payee_name:          searchPayee,
    dept_paid_on_behalf_of: searchDept,
  }
  if (field === 'quarter') {
    filterQuarter.value = value
  } else if (map[field] && value) {
    map[field].value = value
  } else {
    return
  }
  page.value = 0
  fetchSettlements()
}

const resetFilters = () => {
  searchPayee.value   = ''
  searchDept.value    = ''
  filterQuarter.value = ''
  fromDate.value      = ''
  toDate.value        = ''
  year.value          = String(CURRENT_YEAR)
  limit.value         = 25
  page.value          = 0
  sortDir.value       = 'desc'
  sortField.value     = 'payment_date'
  setHashParams({})
  fetchSettlements()
}

onMounted(() => {
  readUrlParams(p => {
    if (p.year)    year.value          = p.year
    if (p.payee)   searchPayee.value   = p.payee
    if (p.dept)    searchDept.value    = p.dept
    if (p.quarter) filterQuarter.value = p.quarter
    if (p.from)    fromDate.value      = p.from
    if (p.to)      toDate.value        = p.to
    if (p.limit)   limit.value         = Number(p.limit)
  })
  fetchSettlements()
})
</script>

<template>
  <div class="data-explorer">
    <div class="explorer-header">
      <h1>Settlements &amp; Judgments</h1>
      <p class="subtitle">Commonwealth of Massachusetts settlement and judgment payments</p>
      <button
        @click="helpModal.openHelpModal('Settlements & Judgments', 'https://cthru.data.socrata.com/stories/s/tq8s-u87j', 'Records of settlement and judgment payments made by the Commonwealth of Massachusetts on behalf of state agencies, sourced from the CTHRU platform.')"
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
        <div class="stat-label">Total Amount in Search</div>
        <div class="stat-value">${{ formatCurrency(totalAmount) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Average Amount in Search</div>
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
        <label>Fiscal Year (BFY):</label>
        <select v-model="year" @change="handleYearChange">
          <option v-for="y in settlementYears" :key="y" :value="y">FY{{ y }}</option>
        </select>
      </div>

      <div class="filter-group">
        <label>Payee:</label>
        <input type="text" v-model="searchPayee" @keyup.enter="handleSearch" placeholder="Payee name...">
      </div>

      <div class="filter-group">
        <label>Department:</label>
        <input type="text" v-model="searchDept" @keyup.enter="handleSearch" placeholder="Department name...">
      </div>

      <div class="filter-group">
        <label>Quarter:</label>
        <select v-model="filterQuarter" @change="handleSearch">
          <option value="">All Quarters</option>
          <option value="First">First</option>
          <option value="Second">Second</option>
          <option value="Third">Third</option>
          <option value="Fourth">Fourth</option>
        </select>
      </div>

      <div class="filter-group">
        <label>From Date:</label>
        <input type="date" v-model="fromDate" @change="handleSearch">
      </div>

      <div class="filter-group">
        <label>To Date:</label>
        <input type="date" v-model="toDate" @change="handleSearch">
      </div>

      <div class="filter-group">
        <label>Sort by:</label>
        <select v-model="sortField" @change="handleSearch">
          <option value="payment_date">Payment Date</option>
          <option value="line_amount">Amount</option>
          <option value="payee_name">Payee</option>
          <option value="dept_paid_on_behalf_of">Department</option>
          <option value="quarter">Quarter</option>
          <option value="bfy">Fiscal Year</option>
          <option value="paid_on_behalf_of">Agency Code</option>
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
        <h2>Charts &amp; Analysis</h2>
      </div>
      <div v-show="showCharts" class="charts-grid">
        <div class="chart-card">
          <h3>Top Payees by Amount</h3>
          <canvas ref="payeeBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Top Departments by Amount</h3>
          <canvas ref="deptBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Settlements by Quarter (This Page)</h3>
          <canvas ref="quarterBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Payment Trend by Day (This Page)</h3>
          <canvas v-if="dailySettlements.length >= 2" ref="trendRef"></canvas>
          <p v-else class="chart-empty">
            Records on this page span fewer than 2 days — try a wider date range or increasing records per page.
          </p>
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
                <th @click="sortBy('payee_name')" style="cursor:pointer">Payee {{ getSortIndicator('payee_name') }}</th>
                <th @click="sortBy('dept_paid_on_behalf_of')" style="cursor:pointer">Department {{ getSortIndicator('dept_paid_on_behalf_of') }}</th>
                <th @click="sortBy('paid_on_behalf_of')" style="cursor:pointer">Agency {{ getSortIndicator('paid_on_behalf_of') }}</th>
                <th @click="sortBy('line_amount')" style="cursor:pointer">Amount {{ getSortIndicator('line_amount') }}</th>
                <th @click="sortBy('payment_date')" style="cursor:pointer">Payment Date {{ getSortIndicator('payment_date') }}</th>
                <th @click="sortBy('bfy')" style="cursor:pointer">BFY {{ getSortIndicator('bfy') }}</th>
                <th @click="sortBy('quarter')" style="cursor:pointer">Quarter {{ getSortIndicator('quarter') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(record, i) in sortedRecords" :key="i">
                <td data-label="Payee">
                  <span class="drilldown" @click="drillDown('payee_name', record.payee_name)">{{ record.payee_name }}</span>
                </td>
                <td data-label="Department">
                  <span class="drilldown" @click="drillDown('dept_paid_on_behalf_of', record.dept_paid_on_behalf_of)">{{ record.dept_paid_on_behalf_of }}</span>
                </td>
                <td data-label="Agency">{{ record.paid_on_behalf_of }}</td>
                <td data-label="Amount" class="total-cost">${{ formatCurrency(record.line_amount) }}</td>
                <td data-label="Payment Date">{{ formatDate(record.payment_date) }}</td>
                <td data-label="BFY">{{ record.bfy }}</td>
                <td data-label="Quarter">
                  <span class="drilldown" @click="drillDown('quarter', record.quarter)">{{ record.quarter }}</span>
                </td>
              </tr>
            </tbody>
          </table>

          <div class="pagination" v-if="totalRecords > 0">
            <button :disabled="currentPage === 1" @click="firstPage" class="page-btn" title="First Page">« First</button>
            <button :disabled="currentPage === 1" @click="previousPage" class="page-btn" title="Previous Page">‹ Previous</button>

            <span class="pagination-info">
              Showing {{ formatNumber(startRecord) }}–{{ formatNumber(endRecord) }} of {{ formatNumber(totalRecords) }} records
            </span>

            <select :value="currentPage" @change="onPageSelect" class="page-select" title="Select Page">
              <option v-for="p in pageOptions" :key="p" :value="p">Page {{ p }} of {{ totalPages }}</option>
            </select>

            <button :disabled="currentPage === totalPages" @click="nextPage" class="page-btn" title="Next Page">Next ›</button>
            <button :disabled="currentPage === totalPages" @click="lastPage" class="page-btn" title="Last">Last »</button>
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
