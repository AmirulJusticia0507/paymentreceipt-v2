import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    // Proxy ke backend Go. Dipakai di development supaya frontend dan API
    // berada di origin yang sama - ini penting karena token auth disimpan di
    // cookie SameSite=Lax, yang tidak akan terkirim ke origin berbeda.
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/healthz': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    // Pisah vendor supaya bundling framework tidak invalidate cache saat
    // kode aplikasi berubah.
    //
    // Rollout 4 mempersempit manualChunks menjadi fungsi, jadi bentuk objek
    // sudah tidak lolos type-check. Bentuk fungsi juga lebih eksplisit
    // soal paket mana yang masuk ke vendor chunk mana.
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined

          if (id.includes('@tanstack')) return 'vendor-query'
          if (id.includes('vue') || id.includes('pinia')) return 'vendor-vue'

          return undefined
        },
      },
    },
    // Vite memberi peringatan di 500 kB. Vue + router + query sudah
    // mendekati batas itu, jadi batasnya disesuaikan ke kondisi nyata.
    chunkSizeWarningLimit: 700,
  },
})