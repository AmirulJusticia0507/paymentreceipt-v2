import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { VueQueryPlugin, QueryClient } from '@tanstack/vue-query'

import App from './App.vue'
import { router } from './router'
import { setUnauthorizedHandler } from './lib/http'
import { useAuthStore } from './stores/auth'
import { useThemeStore } from './stores/theme'

import '@fontsource-variable/inter'
import './styles/main.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Data dianggap segar 30 detik. Untuk sistem internal yang datanya
      // berubah sering, ini mencegah refetch berlebihan tanpa membuat user
      // melihat data basi terlalu lama.
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(VueQueryPlugin, { queryClient })

// Tema harus siap SEBELUM router mulai, supaya tidak ada kilatan putih
// di halaman gelap saat aplikasi pertama kali dimuat.
useThemeStore().init()

// Saat refresh gagal, sesi benar-benar habis. Kosongkan store lalu lempar
// ke login supaya router guard ikut reacted.
setUnauthorizedHandler(() => {
  useAuthStore().clear()
  if (router.currentRoute.value.name !== 'login') {
    void router.push({ name: 'login' })
  }
})

app.mount('#app')