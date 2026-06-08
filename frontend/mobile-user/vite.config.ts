import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:58080'
const wsProxyTarget = process.env.VITE_WS_PROXY_TARGET || 'ws://127.0.0.1:58081'
const srsProxyTarget = process.env.VITE_SRS_PROXY_TARGET || 'http://127.0.0.1:1985'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
      '/ws': {
        target: wsProxyTarget,
        changeOrigin: true,
        ws: true,
      },
      '/rtc': {
        target: srsProxyTarget,
        changeOrigin: true,
      },
    },
  },
})
