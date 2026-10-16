/**
 * Store untuk preferensi tampilan.
 *
 * Hanya menyimpan hal yang benar-benar preferensi user: tema. Token auth TIDAK
 * ada di sini karena token hidup di cookie HttpOnly - lihat AGENTS.md § Keamanan.
 */

import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

type Theme = 'light' | 'dark'

const STORAGE_KEY = 'pr-theme'

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>('light')

  function applyTo(documentEl: HTMLElement): void {
    documentEl.classList.toggle('dark', theme.value === 'dark')
  }

  function init(): void {
    const saved = localStorage.getItem(STORAGE_KEY)

    if (saved === 'light' || saved === 'dark') {
      theme.value = saved
    } else if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      // Belum pernah memilih manual - ikuti preferensi sistem.
      theme.value = 'dark'
    }

    applyTo(document.documentElement)
  }

  function toggle(): void {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  // Menyimpan ke localStorage setiap berubah, lalu langsung diterapkan ke DOM.
  // Persistensi ditulis di sini, bukan di setiap pemanggil toggle().
  watch(theme, (value) => {
    localStorage.setItem(STORAGE_KEY, value)
    applyTo(document.documentElement)
  })

  return { theme, init, toggle }
})
