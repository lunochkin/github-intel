import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/summary': { target: 'http://localhost:8800', changeOrigin: true },
      '/healthz': { target: 'http://localhost:8800', changeOrigin: true },
    },
  },
})
