<script setup lang="ts">
/**
 * Layout utama aplikasi: sidebar + topbar + area konten.
 *
 * Dipisah dari halaman supaya navigasi tidak ikut re-mount setiap kali pindah
 * route - kalau sidebar ada di tiap halaman, tiap perpindahan akan rebuild
 * seluruh chrome.
 */
import { computed, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import {
  FileText,
  Bell,
  FolderTree,
  LayoutDashboard,
  LogOut,
  Menu,
  Moon,
  Package,
  Receipt,
  Settings,
  Sun,
  Tags,
  Truck,
  Users,
  X,
} from 'lucide-vue-next'

import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import type { UserRole } from '@/lib/types'
import { http, requestWithMeta } from '@/lib/http'

interface ReceiptNotice { id: number; kode: string; status: string; department_nama: string; days_waiting: number }

interface NavItem {
  label: string
  to: string
  icon: typeof LayoutDashboard
  roles?: UserRole[]
  // Modul belum selesai (Phase 5-7). Ditampilkan greyed supaya struktur menu
  // sudah terlihat sekarang tanpa menunggu semuanya siap.
  disabled?: boolean
}

interface NavGroup {
  title: string
  items: NavItem[]
}

const NAV_GROUPS: NavGroup[] = [
  {
    title: 'Umum',
    items: [
      { label: 'Dashboard', to: '/', icon: LayoutDashboard },
      { label: 'Payment Receipt', to: '/payment-receipts', icon: Receipt },
      { label: 'Arsip PR', to: '/pr-archives', icon: FileText },
      { label: 'Laporan Operasional', to: '/reports/operational', icon: FileText },
    ],
  },
  {
    title: 'Master Data',
    items: [
      { label: 'Department', to: '/departments', icon: Package },
      { label: 'Satuan', to: '/satuan', icon: Tags },
      { label: 'Kategori', to: '/categories', icon: Tags },
      { label: 'Base', to: '/bases', icon: FolderTree },
      { label: 'Sumber Beli', to: '/sources', icon: Truck },
    ],
  },
  {
    title: 'Pengajuan',
    items: [
      { label: 'Request', to: '/requests', icon: FolderTree },
      { label: 'Request Base', to: '/request-bases', icon: FolderTree },
    ],
  },
  {
    title: 'Administrasi',
    items: [
      { label: 'Pengguna', to: '/users', icon: Users, roles: ['admin'] },
      { label: 'Audit Log', to: '/audit-logs', icon: FileText, roles: ['admin'] },
    ],
  },
]

const auth = useAuthStore()
const theme = useThemeStore()
const route = useRoute()
const router = useRouter()

const sidebarOpen = ref(false)
const collapsed = ref(localStorage.getItem('pr-sidebar-collapsed') === 'true')
const userMenuOpen = ref(false)
const noticesOpen = ref(false)
const notices = ref<ReceiptNotice[]>([])
const noticesError = ref('')
const noticesLoading = ref(false)
const openingNoticeId = ref<number | null>(null)

async function loadNotices(): Promise<void> {
  noticesLoading.value = true
  noticesError.value = ''
  try {
    const result = await requestWithMeta<ReceiptNotice[]>('/api/v1/notifications/receipts')
    notices.value = result.data
  } catch {
    noticesError.value = 'Pengingat tidak dapat dimuat.'
  } finally {
    noticesLoading.value = false
  }
}

async function openNotice(notice: ReceiptNotice): Promise<void> {
  openingNoticeId.value = notice.id
  noticesError.value = ''
  try {
    await http.post<void>(`/api/v1/notifications/receipts/${notice.id}/read`)
    notices.value = notices.value.filter((item) => item.id !== notice.id)
    noticesOpen.value = false
    await router.push(`/payment-receipts/${notice.id}`)
  } catch {
    noticesError.value = 'Notifikasi belum bisa ditandai dibaca. Coba lagi.'
  } finally {
    openingNoticeId.value = null
  }
}

watch(() => route.fullPath, () => { void loadNotices() }, { immediate: true })

function toggleCollapse(): void {
  collapsed.value = !collapsed.value
  localStorage.setItem('pr-sidebar-collapsed', String(collapsed.value))
}

function toggleUserMenu(): void {
  userMenuOpen.value = !userMenuOpen.value
}

// Tutup user menu saat pindah halaman.
watch(() => route.fullPath, () => {
  userMenuOpen.value = false
  sidebarOpen.value = false
})

/**
 * Keluar lalu kembali ke halaman login.
 *
 * Navigasi HARUS eksplisit. Router guard hanya berjalan saat ada perpindahan
 * route - kalau kita hanya mengosongkan store, user akan tetap melihat
 * dashboard dengan data yang sudah tidak boleh dia akses.
 */
async function handleLogout(): Promise<void> {
  await auth.logout()
  await router.push({ name: 'login' })
}

const visibleGroups = computed(() =>
  NAV_GROUPS.map((group) => ({
    ...group,
    items: group.items.filter((item) => !item.roles || auth.hasRole(...item.roles)),
  })).filter((group) => group.items.length > 0),
)

// Tutup drawer setiap pindah halaman - kalau tidak, di mobile sidebar akan
// tetap menutupi konten setelah user memilih menu.
watch(() => route.fullPath, () => {
  sidebarOpen.value = false
})

const initials = computed(() => {
  const name = auth.user?.full_name || auth.user?.username || '?'
  return name
    .split(' ')
    .map((part) => part[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
})

const roleLabel: Record<UserRole, string> = {
  admin: 'Administrator',
  ga_staff: 'General Affairs',
  approver: 'Approver',
  viewer: 'Pengunjung',
  requester: 'Requester (Pemohon)',
}
</script>

<template>
  <div class="min-h-dvh bg-surface">
    <!-- Backdrop untuk drawer mobile -->
    <div
      v-if="sidebarOpen"
      class="fixed inset-0 z-30 bg-black/40 lg:hidden"
      aria-hidden="true"
      @click="sidebarOpen = false"
    />

    <!-- Sidebar -->
    <aside
      class="fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-default bg-raised transition-transform-[var(--transition-fast)] lg:translate-x-0"
      :class="[sidebarOpen ? 'translate-x-0' : '-translate-x-full', collapsed ? 'lg:w-16' : 'lg:w-64']"
      aria-label="Navigasi utama"
    >
      <!-- Brand -->
      <div class="flex h-16 shrink-0 items-center gap-2.5 border-b border-default px-3.5">
        <div
          class="flex items-center justify-center rounded-xl bg-white p-1.5 shadow-2xs ring-1 ring-black/5 dark:bg-white dark:ring-white/20 transition-all shrink-0"
          :class="collapsed ? 'size-10' : 'h-10 px-2 max-w-[130px]'"
        >
          <img
            src="/susi-air-logo.png"
            alt="Logo Susi Air"
            class="h-full w-auto object-contain"
          />
        </div>
        <div v-if="!collapsed" class="min-w-0">
          <p class="truncate text-sm font-bold text-ink">Payment Receipt</p>
          <p class="truncate text-xs font-semibold text-brand-600 dark:text-brand-400">Susi Air · GA</p>
        </div>
        <button
          type="button"
          class="ml-auto rounded-md p-1 text-ink-muted hover:text-ink lg:hidden"
          aria-label="Tutup menu"
          @click="sidebarOpen = false"
        >
          <X class="size-5" aria-hidden="true" />
        </button>
        <button
          type="button"
          class="hidden lg:block ml-auto rounded-md p-1 text-ink-muted hover:text-ink"
          :aria-label="collapsed ? 'Buka sidebar' : 'Tutup sidebar'"
          @click="toggleCollapse"
        >
          <Menu class="size-5" aria-hidden="true" />
        </button>
      </div>

      <!-- Menu -->
      <nav class="scrollbar-thin flex-1 space-y-6 overflow-y-auto px-3 py-4">
        <div v-for="group in visibleGroups" :key="group.title">
          <p class="px-3 pb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">
            {{ group.title }}
          </p>
          <ul class="space-y-0.5">
            <li v-for="item in group.items" :key="item.to">
              <RouterLink
                v-if="!item.disabled"
                :to="item.to"
                class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-ink-muted transition-[var(--transition-fast)] hover:bg-brand-50 hover:text-ink dark:hover:bg-brand-950/40"
                active-class="!bg-brand-50 !text-brand-700 dark:!bg-brand-950/50 dark:!text-brand-300"
              >
                <component :is="item.icon" class="size-4 shrink-0" aria-hidden="true" />
                <span v-if="!collapsed">{{ item.label }}</span>
              </RouterLink>
              <div
                v-else
                class="flex cursor-not-allowed items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-ink-muted/50"
                :title="`${item.label} - sedang dibangun`"
              >
                <component :is="item.icon" class="size-4 shrink-0" aria-hidden="true" />
                <span v-if="!collapsed">{{ item.label }}</span>
                <span
                  class="ml-auto rounded bg-default px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide"
                >
                  Soon
                </span>
              </div>
            </li>
          </ul>
        </div>
      </nav>

      <!-- User -->
      <div class="shrink-0 border-t border-default p-3">
        <div class="flex items-center gap-3 rounded-lg px-2 py-1.5">
          <div
            class="flex size-8 shrink-0 items-center justify-center rounded-full bg-brand-100 text-xs font-semibold text-brand-700 dark:bg-brand-950 dark:text-brand-300"
          >
            {{ initials }}
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium text-ink">
              {{ auth.user?.full_name || auth.user?.username }}
            </p>
            <p class="truncate text-xs text-ink-muted">
              {{ auth.user ? roleLabel[auth.user.role] : '' }}
            </p>
          </div>
        </div>
        <button
          type="button"
          class="mt-1 flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-ink-muted transition-[var(--transition-fast)] hover:bg-status-cancelled/10 hover:text-status-cancelled"
          @click="handleLogout"
        >
          <LogOut class="size-4" aria-hidden="true" />
          Keluar
        </button>
      </div>
    </aside>

    <!-- Konten -->
    <div class="lg:pl-64">
      <!-- Topbar -->
      <header
        class="sticky top-0 z-20 flex h-16 items-center gap-4 border-b border-default bg-raised/80 px-4 backdrop-blur sm:px-6"
      >
        <button
          type="button"
          class="rounded-md p-2 text-ink-muted hover:bg-default hover:text-ink lg:hidden"
          aria-label="Buka menu"
          @click="sidebarOpen = true"
        >
          <Menu class="size-5" aria-hidden="true" />
        </button>

        <nav v-if="!collapsed" class="hidden lg:flex items-center gap-1 text-xs text-ink-muted" aria-label="Breadcrumb">
          <RouterLink to="/" class="hover:text-brand-600 transition-colors">Home</RouterLink>
          <span class="text-ink-muted/50">/</span>
          <span class="font-medium text-ink">{{ route.meta.title ?? 'Dashboard' }}</span>
        </nav>

        <h1 class="lg:hidden truncate text-base font-semibold text-ink">
          {{ route.meta.title ?? 'Dashboard' }}
        </h1>

        <div class="ml-auto flex items-center gap-2 relative">
          <div class="relative">
            <button type="button" class="relative rounded-lg p-2 text-ink-muted hover:bg-default hover:text-ink focus-visible:outline-2 focus-visible:outline-brand-500" aria-label="Pengingat receipt" :aria-expanded="noticesOpen" @click="noticesOpen = !noticesOpen">
              <Bell class="size-5" aria-hidden="true" />
              <span v-if="notices.length" class="absolute right-0.5 top-0.5 min-w-4 rounded-full bg-status-cancelled px-1 text-[10px] leading-4 text-white">{{ notices.length }}</span>
            </button>
            <section v-if="noticesOpen" class="absolute right-0 top-full z-50 mt-2 w-[min(22rem,calc(100vw-2rem))] rounded-xl border border-default bg-raised p-3 shadow-lg" aria-label="Receipt perlu perhatian">
              <div class="mb-2 flex items-center justify-between"><h2 class="text-sm font-semibold text-ink">Receipt perlu perhatian</h2><button class="text-xs text-brand-600 hover:text-brand-700" @click="loadNotices">Muat ulang</button></div>
              <p v-if="noticesLoading" class="py-4 text-center text-sm text-ink-muted">Memuat pengingat…</p>
              <p v-else-if="noticesError" class="py-4 text-sm text-status-cancelled">{{ noticesError }}</p>
              <p v-else-if="!notices.length" class="py-4 text-sm text-ink-muted">Tidak ada receipt aktif.</p>
              <ul v-else class="max-h-80 space-y-1 overflow-y-auto">
                <li v-for="notice in notices" :key="notice.id"><button type="button" class="block w-full rounded-lg px-2 py-2 text-left hover:bg-default focus-visible:outline-2 focus-visible:outline-brand-500 disabled:cursor-wait disabled:opacity-60" :disabled="openingNoticeId === notice.id" @click="openNotice(notice)"><span class="block text-sm font-medium text-ink">{{ notice.kode }} · {{ notice.status }}</span><span class="text-xs text-ink-muted">{{ notice.department_nama }} · {{ notice.days_waiting >= 3 ? `${notice.days_waiting} hari, tertunda` : `${notice.days_waiting} hari` }}</span><span v-if="openingNoticeId === notice.id" class="sr-only">Menandai dibaca</span></button></li>
              </ul>
            </section>
          </div>
          <button
            type="button"
            class="flex items-center gap-1.5 rounded-xl border border-default bg-surface/70 px-2.5 py-1.5 text-xs font-semibold text-ink shadow-2xs backdrop-blur-xs transition-all hover:bg-default hover:shadow-xs focus-visible:outline-2 focus-visible:outline-brand-500"
            :aria-label="theme.theme === 'dark' ? 'Mode terang' : 'Mode gelap'"
            @click="theme.toggle()"
          >
            <Sun v-if="theme.theme === 'dark'" class="size-4 text-amber-400 transition-transform hover:rotate-45" aria-hidden="true" />
            <Moon v-else class="size-4 text-brand-600 transition-transform hover:-rotate-12" aria-hidden="true" />
            <span class="hidden sm:inline">{{ theme.theme === 'dark' ? 'Mode Terang' : 'Mode Gelap' }}</span>
          </button>

          <!-- User menu -->
          <button
            type="button"
            class="flex items-center gap-2 rounded-lg px-2 py-1.5 text-sm font-medium text-ink transition-colors hover:bg-default"
            @click="toggleUserMenu"
          >
            <span class="flex size-7 items-center justify-center rounded-full bg-brand-100 text-xs font-semibold text-brand-700 dark:bg-brand-950 dark:text-brand-300">
              {{ initials }}
            </span>
            <span class="hidden sm:inline">{{ auth.user?.username }}</span>
            <span class="rounded-md bg-default px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-ink-muted">
              {{ auth.user?.role }}
            </span>
          </button>

          <div
            v-if="userMenuOpen"
            class="absolute top-full right-0 mt-2 w-56 rounded-xl border border-default bg-raised shadow-lg z-50"
          >
            <div class="px-4 py-3 border-b border-default">
              <p class="text-sm font-medium text-ink truncate">{{ auth.user?.full_name || auth.user?.username }}</p>
              <p class="text-xs text-ink-muted truncate">{{ auth.user ? roleLabel[auth.user.role] : '' }}</p>
            </div>
            <div class="py-1">
              <button
                type="button"
                class="flex w-full items-center gap-3 px-4 py-2 text-sm text-ink-muted hover:bg-default hover:text-ink"
                @click="theme.toggle()"
              >
                <Moon v-if="theme.theme === 'dark'" class="size-4" aria-hidden="true" />
                <Sun v-else class="size-4" aria-hidden="true" />
                {{ theme.theme === 'dark' ? 'Mode terang' : 'Mode gelap' }}
              </button>
              <RouterLink
                to="/profile"
                class="flex w-full items-center gap-3 px-4 py-2 text-sm text-ink-muted transition-colors hover:bg-default hover:text-ink focus-visible:outline-2 focus-visible:outline-brand-500"
              >
                <Settings class="size-4" aria-hidden="true" />
                Profil saya
              </RouterLink>
              <button
                type="button"
                class="flex w-full items-center gap-3 px-4 py-2 text-sm text-status-cancelled hover:bg-status-cancelled/10"
                @click="handleLogout"
              >
                <LogOut class="size-4" aria-hidden="true" />
                Keluar
              </button>
            </div>
          </div>
        </div>
      </header>

      <main class="p-4 sm:p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>
