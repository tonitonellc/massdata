<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { applyChartTheme } from '../utils/chartUtils'
import { themeManager } from '../utils/themeManager'

const isDark = ref(false)

const toggle = () => {
  isDark.value = themeManager.toggleDarkMode()
  applyChartTheme(isDark.value)
}

const handleKeydown = (event) => {
  if (event.key !== '`') return
  const tag = document.activeElement?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
  toggle()
}

onMounted(() => {
  themeManager.initializeTheme()
  isDark.value = themeManager.isDarkMode()
  applyChartTheme(isDark.value)
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <button
    class="theme-toggle-btn"
    @click="toggle"
    :title="isDark ? 'Switch to light mode (`)' : 'Switch to dark mode (`)'"
    :aria-label="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
  >
    {{ isDark ? '☀️' : '🌙' }}
  </button>
</template>

<style scoped>
.theme-toggle-btn {
  background: none;
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 6px;
  color: inherit;
  cursor: pointer;
  font-size: 1.1rem;
  line-height: 1;
  padding: 0.3rem 0.55rem;
  transition: background 0.2s, border-color 0.2s;
  display: inline-flex;
  align-items: center;
}

.theme-toggle-btn:hover {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.5);
}
</style>
