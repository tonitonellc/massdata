// URL filter sync utilities.
// The app uses hash-based routing: #viewname?param1=val1&param2=val2

// Human-readable operator aliases so shared URLs are legible
const OP_ENCODE = { '>': 'gt', '<': 'lt', '>=': 'gte', '<=': 'lte', '=': 'eq', '!=': 'neq', 'similar to': 'like' }
const OP_DECODE = Object.fromEntries(Object.entries(OP_ENCODE).map(([k, v]) => [v, k]))

export const encodeOp = op => OP_ENCODE[op] ?? op
export const decodeOp = code => OP_DECODE[code] ?? code

// Read query params from the current hash, e.g. #spending?q=foo&op=gt → { q: 'foo', op: 'gt' }
export function getHashParams() {
  const qi = window.location.hash.indexOf('?')
  if (qi === -1) return {}
  return Object.fromEntries(new URLSearchParams(window.location.hash.slice(qi + 1)))
}

// Write query params into the hash without changing the view segment.
// Entries with null / undefined / '' are omitted so the URL stays clean.
// Bails out if no hash view is active yet — prevents params from landing in the
// URL search string (?foo=bar) instead of the hash (#view?foo=bar).
export function setHashParams(params) {
  const hash = window.location.hash
  const qi = hash.indexOf('?')
  const base = qi === -1 ? hash : hash.slice(0, qi)
  if (!base || base === '#') return
  const clean = Object.fromEntries(
    Object.entries(params).filter(([, v]) => v !== '' && v !== null && v !== undefined)
  )
  const qs = new URLSearchParams(clean).toString()
  history.replaceState(null, '', qs ? `${base}?${qs}` : base)
}
