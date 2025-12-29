import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist', // Папка, куда попадет готовый билд
  },
  server: {
    host: true,
    port: 3000
  }
})