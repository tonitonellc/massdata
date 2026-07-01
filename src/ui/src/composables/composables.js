import { ref, watch } from 'vue'
import { debounce } from '../utils/debounce'
import { getHashParams, setHashParams } from '../utils/urlFilters'

export function useSortable() {
  const sortField = ref(null)
  const sortDirection = ref('asc')

  const sortBy = (field) => {
    if (sortField.value === field) {
      sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortField.value = field
      sortDirection.value = 'asc'
    }
  }

  const getSortIndicator = (field) => {
    if (sortField.value !== field) return ''
    return sortDirection.value === 'asc' ? '↑' : '↓'
  }

  const sortRecords = (records, field) => {
    if (!field) return records
    return [...records].sort((a, b) => {
      let valA = a[field]
      let valB = b[field]
      if (typeof valA === 'string' && !isNaN(valA)) valA = parseFloat(valA)
      if (typeof valB === 'string' && !isNaN(valB)) valB = parseFloat(valB)
      if (valA < valB) return sortDirection.value === 'asc' ? -1 : 1
      if (valA > valB) return sortDirection.value === 'asc' ? 1 : -1
      return 0
    })
  }

  return { sortField, sortDirection, sortBy, getSortIndicator, sortRecords }
}

export function useHelpModal() {
  const showModal = ref(false)
  const modalContent = ref({ title: '', url: '', description: '' })

  const openHelpModal = (title, url, description) => {
    modalContent.value = { title, url, description }
    showModal.value = true
  }

  const closeModal = () => { showModal.value = false }

  return { showModal, modalContent, openHelpModal, closeModal }
}

export function useCollapsibleSections() {
  const showCharts = ref(true)
  const showTable = ref(true)
  return { showCharts, showTable }
}

// useUrlSync — bidirectional sync between filter state and URL hash params.
// getParams: () => Object  maps current state to URL params; falsy values are omitted.
// deps:      reactive refs to watch for changes.
// Returns readUrlParams(applyFn) — call in onMounted to apply any URL params to state.
export function useUrlSync(getParams, deps) {
  const syncUrl = debounce(() => setHashParams(getParams()), 250)
  watch(deps, syncUrl, { deep: true })

  return function readUrlParams(applyFn) {
    const params = getHashParams()
    if (Object.keys(params).length > 0) {
      applyFn(params)
      // Cancel the write-back the watch just scheduled — params are already in the URL.
      syncUrl.cancel()
    }
  }
}
