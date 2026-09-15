<script setup>
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({ visible: Boolean })
const emit = defineEmits(['select'])

const mapEl = ref(null)
let map, tileLayer, stateLayer

const isDark = ref(document.body.classList.contains('dark-mode'))
const geocoding = ref(false)

const observer = new MutationObserver(() => {
  isDark.value = document.body.classList.contains('dark-mode')
})

const TILES = {
  light: 'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png?key=cb1_3mde_1_7fab56d4350fc17e8f648e77',
  dark:  'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png?key=cb1_3mde_1_7fab56d4350fc17e8f648e77',
}
const ATTR = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>'

const STATE_ABBREVS = {
  Alabama: 'AL', Alaska: 'AK', Arizona: 'AZ', Arkansas: 'AR', California: 'CA',
  Colorado: 'CO', Connecticut: 'CT', Delaware: 'DE', Florida: 'FL', Georgia: 'GA',
  Hawaii: 'HI', Idaho: 'ID', Illinois: 'IL', Indiana: 'IN', Iowa: 'IA',
  Kansas: 'KS', Kentucky: 'KY', Louisiana: 'LA', Maine: 'ME', Maryland: 'MD',
  Massachusetts: 'MA', Michigan: 'MI', Minnesota: 'MN', Mississippi: 'MS',
  Missouri: 'MO', Montana: 'MT', Nebraska: 'NE', Nevada: 'NV', 'New Hampshire': 'NH',
  'New Jersey': 'NJ', 'New Mexico': 'NM', 'New York': 'NY', 'North Carolina': 'NC',
  'North Dakota': 'ND', Ohio: 'OH', Oklahoma: 'OK', Oregon: 'OR', Pennsylvania: 'PA',
  'Rhode Island': 'RI', 'South Carolina': 'SC', 'South Dakota': 'SD', Tennessee: 'TN',
  Texas: 'TX', Utah: 'UT', Vermont: 'VT', Virginia: 'VA', Washington: 'WA',
  'West Virginia': 'WV', Wisconsin: 'WI', Wyoming: 'WY', 'District of Columbia': 'DC',
}

function stateStyle() {
  return isDark.value
    ? { fillColor: '#667eea', fillOpacity: 0.12, color: '#818cf8', weight: 1.5 }
    : { fillColor: '#3b82f6', fillOpacity: 0.12, color: '#60a5fa', weight: 1.5 }
}
const hoverStyle = { fillColor: '#667eea', fillOpacity: 0.4, color: '#667eea', weight: 2.5 }

async function reverseGeocode(lat, lng) {
  try {
    const r = await fetch(
      `https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lng}&format=json&zoom=10`,
      { headers: { 'Accept-Language': 'en' } }
    )
    const d = await r.json()
    const addr = d.address || {}
    const city = addr.city || addr.town || addr.village || addr.suburb || ''
    const stateCode = (addr['ISO3166-2-lvl4'] || '').replace('US-', '')
    return { city, state: stateCode }
  } catch {
    return { city: '', state: '' }
  }
}

function makePopupContent(label) {
  const bg  = isDark.value ? '#1e2a3a' : '#ffffff'
  const fg  = isDark.value ? '#e2e8f0' : '#2c3e50'
  const sub = isDark.value ? '#94a3b8' : '#6c757d'
  return `
    <div style="min-width:180px;background:${bg};color:${fg};padding:2px 0">
      <div style="font-size:0.8rem;color:${sub};margin-bottom:4px">Location filter</div>
      <strong style="font-size:1rem">${label}</strong>
      <button id="map-filter-btn" style="
        display:block;width:100%;margin-top:10px;
        background:#667eea;color:white;border:none;border-radius:6px;
        padding:7px 0;cursor:pointer;font-size:0.85rem;font-weight:600;
      ">Filter to ${label}</button>
    </div>`
}

onMounted(async () => {
  observer.observe(document.body, { attributes: true, attributeFilter: ['class'] })

  map = L.map(mapEl.value, { center: [42.35, -71.9], zoom: 7, minZoom: 3, maxZoom: 18 })

  tileLayer = L.tileLayer(isDark.value ? TILES.dark : TILES.light, {
    attribution: ATTR, maxZoom: 18,
  }).addTo(map)

  try {
    const res = await fetch('https://raw.githubusercontent.com/PublicaMundi/MappingAPI/master/data/geojson/us-states.json')
    const geojson = await res.json()
    stateLayer = L.geoJSON(geojson, {
      style: stateStyle,
      onEachFeature(feature, layer) {
        const abbrev = STATE_ABBREVS[feature.properties.name]
        layer.bindTooltip(feature.properties.name, { sticky: true, opacity: 0.9 })
        layer.on({
          mouseover: e => e.target.setStyle(hoverStyle),
          mouseout:  e => stateLayer.resetStyle(e.target),
          click: e => {
            // At low zoom, clicking the state polygon = state-only filter.
            // At high zoom, let the event bubble to the map click handler so
            // Nominatim can resolve the specific city the user clicked.
            if (map.getZoom() < 8) {
              L.DomEvent.stopPropagation(e)
              if (abbrev) emit('select', { state: abbrev, city: '' })
            }
          },
        })
      },
    }).addTo(map)
  } catch (e) {
    console.error('Failed to load state boundaries:', e)
  }

  // City-level: reverse-geocode any map click at zoom ≥ 8.
  // State polygon clicks don't stop propagation at high zoom, so they reach here.
  map.on('click', async e => {
    if (map.getZoom() < 8) return
    geocoding.value = true
    const { city, state } = await reverseGeocode(e.latlng.lat, e.latlng.lng)
    geocoding.value = false
    if (!city && !state) return
    const label = [city, state].filter(Boolean).join(', ')
    const popup = L.popup({ className: 'spending-map-popup', closeButton: true })
      .setLatLng(e.latlng)
      .setContent(makePopupContent(label))
    popup.on('add', () => {
      const btn = popup.getElement()?.querySelector('#map-filter-btn')
      if (btn) {
        L.DomEvent.on(btn, 'click', () => {
          emit('select', { city: city.toUpperCase(), state })
          map.closePopup(popup)
        })
      }
    })
    popup.openOn(map)
  })
})

watch(isDark, dark => {
  if (!map) return
  map.removeLayer(tileLayer)
  tileLayer = L.tileLayer(dark ? TILES.dark : TILES.light, { attribution: ATTR, maxZoom: 18 }).addTo(map)
  stateLayer?.setStyle(stateStyle)
})

watch(() => props.visible, val => {
  if (val) nextTick(() => map?.invalidateSize())
})

onBeforeUnmount(() => {
  observer.disconnect()
  map?.remove()
})
</script>

<template>
  <div class="spending-map-wrap" :class="{ dark: isDark }">
    <div ref="mapEl" class="spending-map"></div>
    <div v-if="geocoding" class="map-geocoding-badge">Locating…</div>
    <div class="map-hint">
      Click a state to filter by state &nbsp;·&nbsp; Zoom in then click to filter by city
    </div>
  </div>
</template>

<style scoped>
.spending-map-wrap {
  position: relative;
  border-radius: 0 0 8px 8px;
  overflow: hidden;
  border: 1px solid #dee2e6;
  border-top: none;
}

.spending-map-wrap.dark {
  border-color: #334155;
}

.spending-map {
  height: 440px;
  width: 100%;
  z-index: 0;
}

.map-hint {
  background: #f1f5f9;
  color: #64748b;
  font-size: 0.78rem;
  padding: 0.5rem 1rem;
  text-align: center;
  border-top: 1px solid #dee2e6;
}

.spending-map-wrap.dark .map-hint {
  background: #0f1923;
  color: #64748b;
  border-color: #334155;
}

.map-geocoding-badge {
  position: absolute;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(102, 126, 234, 0.92);
  color: white;
  padding: 5px 14px;
  border-radius: 20px;
  font-size: 0.8rem;
  font-weight: 600;
  pointer-events: none;
  z-index: 1000;
}

/* ── Leaflet zoom controls ────────────────────────────────────────── */
.spending-map-wrap :deep(.leaflet-bar a) {
  background-color: #ffffff;
  color: #2c3e50;
  border-color: #ccc;
}
.spending-map-wrap :deep(.leaflet-bar a:hover) {
  background-color: #f0f0f0;
}
.spending-map-wrap.dark :deep(.leaflet-bar a) {
  background-color: #1e2a3a;
  color: #e2e8f0;
  border-color: #334155;
}
.spending-map-wrap.dark :deep(.leaflet-bar a:hover) {
  background-color: #2d3a4a;
  color: #ffffff;
}
.spending-map-wrap.dark :deep(.leaflet-bar) {
  border-color: #334155;
}

/* ── Leaflet attribution ────────────────────────────────────────────*/
.spending-map-wrap.dark :deep(.leaflet-control-attribution) {
  background: rgba(15, 25, 35, 0.85);
  color: #64748b;
}
.spending-map-wrap.dark :deep(.leaflet-control-attribution a) {
  color: #818cf8;
}

/* ── Popup overrides ────────────────────────────────────────────────*/
:global(.spending-map-popup .leaflet-popup-content-wrapper) {
  border-radius: 8px;
  padding: 4px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
}
:global(.spending-map-popup .leaflet-popup-content) {
  margin: 10px 14px;
}
:global(body.dark-mode .spending-map-popup .leaflet-popup-content-wrapper) {
  background: #1e2a3a;
  border: 1px solid #334155;
}
:global(body.dark-mode .spending-map-popup .leaflet-popup-tip) {
  background: #1e2a3a;
}
</style>
