export const SERVER_ADDR = process.env.VUE_APP_SERVER_ADDR || ""

export const getApiUrl = (endpoint) => `${SERVER_ADDR}${endpoint}`
