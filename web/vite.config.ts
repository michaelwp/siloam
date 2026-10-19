import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // during `npm run dev`, forward API calls to the Go server
    proxy: { '/api': 'http://localhost:8080' },
  },
})
