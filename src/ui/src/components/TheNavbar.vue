<template>
  <nav class="navbar" style="z-index: 2000;" ref="navbarRef">
    <div class="nav-content">
      <div class="nav-brand" @click="navigate('home')" style="cursor: pointer;">
        BayState.info
        <small>beta!</small>
      </div>

      <button class="hamburger" @click.stop="toggleMenu" :class="{ active: menuOpen }">
        <span></span>
        <span></span>
        <span></span>
      </button>

      <div class="nav-links" :class="{ open: menuOpen }">
        <button @click="navigate('home')" :class="{ active: modelValue === 'home' }">Home</button>

        <div class="nav-section">
          <button class="nav-section-header" @click.stop="toggleSection('fiscal')">
            <span class="chevron">{{ expandedSections.fiscal ? '▼' : '▶' }}</span>
            Fiscal & Admin
          </button>
          <div v-show="expandedSections.fiscal" class="nav-section-items">
            <button @click="navigate('spending')"     :class="{ active: modelValue === 'spending' }">Spending</button>
            <button @click="navigate('revenue')"      :class="{ active: modelValue === 'revenue' }">Revenue</button>
            <button @click="navigate('payroll')"      :class="{ active: modelValue === 'payroll' }">Payroll</button>
            <button @click="navigate('settlements')"  :class="{ active: modelValue === 'settlements' }">Settlements & Judgments</button>
            <button @click="navigate('annual')"       :class="{ active: modelValue === 'annual' }">Annual Reports</button>
          </div>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'

const props = defineProps({
  modelValue: { type: String, required: true }
})
const emit = defineEmits(['update:modelValue'])

const menuOpen = ref(false)
const navbarRef = ref(null)
const expandedSections = ref({ fiscal: false })

const navigate = (view) => {
  emit('update:modelValue', view)
  window.location.hash = view
  menuOpen.value = false
  collapseAllSections()
}

const toggleMenu = () => { menuOpen.value = !menuOpen.value }

const toggleSection = (section) => {
  const isCurrentlyOpen = expandedSections.value[section]
  if (window.innerWidth > 1024) {
    Object.keys(expandedSections.value).forEach(k => { expandedSections.value[k] = false })
    if (!isCurrentlyOpen) expandedSections.value[section] = true
  } else {
    expandedSections.value[section] = !isCurrentlyOpen
  }
}

const collapseAllSections = () => {
  Object.keys(expandedSections.value).forEach(k => { expandedSections.value[k] = false })
}

const handleOutsideClick = (e) => {
  if (navbarRef.value && !navbarRef.value.contains(e.target)) collapseAllSections()
}

onMounted(() => document.addEventListener('click', handleOutsideClick))
onUnmounted(() => document.removeEventListener('click', handleOutsideClick))
</script>
