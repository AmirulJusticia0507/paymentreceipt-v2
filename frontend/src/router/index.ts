import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import type { UserRole } from '@/lib/types'

declare module 'vue-router' {
  interface RouteMeta {
    /** Halaman ini butuh sesi login. */
    requiresAuth?: boolean
    /** Role yang boleh mengakses. Kosong = semua user yang sudah login. */
    roles?: UserRole[]
    title?: string
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/pages/LoginPage.vue'),
    meta: { title: 'Masuk' },
  },
  {
    path: '/',
    component: () => import('@/layouts/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/pages/DashboardPage.vue'),
        meta: { title: 'Dashboard' },
      },
      {
        // Master data butuh login, tapi TIDAK butuh role admin untuk dibaca -
        // user biasa perlu melihat daftar department dan satuan saat mengisi
        // form payment receipt. Penulisan dicegah di router pada sisi server
        // dengan RequireRole; di sini hanya menjaga halaman create/edit.
        path: 'departments',
        name: 'departments',
        component: () => import('@/pages/DepartmentPage.vue'),
        meta: { requiresAuth: true, title: 'Department' },
      },
      {
        path: 'satuan',
        name: 'satuan',
        component: () => import('@/pages/SatuanPage.vue'),
        meta: { requiresAuth: true, title: 'Satuan' },
      },
      {
        path: 'categories',
        name: 'categories',
        component: () => import('@/pages/CategoryPage.vue'),
        meta: { requiresAuth: true, title: 'Kategori' },
      },
      {
        path: 'bases',
        name: 'bases',
        component: () => import('@/pages/BasePage.vue'),
        meta: { requiresAuth: true, title: 'Base' },
      },
    ],
  },
  {
    // Catch-all harus DI AKHIR. Kalau diletakkan lebih awal, route yang
    // belum dibuat akan ikut tertangkap dan user tidak pernah melihat 404.
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/pages/NotFoundPage.vue'),
    meta: { title: 'Halaman tidak ditemukan' },
  },
]

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

/**
 * Penjaga navigasi.
 *
 * Ini BUKAN ganti pengaman server - endpoint tetap menolak request tanpa token
 * apa pun yang dilakukan frontend. Yang kita cegah di sini hanya experience:
 * supaya user tidak sempat melihat halaman kosong lalu suddenly kena redirect.
 */
router.beforeEach(async (to) => {
  const auth = useAuthStore()

  // Sesi harus dipulihkan sebelum keputusan diambil. Tanpa ini, refresh
  // halaman di URL yang dilindungi akan selalu mengira user belum login
  // dan memantulkan mereka ke /login.
  if (!auth.initialized) {
    await auth.restore()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return {
      name: 'login',
      // Simpan tujuan supaya setelah login user kembali ke halaman yang
      // dicoba, bukan ke dashboard.
      query: to.fullPath !== '/' ? { redirect: to.fullPath } : undefined,
    }
  }

  // Sudah login tapi mau buka /login lagi - arahkan ke tujuan asalnya.
  if (to.name === 'login' && auth.isAuthenticated) {
    const redirect = to.query.redirect
    return typeof redirect === 'string' && redirect.startsWith('/') ? redirect : { name: 'dashboard' }
  }

  const required = to.meta.roles
  if (required?.length && !auth.hasRole(...required)) {
    // Ke dashboard, bukan ke /403: user mungkin salah klik, bukan memang tidak
    // berwenang. Halaman 403 bisa menyusul di Phase 10.
    return { name: 'dashboard' }
  }

  return true
})

router.afterEach((to) => {
  document.title = to.meta.title
    ? `${to.meta.title} · Payment Receipt`
    : 'Payment Receipt'
})
