import Chart from 'chart.js/auto'
import { formatCurrency, formatNumber } from './format'

const LIGHT_TEXT = '#4a5568'
const DARK_TEXT  = '#94a3b8'
const LIGHT_GRID = 'rgba(0, 0, 0, 0.08)'
const DARK_GRID  = 'rgba(255, 255, 255, 0.08)'

function textColor() {
  return document.body.classList.contains('dark-mode') ? DARK_TEXT : LIGHT_TEXT
}

export const applyChartTheme = (isDark) => {
  const text = isDark ? DARK_TEXT : LIGHT_TEXT
  const grid = isDark ? DARK_GRID : LIGHT_GRID

  Chart.defaults.color = text

  Object.values(Chart.instances).forEach(chart => {
    const legendLabels = chart.options.plugins?.legend?.labels
    if (legendLabels) legendLabels.color = text

    const scales = chart.options.scales
    if (scales) {
      Object.values(scales).forEach(scale => {
        if (scale.ticks) scale.ticks.color = text
        if (scale.grid)  scale.grid.color  = grid
        else             scale.grid = { color: grid }
      })
    }

    chart.update('none')
  })
}

export const chartColors = {
  primary: '#667eea',
  blue: '#36A2EB',
  red: '#FF6384',
  yellow: '#FFCE56',
  teal: '#4BC0C0',
  purple: '#9966FF',
  orange: '#FF9F40',
  green: '#27ae60',
  gray: '#6c757d',
}

export const pieBackgrounds = [
  chartColors.blue, chartColors.red, chartColors.yellow,
  chartColors.teal, chartColors.purple, chartColors.orange,
  chartColors.green, '#cbd5e0', '#a0aec0', '#718096',
]

const truncate = (label, max) => (label && label.length > max ? label.slice(0, max) + '…' : (label || ''))

// Horizontal bar avoids the rotated/cut-off x-axis labels that long
// department, vendor, and category names caused on vertical bars.
export const makeBarChart = (canvasRef, data, color, { currency = true, valueLabel = 'Total' } = {}) => {
  if (!canvasRef?.getContext('2d') || !data || data.length === 0) return null
  const fmt = v => currency ? '$' + formatNumber(Math.round(v / 1000)) + 'k' : formatNumber(v)
  const labels = data.map(d => d.label)
  return new Chart(canvasRef, {
    type: 'bar',
    data: {
      labels,
      datasets: [{ label: valueLabel, data: data.map(d => d.amount), backgroundColor: color, borderRadius: 3 }],
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: true },
      layout: { padding: { left: 12 } },
      scales: {
        x: { beginAtZero: true, ticks: { callback: fmt } },
        y: { ticks: { autoSkip: false, callback: (_, i) => truncate(labels[i], 20) } },
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            title: (items) => labels[items[0].dataIndex] || '',
            label: (item) => currency ? `${valueLabel}: $${formatCurrency(item.raw)}` : `${valueLabel}: ${formatNumber(item.raw)}`,
          },
        },
      },
    },
  })
}

// Doughnut showing the top N entries plus a grouped "Other" slice, so the
// relative share of each category is visible at a glance.
export const makeDoughnutChart = (canvasRef, data, { top = 5, currency = true, palette = pieBackgrounds } = {}) => {
  if (!canvasRef?.getContext('2d') || !data || data.length === 0) return null
  const sorted = [...data].sort((a, b) => b.amount - a.amount)
  const head = sorted.slice(0, top)
  const restTotal = sorted.slice(top).reduce((s, d) => s + d.amount, 0)
  const slices = restTotal > 0 ? [...head, { label: 'Other', amount: restTotal }] : head
  const total = slices.reduce((s, d) => s + d.amount, 0)
  const color = textColor()
  return new Chart(canvasRef, {
    type: 'doughnut',
    data: {
      labels: slices.map(d => d.label),
      datasets: [{ data: slices.map(d => d.amount), backgroundColor: palette, borderColor: '#1a202c', borderWidth: 2 }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { position: 'right', labels: { color, boxWidth: 12, font: { size: 11 }, generateLabels: (chart) => chart.data.labels.map((label, i) => ({
          text: truncate(label, 22), fillStyle: palette[i % palette.length], strokeStyle: 'transparent', index: i, fontColor: color,
        })) } },
        tooltip: {
          callbacks: {
            label: (item) => {
              const pct = total > 0 ? ((item.raw / total) * 100).toFixed(1) : 0
              const val = currency ? '$' + formatCurrency(item.raw) : formatNumber(item.raw)
              return `${item.label}: ${val} (${pct}%)`
            },
          },
        },
      },
    },
  })
}

// Line chart for trends over time (e.g. monthly revenue/spending).
export const makeLineChart = (canvasRef, data, color, { currency = true, valueLabel = 'Total' } = {}) => {
  if (!canvasRef?.getContext('2d') || !data || data.length === 0) return null
  return new Chart(canvasRef, {
    type: 'line',
    data: {
      labels: data.map(d => d.label),
      datasets: [{
        label: valueLabel,
        data: data.map(d => d.amount),
        borderColor: color,
        backgroundColor: color + '33',
        fill: true,
        tension: 0.3,
        pointRadius: 3,
        pointHoverRadius: 5,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        y: { beginAtZero: true, ticks: { callback: v => currency ? '$' + formatNumber(Math.round(v / 1000)) + 'k' : formatNumber(v) } },
        x: { ticks: { maxRotation: 0, autoSkip: true } },
      },
      plugins: {
        legend: { display: false },
        tooltip: { callbacks: { label: (item) => currency ? `$${formatCurrency(item.raw)}` : formatNumber(item.raw) } },
      },
    },
  })
}

// Stacked horizontal bar for comparing two components of a total across
// categories (e.g. base pay vs overtime by department).
export const makeStackedBarChart = (canvasRef, labels, series, colors) => {
  if (!canvasRef?.getContext('2d') || !labels || labels.length === 0) return null
  const color = textColor()
  return new Chart(canvasRef, {
    type: 'bar',
    data: {
      labels,
      datasets: series.map((s, i) => ({
        label: s.label,
        data: s.data,
        backgroundColor: colors[i % colors.length],
        borderRadius: 3,
      })),
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: true },
      layout: { padding: { left: 12 } },
      scales: {
        x: { stacked: true, beginAtZero: true, ticks: { callback: v => '$' + formatNumber(Math.round(v / 1000)) + 'k' } },
        y: { stacked: true, ticks: { autoSkip: false, callback: (_, i) => truncate(labels[i], 20) } },
      },
      plugins: {
        legend: { position: 'bottom', labels: { color, boxWidth: 12, font: { size: 11 } } },
        tooltip: {
          callbacks: {
            title: (items) => labels[items[0].dataIndex] || '',
            label: (item) => `${item.dataset.label}: $${formatCurrency(item.raw)}`,
          },
        },
      },
    },
  })
}

// Aggregate records into a daily time series, sorted chronologically.
// Returns [{ label: 'Sep 29', amount }].
export const aggregateByDay = (records, dateField, valueField) => {
  const map = new Map()
  records.forEach(r => {
    const raw = r[dateField]
    if (!raw) return
    const d = new Date(raw)
    if (isNaN(d.getTime())) return
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    map.set(key, (map.get(key) || 0) + (r[valueField] || 0))
  })
  return Array.from(map.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, amount]) => {
      const [y, m, day] = key.split('-')
      const label = new Date(Number(y), Number(m) - 1, Number(day))
        .toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
      return { label, amount }
    })
}

// Aggregate records into a monthly time series. monthFn(record) should
// return a sortable key like "2024-07"; labelFn formats it for display.
export const aggregateByMonth = (records, dateField, valueField) => {
  const map = new Map()
  records.forEach(r => {
    const raw = r[dateField]
    if (!raw) return
    const d = new Date(raw)
    if (isNaN(d)) return
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
    map.set(key, (map.get(key) || 0) + (r[valueField] || 0))
  })
  return Array.from(map.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, amount]) => {
      const [y, m] = key.split('-')
      const label = new Date(Number(y), Number(m) - 1, 1).toLocaleDateString('en-US', { month: 'short', year: '2-digit' })
      return { label, amount }
    })
}

// Fiscal years run July to June, so we need to map 1 = July
// and not January, etc.
const FISCAL_MONTH_NAMES = {
  1: 'Jul', 2: 'Aug', 3: 'Sep', 4: 'Oct', 5: 'Nov', 6: 'Dec',
  7: 'Jan', 8: 'Feb', 9: 'Mar', 10: 'Apr', 11: 'May', 12: 'Jun',
}

// Aggregate by Commonwealth fiscal_period_month.
// The field value may be a bare number ("4") or a labelled string like
// "04 (October)".
export const aggregateByFiscalPeriod = (records, periodField, valueField) => {
  const map = new Map()
  records.forEach(r => {
    const p = r[periodField]
    if (p === null || p === undefined || p === '') return
    const num = parseInt(p, 10)
    if (isNaN(num)) return
    map.set(num, (map.get(num) || 0) + (r[valueField] || 0))
  })
  return Array.from(map.entries())
    .sort(([a], [b]) => a - b)
    .map(([period, amount]) => ({ label: FISCAL_MONTH_NAMES[period] || `P${period}`, amount }))
}
