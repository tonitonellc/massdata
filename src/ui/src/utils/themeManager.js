// dark theme manager pulled from bostondata
export const themeManager = {
  isDarkMode: () => {
    if (typeof window !== "undefined") {
      return localStorage.getItem("darkMode") === "true"
    }
    return false
  },

  setDarkMode: (enabled) => {
    if (typeof window !== "undefined") {
      localStorage.setItem("darkMode", enabled.toString())
      if (enabled) {
        document.body.classList.add("dark-mode")
      } else {
        document.body.classList.remove("dark-mode")
      }
    }
  },

  toggleDarkMode: () => {
    const enabled = !themeManager.isDarkMode()
    themeManager.setDarkMode(enabled)
    return enabled
  },

  initializeTheme: () => {
    if (typeof window !== "undefined") {
      if (themeManager.isDarkMode()) {
        document.body.classList.add("dark-mode")
      }
    }
  },
}
