<template>
  <div id="app">
    <TheNavbar v-model="view" />

    <div class="main-container">
      <div v-if="view === 'home'" class="home-view">
        <h1>Make Big Digs into Commonwealth of Massachusetts datasets</h1>
        <p class="subtitle">
          Welcome to <a href="https://baystate.info">BayState.info</a>!<br>
          Public records provided by the Commonwealth of Massachusetts are available
          to this application immediately upon publishing, and the most recent items
          are shown first by default.
        </p>

        <div class="developer-section">
          <div class="developer-card">
            <div class="developer-bio">
              <img
                src="https://avatars.githubusercontent.com/u/32644679?s=400&u=2f837bf05341ea68ae205a9aba576793c87e477a&v=4"
                alt="Toni Noble"
                class="developer-photo"
              >
              <h2>Built by Toni Noble</h2>
              <p>
                <b>Software Engineer & Certified Wellness Professional</b><br>
                Founder of Toni Tone & Sto Lat | NASM-CPT, CNC, CWC
              </p>
            </div>
            <p class="developer-email">
              <a href="mailto:toni@tonitoned.com">toni@tonitoned.com</a>
            </p>
            <div class="button-group-home">
              <a href="https://linkedin.com/in/tonistark" target="_blank" class="link-btn linkedin">LinkedIn</a>
              <a href="https://github.com/dethmasque" target="_blank" class="link-btn github">GitHub</a>
              <a href="https://www.paypal.com/donate/?hosted_button_id=J8N8PPM9QZTGS" target="_blank" class="link-btn donate">Support this project</a>
            </div>
          </div>

          <div class="developer-card features-card">
            <div class="brace-layout">
              <span class="the-brace">{</span>
              <div class="features-content">
                <small>
                  <b>This website is in active development.</b>
                  <br><br>
                  <b>July 1st, 2026 Release:</b><br>
                  • Initial deploy with support for Spending, Revenue, Payroll, and 
                    Settlements and Judgments datasets. <br>
                  • Annual Reports available for comparing Spending and Revenue data,
                    year-over-year. <br> 
                  • Location Map Filter available for automatically querying Spending 
                    data for a selected city and / or state. <br> 
                  • Sharing and saving queries is possible with URL parameters populating 
                    search filters. <br>
                  • Dark mode available via toggle in footer or <code>`</code> (backtick) 
                    keyboard shortcut. <br><br> 
                </small>
              </div>
            </div>
          </div>
        </div>
      </div>

      <SpendingView     v-if="view === 'spending'" />
      <RevenueView      v-if="view === 'revenue'" />
      <PayrollView      v-if="view === 'payroll'" />
      <AnnualReportView v-if="view === 'annual'" />
      <SettlementsView  v-if="view === 'settlements'" />
    </div>

    <TheFooter />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import TheFooter from './components/TheFooter.vue'
import TheNavbar from './components/TheNavbar.vue'
import AnnualReportView from './views/AnnualReportView.vue'
import PayrollView from './views/PayrollView.vue'
import RevenueView from './views/RevenueView.vue'
import SettlementsView from './views/SettlementsView.vue'
import SpendingView from './views/SpendingView.vue'

const VALID_VIEWS = ['home', 'spending', 'revenue', 'payroll', 'annual', 'settlements']

const view = ref('home')

onMounted(() => {
  try {
    const [navEntry] = performance.getEntriesByType('navigation')
    if (navEntry?.type === 'reload') {
      Object.keys(localStorage)
        .filter(k => k.startsWith('massdata-filter-'))
        .forEach(k => localStorage.removeItem(k))
    }
  } catch (e) {}

  const hashStr = window.location.hash.replace('#', '')
  const viewName = hashStr.split('?')[0]
  if (VALID_VIEWS.includes(viewName)) {
    view.value = viewName
  }
})
</script>
