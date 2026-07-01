<template>
  <footer class="site-footer">
    <div class="footer-content">
      <p class="footer-copyright">&copy; 2026 Toni Tone, LLC. All rights reserved.</p>
      <div class="footer-actions">
        <ThemeToggle />
        <button class="footer-link-btn" @click="showTerms = true">Terms and Conditions</button>
      </div>
    </div>
  </footer>

  <div v-if="showTerms" class="modal-overlay" @click.self="showTerms = false">
    <div class="modal-content terms-modal">
      <div class="modal-header">
        <h2>Terms and Conditions</h2>
        <button @click="showTerms = false" class="close-btn">&times;</button>
      </div>
      <div class="modal-body scrollable-modal-body">
        <p><small>Last Updated: June 25th, 2026</small></p>

        <h3>1. Acceptance of Terms</h3>
        <p>By accessing and using this website (the "Platform"), you agree to be bound by these Terms and Conditions. This Platform is provided by Toni Tone, LLC ("we," "us," or "our") as a tool for visualizing and interacting with public datasets.</p>

        <h3>2. Nature of Service</h3>
        <p>The Platform serves as a visualization and interface layer for public data. We do not own the underlying data. The Platform is provided "as is" and "as available" for informational and analytical purposes only.</p>

        <h3>3. Data License & Attribution</h3>
        <ul>
          <li><strong>Data Source:</strong> Spending data is provided by the Commonwealth of Massachusetts via the CTHRU Spending platform.</li>
          <li><strong>Accuracy:</strong> While we strive to display data accurately, we do not guarantee the correctness, completeness, or timeliness of the information provided by the Commonwealth.</li>
        </ul>

        <h3>4. Intellectual Property</h3>
        <ul>
          <li><strong>Software and Design:</strong> All rights, titles, and interests in the Platform's user interface, design, graphics, and proprietary code are the exclusive property of Toni Tone, LLC.</li>
        </ul>

        <h3>5. Disclaimer of Warranties</h3>
        <p>Your use of the Platform is at your sole risk. We disclaim all warranties of any kind, whether express or implied. We are not responsible for any decisions made based on the data visualizations provided.</p>

        <h3>6. Contact Information</h3>
        <p>For questions regarding these terms, please contact <a href="mailto:toni@tonitoned.com">toni@tonitoned.com</a>.</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import ThemeToggle from './ThemeToggle.vue'
const showTerms = ref(false)
</script>

<style scoped>
.footer-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.footer-link-btn {
  background: none;
  border: none;
  color: #667eea;
  text-decoration: underline;
  cursor: pointer;
  font-size: 0.9rem;
  padding: 0;
}

.footer-link-btn:hover {
  color: #5568d3;
}

.scrollable-modal-body {
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 1rem;
  text-align: left;
}

.terms-modal {
  max-width: 800px;
  width: 90%;
}

h3 {
  margin-top: 1.5rem;
  margin-bottom: 0.5rem;
  font-size: 1.1rem;
  color: #333;
}
</style>
