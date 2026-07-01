<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useCollapsibleSections, useHelpModal, useSortable, useUrlSync } from '../composables/composables'
import { aggregateByFiscalPeriod, chartColors, makeBarChart, makeDoughnutChart, makeLineChart } from '../utils/chartUtils'
import { getApiUrl } from '../utils/env'
import { formatCurrency, formatNumber } from '../utils/format'
import { setHashParams } from '../utils/urlFilters'

const CURRENT_YEAR = new Date().getFullYear()
const revenueYears = Array.from({ length: CURRENT_YEAR - 2010 }, (_, i) => CURRENT_YEAR - i)

const records = ref([])
const loading = ref(false)
const error = ref('')
const totalRecords = ref(0)
const totalAmount = ref(0)

const year = ref(CURRENT_YEAR)
const page = ref(0)
const limit = ref(25)
const searchDept = ref('')
const searchCabinet = ref('')
const searchCategory = ref('')
const searchClass = ref('')
const searchFund = ref('')
const sortDir = ref('desc')
const sortField = ref('fiscal_year')

const { sortField: clientSortField, sortDirection, sortBy, getSortIndicator, sortRecords } = useSortable()
const helpModal = useHelpModal()

const readUrlParams = useUrlSync(
  () => ({
    year: year.value !== CURRENT_YEAR ? String(year.value) : undefined,
    dept: searchDept.value || undefined,
    cabinet: searchCabinet.value || undefined,
    category: searchCategory.value || undefined,
    class: searchClass.value || undefined,
    fund: searchFund.value || undefined,
    limit: limit.value !== 25 ? String(limit.value) : undefined,
  }),
  [year, searchDept, searchCabinet, searchCategory, searchClass, searchFund, limit]
)
const { showCharts, showTable } = useCollapsibleSections()

const deptBarRef = ref(null)
const categoryBarRef = ref(null)
const cabinetBarRef = ref(null)
const fundBarRef = ref(null)
const categoryDoughnutRef = ref(null)
const monthlyTrendRef = ref(null)
let deptChart, categoryChart, cabinetChart, fundChart, categoryDoughnutChart, monthlyTrendChart

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
    map.set(key, (map.get(key) || 0) + (r.revenue_collected || 0))
  })
  return Array.from(map.entries())
    .map(([label, amount]) => ({ label, amount }))
    .filter(x => x.amount > 0)
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 10)
}

const topDepts = computed(() => aggregateByField('department_name'))
const topCategories = computed(() => aggregateByField('revenue_category_name'))
const topCabinets = computed(() => aggregateByField('cabinet_name'))
const topFunds = computed(() => aggregateByField('fund_name'))
const monthlyRevenue = computed(() => aggregateByFiscalPeriod(records.value, 'fiscal_period_month', 'revenue_collected'))

const renderCharts = async () => {
  if (loading.value || records.value.length === 0) return
  await nextTick()
  if (deptChart) deptChart.destroy()
  if (categoryChart) categoryChart.destroy()
  if (cabinetChart) cabinetChart.destroy()
  if (fundChart) fundChart.destroy()
  if (categoryDoughnutChart) categoryDoughnutChart.destroy()
  if (monthlyTrendChart) monthlyTrendChart.destroy()
  deptChart          = makeBarChart(deptBarRef.value,     topDepts.value,      chartColors.primary, { valueLabel: 'Revenue' })
  categoryChart       = makeBarChart(categoryBarRef.value, topCategories.value, chartColors.green,   { valueLabel: 'Revenue' })
  cabinetChart        = makeBarChart(cabinetBarRef.value,  topCabinets.value,   chartColors.teal,    { valueLabel: 'Revenue' })
  fundChart           = makeBarChart(fundBarRef.value,     topFunds.value,      chartColors.orange,  { valueLabel: 'Revenue' })
  categoryDoughnutChart = makeDoughnutChart(categoryDoughnutRef.value, topCategories.value, { top: 5 })
  monthlyTrendChart     = makeLineChart(monthlyTrendRef.value, monthlyRevenue.value, chartColors.primary, { valueLabel: 'Revenue' })
}

watch([records, loading], ([r, l]) => { if (!l && r?.length > 0) renderCharts() })

const fetchRevenue = async () => {
  loading.value = true
  error.value = ''
  const params = new URLSearchParams({
    year: String(year.value),
    page: String(page.value),
    limit: String(limit.value),
    sort: sortDir.value,
    sort_field: sortField.value,
  })
  if (searchDept.value) params.set('dept', searchDept.value)
  if (searchCabinet.value) params.set('cabinet', searchCabinet.value)
  if (searchCategory.value) params.set('category', searchCategory.value)
  if (searchClass.value) params.set('class', searchClass.value)
  if (searchFund.value) params.set('fund', searchFund.value)
  try {
    const resp = await fetch(getApiUrl(`/api/mass-revenue?${params}`))
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = await resp.json()
    if (data.error) throw new Error(data.error)
    records.value = data.data ?? []
    totalRecords.value = data.count ?? 0
    totalAmount.value = data.total_amount ?? 0
  } catch (err) {
    console.error(err)
    error.value = `Error fetching revenue data: ${err.message}`
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { page.value = 0; fetchRevenue() }
const handleYearChange = () => { page.value = 0; fetchRevenue() }

const goToPage = (p) => { page.value = p - 1; fetchRevenue() }
const firstPage = () => goToPage(1)
const previousPage = () => goToPage(currentPage.value - 1)
const nextPage = () => goToPage(currentPage.value + 1)
const lastPage = () => goToPage(totalPages.value)
const onPageSelect = (e) => goToPage(Number(e.target.value))
const onLimitChange = () => { page.value = 0; fetchRevenue() }

const resetFilters = () => {
  searchDept.value = ''
  searchCabinet.value = ''
  searchCategory.value = ''
  searchClass.value = ''
  searchFund.value = ''
  year.value = CURRENT_YEAR
  limit.value = 25
  page.value = 0
  sortDir.value = 'desc'
  sortField.value = 'fiscal_year'
  setHashParams({})
  fetchRevenue()
}

const drillDown = (field, value) => {
  const map = {
    department_name:      searchDept,
    cabinet_name:         searchCabinet,
    revenue_category_name: searchCategory,
    revenue_class_name:   searchClass,
    fund_name:            searchFund,
  }
  if (!map[field] || !value) return
  map[field].value = value
  page.value = 0
  fetchRevenue()
}

onMounted(() => {
  readUrlParams(p => {
    if (p.year) year.value = Number(p.year)
    if (p.dept) searchDept.value = p.dept
    if (p.cabinet) searchCabinet.value = p.cabinet
    if (p.category) searchCategory.value = p.category
    if (p.class) searchClass.value = p.class
    if (p.fund) searchFund.value = p.fund
    if (p.limit) limit.value = Number(p.limit)
  })
  fetchRevenue()
})
</script>

<template>
  <div class="data-explorer">
    <div class="explorer-header">
      <h1>State Revenue</h1>
      <p class="subtitle">Revenue transactions for the Commonwealth of Massachusetts</p>
      <button
        @click="helpModal.openHelpModal('State Revenue', 'https://cthrurevenue.mass.gov/#!/year/2026/', 'Revenue transactions collected by the Commonwealth of Massachusetts, sourced from the CTHRU platform.')"
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
        <div class="stat-label">Total Revenue in Search</div>
        <div class="stat-value">${{ formatCurrency(totalAmount) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Average Revenue per Record</div>
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
          <option v-for="y in revenueYears" :key="y" :value="y">FY{{ y }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>Department:</label>
        <input type="text" v-model="searchDept" @keyup.enter="handleSearch" placeholder="Department name...">
      </div>
      <div class="filter-group">
        <label>Cabinet:</label>
        <input type="text" v-model="searchCabinet" @keyup.enter="handleSearch" placeholder="Cabinet name...">
      </div>
      <div class="filter-group">
        <label>Revenue Category:</label>
        <input type="text" v-model="searchCategory" @keyup.enter="handleSearch" placeholder="Category...">
      </div>
      <div class="filter-group">
        <label>Revenue Class:</label>
        <input type="text" v-model="searchClass" @keyup.enter="handleSearch" placeholder="Revenue class...">
      </div>
      <div class="filter-group">
        <label>Fund:</label>
        <input type="text" v-model="searchFund" @keyup.enter="handleSearch" placeholder="Fund name...">
      </div>
      <div class="filter-group">
        <label>Sort by:</label>
        <select v-model="sortField" @change="handleSearch">
          <option value="fiscal_year">Fiscal Year</option>
          <option value="fiscal_period">Fiscal Period</option>
          <option value="revenue_collected">Amount Collected</option>
          <option value="department_name">Department</option>
          <option value="cabinet_name">Cabinet</option>
          <option value="revenue_category_name">Category</option>
          <option value="revenue_class_name">Revenue Class</option>
          <option value="fund_name">Fund</option>
          <option value="revenue_source_name">Revenue Source</option>
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
          <h3>Top Departments by Revenue (This Page)</h3>
          <canvas ref="deptBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Top Revenue Categories (This Page)</h3>
          <canvas ref="categoryBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Top Cabinets by Revenue (This Page)</h3>
          <canvas ref="cabinetBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Top Funds by Revenue (This Page)</h3>
          <canvas ref="fundBarRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Revenue by Fiscal Period (This Page)</h3>
          <canvas ref="monthlyTrendRef"></canvas>
        </div>
        <div class="chart-card">
          <h3>Revenue Category Distribution (This Page)</h3>
          <canvas ref="categoryDoughnutRef"></canvas>
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
                <th @click="sortBy('department_name')" style="cursor: pointer;">Department {{ getSortIndicator('department_name') }}</th>
                <th @click="sortBy('cabinet_name')" style="cursor: pointer;">Cabinet {{ getSortIndicator('cabinet_name') }}</th>
                <th @click="sortBy('revenue_category_name')" style="cursor: pointer;">Category {{ getSortIndicator('revenue_category_name') }}</th>
                <th @click="sortBy('revenue_class_name')" style="cursor: pointer;">Class {{ getSortIndicator('revenue_class_name') }}</th>
                <th @click="sortBy('revenue_source_name')" style="cursor: pointer;">Source {{ getSortIndicator('revenue_source_name') }}</th>
                <th @click="sortBy('fund_name')" style="cursor: pointer;">Fund {{ getSortIndicator('fund_name') }}</th>
                <th @click="sortBy('revenue_collected')" style="cursor: pointer;">Collected {{ getSortIndicator('revenue_collected') }}</th>
                <th @click="sortBy('fiscal_period_month')" style="cursor: pointer;">Period {{ getSortIndicator('fiscal_period_month') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="record in sortedRecords" :key="record.trans_no">
                <td data-label="Department">
                  <span class="drilldown" @click="drillDown('department_name', record.department_name)">{{ record.department_name }}</span>
                </td>
                <td data-label="Cabinet">
                  <span class="drilldown" @click="drillDown('cabinet_name', record.cabinet_name)">{{ record.cabinet_name }}</span>
                </td>
                <td data-label="Category">
                  <span class="drilldown" @click="drillDown('revenue_category_name', record.revenue_category_name)">{{ record.revenue_category_name }}</span>
                </td>
                <td data-label="Class">
                  <span class="drilldown" @click="drillDown('revenue_class_name', record.revenue_class_name)">{{ record.revenue_class_name }}</span>
                </td>
                <td data-label="Source">
                  <span class="drilldown" @click="drillDown('revenue_source_name', record.revenue_source_name)">{{ record.revenue_source_name }}</span>
                </td>
                <td data-label="Fund">
                  <span class="drilldown" @click="drillDown('fund_name', record.fund_name)">{{ record.fund_name }}</span>
                </td>
                <td data-label="Collected" class="total-cost">${{ formatCurrency(record.revenue_collected) }}</td>
                <td data-label="Period">{{ record.fiscal_period_month }}</td>
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
