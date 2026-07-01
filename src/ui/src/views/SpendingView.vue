<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useCollapsibleSections, useHelpModal, useSortable, useUrlSync } from '../composables/composables'
import { aggregateByDay, chartColors, makeBarChart, makeDoughnutChart, makeLineChart } from '../utils/chartUtils'
import { getApiUrl } from '../utils/env'
import { formatCurrency, formatDate, formatNumber } from '../utils/format'
import { setHashParams } from '../utils/urlFilters'

const CURRENT_YEAR = new Date().getFullYear()

const spendingYears = Array.from({ length: CURRENT_YEAR - 2018 }, (_, i) => CURRENT_YEAR - i)

const records = ref([])
const loading = ref(false)
const error = ref('')
const totalRecords = ref(0)
const totalAmount = ref(0)

const year = ref(CURRENT_YEAR)
const page = ref(0)
const limit = ref(25)
const searchVendor = ref('')
const searchOrg = ref('')
const searchCabinet = ref('')
const searchObjectClass = ref('')
const searchCity = ref('')
const searchState = ref('')
const searchFund = ref('')
const fromDate = ref('')
const toDate = ref('')
const sortDir = ref('desc')
const sortField = ref('date')

const { sortField: clientSortField, sortDirection, sortBy, getSortIndicator, sortRecords } = useSortable()
const helpModal = useHelpModal()

const readUrlParams = useUrlSync(
  () => ({
    year: year.value !== CURRENT_YEAR ? String(year.value) : undefined,
    vendor: searchVendor.value || undefined,
    org: searchOrg.value || undefined,
    cabinet: searchCabinet.value || undefined,
    object_class: searchObjectClass.value || undefined,
    city: searchCity.value || undefined,
    state: searchState.value || undefined,
    fund: searchFund.value || undefined,
    from: fromDate.value || undefined,
    to: toDate.value || undefined,
    limit: limit.value !== 25 ? String(limit.value) : undefined,
  }),
  [year, searchVendor, searchOrg, searchCabinet, searchObjectClass, searchCity, searchState, searchFund, fromDate, toDate, limit]
)
const { showCharts, showTable } = useCollapsibleSections()

const spendingBarRef = ref(null)
const vendorBarRef = ref(null)
const classBarRef = ref(null)
const locationBarRef = ref(null)
const fundDoughnutRef = ref(null)
const monthlyTrendRef = ref(null)
let deptChart, vendorChart, classChart, locationChart, fundDoughnutChart, monthlyTrendChart

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
    map.set(key, (map.get(key) || 0) + (r.amount || 0))
  })
  return Array.from(map.entries())
    .map(([label, amount]) => ({ label, amount }))
    .filter(x => x.amount > 0)
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 10)
}

const topOrgs = computed(() => aggregateByField('department'))
const topVendors = computed(() => aggregateByField('vendor'))
const topObjectClasses = computed(() => aggregateByField('object_class'))
const topCities = computed(() => aggregateByField('city'))
const topFunds = computed(() => aggregateByField('fund'))
const dailySpending = computed(() => aggregateByDay(records.value, 'date', 'amount'))

const renderCharts = async () => {
  if (loading.value || records.value.length === 0) return
  await nextTick()

  if (deptChart) deptChart.destroy()
  if (vendorChart) vendorChart.destroy()
  if (classChart) classChart.destroy()
  if (locationChart) locationChart.destroy()
  if (fundDoughnutChart) fundDoughnutChart.destroy()
  if (monthlyTrendChart) monthlyTrendChart.destroy()

  deptChart     = makeBarChart(spendingBarRef.value,  topOrgs.value,          chartColors.primary, { valueLabel: 'Spending' })
  vendorChart   = makeBarChart(vendorBarRef.value,    topVendors.value,       chartColors.blue,    { valueLabel: 'Spending' })
  classChart    = makeBarChart(classBarRef.value,     topObjectClasses.value, chartColors.teal,    { valueLabel: 'Spending' })
  locationChart = makeBarChart(locationBarRef.value,  topCities.value,        chartColors.orange,  { valueLabel: 'Spending' })
  fundDoughnutChart = makeDoughnutChart(fundDoughnutRef.value, topFunds.value, { top: 5 })
  monthlyTrendChart = makeLineChart(monthlyTrendRef.value, dailySpending.value, chartColors.primary, { valueLabel: 'Spending' })
}

watch([records, loading], ([newRecords, isLoading]) => {
  if (!isLoading && newRecords?.length > 0) renderCharts()
})

const fetchSpending = async () => {
  loading.value = true
  error.value = ''

  const params = new URLSearchParams({
    year: String(year.value),
    page: String(page.value),
    limit: String(limit.value),
    sort: sortDir.value,
    sort_field: sortField.value,
  })
  if (searchVendor.value) params.set('vendor', searchVendor.value)
  if (searchOrg.value) params.set('org', searchOrg.value)
  if (searchCabinet.value) params.set('cabinet', searchCabinet.value)
  if (searchObjectClass.value) params.set('object_class', searchObjectClass.value)
  if (searchCity.value) params.set('city', searchCity.value)
  if (searchState.value) params.set('state', searchState.value)
  if (searchFund.value) params.set('fund', searchFund.value)
  if (fromDate.value) params.set('from', fromDate.value)
  if (toDate.value) params.set('to', toDate.value)

  try {
    const resp = await fetch(getApiUrl(`/api/mass-spending?${params}`))
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = await resp.json()
    if (data.error) throw new Error(data.error)
    records.value = data.data ?? []
    totalRecords.value = data.count ?? 0
    totalAmount.value = data.total_amount ?? 0
  } catch (err) {
    console.error(err)
    error.value = `Error fetching spending data: ${err.message}`
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { page.value = 0; fetchSpending() }
const handleYearChange = () => { page.value = 0; fetchSpending() }

const goToPage = (p) => { page.value = p - 1; fetchSpending() }
const firstPage = () => goToPage(1)
const previousPage = () => goToPage(currentPage.value - 1)
const nextPage = () => goToPage(currentPage.value + 1)
const lastPage = () => goToPage(totalPages.value)
const onPageSelect = (e) => goToPage(Number(e.target.value))
const onLimitChange = () => { page.value = 0; fetchSpending() }

const resetFilters = () => {
  searchVendor.value = ''
  searchOrg.value = ''
  searchCabinet.value = ''
  searchObjectClass.value = ''
  searchCity.value = ''
  searchState.value = ''
  searchFund.value = ''
  fromDate.value = ''
  toDate.value = ''
  year.value = CURRENT_YEAR
  limit.value = 25
  page.value = 0
  sortDir.value = 'desc'
  sortField.value = 'date'
  setHashParams({})
  fetchSpending()
}

const drillDown = (field, value) => {
  const map = {
    vendor: searchVendor,
    department: searchOrg,
    cabinet_secretariat: searchCabinet,
    object_class: searchObjectClass,
    city: searchCity,
    state: searchState,
    fund: searchFund,
  }
  if (!map[field] || !value) return
  map[field].value = value
  page.value = 0
  fetchSpending()
}

onMounted(() => {
  readUrlParams(p => {
    if (p.year) year.value = Number(p.year)
    if (p.vendor) searchVendor.value = p.vendor
    if (p.org) searchOrg.value = p.org
    if (p.cabinet) searchCabinet.value = p.cabinet
    if (p.object_class) searchObjectClass.value = p.object_class
    if (p.city) searchCity.value = p.city
    if (p.state) searchState.value = p.state
    if (p.fund) searchFund.value = p.fund
    if (p.from) fromDate.value = p.from
    if (p.to) toDate.value = p.to
    if (p.limit) limit.value = Number(p.limit)
  })
  fetchSpending()
})
</script>

<template>
  <div class="data-explorer">
    <div class="explorer-header">
      <h1>State Spending</h1>
      <p class="subtitle">Public checkbook data for the Commonwealth of Massachusetts</p>
      <button
        @click="helpModal.openHelpModal('State Spending', 'https://cthruspending.mass.gov/#!/year/2026/', 'Complete records of financial transactions and vendor payments made by the Commonwealth of Massachusetts, sourced from the CTHRU Spending platform.')"
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
        <div class="stat-label">Total Spending in Search</div>
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
        <label>Fiscal Year:</label>
        <select v-model.number="year" @change="handleYearChange">
          <option v-for="y in spendingYears" :key="y" :value="y">FY{{ y }}</option>
        </select>
      </div>

      <div class="filter-group">
        <label>Vendor:</label>
        <input type="text" v-model="searchVendor" @keyup.enter="handleSearch" placeholder="Vendor name...">
      </div>

      <div class="filter-group">
        <label>Department:</label>
        <input type="text" v-model="searchOrg" @keyup.enter="handleSearch" placeholder="Department name...">
      </div>

      <div class="filter-group">
        <label>Cabinet:</label>
        <input type="text" v-model="searchCabinet" @keyup.enter="handleSearch" placeholder="Cabinet...">
      </div>

      <div class="filter-group">
        <label>Class:</label>
        <input type="text" v-model="searchObjectClass" @keyup.enter="handleSearch" placeholder="Object class...">
      </div>

      <div class="filter-group">
        <label>City:</label>
        <input type="text" v-model="searchCity" @keyup.enter="handleSearch" placeholder="City...">
      </div>

      <div class="filter-group">
        <label>State:</label>
        <input type="text" v-model="searchState" @keyup.enter="handleSearch" placeholder="State (e.g. MA)...">
      </div>

      <div class="filter-group">
        <label>Fund:</label>
        <input type="text" v-model="searchFund" @keyup.enter="handleSearch" placeholder="Fund name...">
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
          <option value="date">Expense Date</option>
          <option value="create_date">Date Entered</option>
          <option value="amount">Amount</option>
          <option value="vendor">Vendor</option>
          <option value="department">Department</option>
          <option value="cabinet_secretariat">Cabinet</option>
          <option value="object_class">Object Class</option>
          <option value="fund">Fund</option>
          <option value="city">City</option>
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
          <h3>Spending by Department</h3>
          <canvas ref="spendingBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Spending by Vendor</h3>
          <canvas ref="vendorBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Spending by Class</h3>
          <canvas ref="classBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Spending by City</h3>
          <canvas ref="locationBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Spending Trend by Day (This Page)</h3>
          <canvas v-if="dailySpending.length >= 2" ref="monthlyTrendRef"></canvas>
          <p v-else class="chart-empty">
            Records on this page span fewer than 2 days — try a wider date range or increasing records per page.
          </p>
        </div>
        <div class="chart-card">
          <h3>Spending by Fund Distribution (This Page)</h3>
          <canvas ref="fundDoughnutRef"></canvas>
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
                <th @click="sortBy('vendor')" style="cursor: pointer;">Vendor {{ getSortIndicator('vendor') }}</th>
                <th @click="sortBy('department')" style="cursor: pointer;">Department {{ getSortIndicator('department') }}</th>
                <th @click="sortBy('object_class')" style="cursor: pointer;">Object Class {{ getSortIndicator('object_class') }}</th>
                <th @click="sortBy('cabinet_secretariat')" style="cursor: pointer;">Cabinet {{ getSortIndicator('cabinet_secretariat') }}</th>
                <th @click="sortBy('city')" style="cursor: pointer;">City {{ getSortIndicator('city') }}</th>
                <th @click="sortBy('state')" style="cursor: pointer;">State {{ getSortIndicator('state') }}</th>
                <th @click="sortBy('fund')" style="cursor: pointer;">Fund {{ getSortIndicator('fund') }}</th>
                <th @click="sortBy('amount')" style="cursor: pointer;">Amount {{ getSortIndicator('amount') }}</th>
                <th @click="sortBy('date')" style="cursor: pointer;">Expense Date {{ getSortIndicator('date') }}</th>
                <th @click="sortBy('create_date')" style="cursor: pointer;">Date Entered {{ getSortIndicator('create_date') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="record in sortedRecords" :key="record.payment_id">
                <td data-label="Vendor">
                  <span class="drilldown" @click="drillDown('vendor', record.vendor)">{{ record.vendor }}</span>
                </td>
                <td data-label="Department">
                  <span class="drilldown" @click="drillDown('department', record.department)">{{ record.department }}</span>
                </td>
                <td data-label="Object Class">
                  <span class="drilldown" @click="drillDown('object_class', record.object_class)">{{ record.object_class }}</span>
                </td>
                <td data-label="Cabinet">
                  <span class="drilldown" @click="drillDown('cabinet_secretariat', record.cabinet_secretariat)">{{ record.cabinet_secretariat }}</span>
                </td>
                <td data-label="City">
                  <span class="drilldown" @click="drillDown('city', record.city)">{{ record.city }}</span>
                </td>
                <td data-label="State">
                  <span class="drilldown" @click="drillDown('state', record.state)">{{ record.state }}</span>
                </td>
                <td data-label="Fund">
                  <span class="drilldown" @click="drillDown('fund', record.fund)">{{ record.fund }}</span>
                </td>
                <td data-label="Amount" class="total-cost">${{ formatCurrency(record.amount) }}</td>
                <td data-label="Expense Date">{{ formatDate(record.date) }}</td>
                <td data-label="Date Entered">{{ formatDate(record.create_date) }}</td>
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
            <button :disabled="currentPage === totalPages" @click="lastPage" class="page-btn" title="Last »">Last »</button>
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
