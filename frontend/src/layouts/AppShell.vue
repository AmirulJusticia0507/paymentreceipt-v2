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
  BarChart3,
  FileText,
  FolderTree,
  LayoutDashboard,
  LogOut,
  Menu,
  Moon,
  Package,
  Receipt,
  Sun,
  Tags,
  Truck,
  X,
} from 'lucide-vue-next'

import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import type { UserRole } from '@/lib/types'

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
      { label: 'Payment Receipt', to: '/payment-receipt', icon: Receipt, disabled: true },
      { label: 'Arsip PR', to: '/pr-archives', icon: FileText },
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
]

const auth = useAuthStore()
const theme = useThemeStore()
const route = useRoute()
const router = useRouter()

const sidebarOpen = ref(false)

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
      :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full'"
      aria-label="Navigasi utama"
    >
      <!-- Brand -->
      <div class="flex h-16 shrink-0 items-center gap-2.5 border-b border-default px-5">
        <div class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-brand-600 text-white">
          <BarChart3 class="size-4" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <p class="truncate text-sm font-semibold text-ink">Payment Receipt</p>
          <p class="truncate text-xs text-ink-muted">General Affairs</p>
        </div>
        <button
          type="button"
          class="ml-auto rounded-md p-1 text-ink-muted hover:text-ink lg:hidden"
          aria-label="Tutup menu"
          @click="sidebarOpen = false"
        >
          <X class="size-5" aria-hidden="true" />
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
                {{ item.label }}
              </RouterLink>
              <div
                v-else
                class="flex cursor-not-allowed items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-ink-muted/50"
                :title="`${item.label} - sedang dibangun`"
              >
                <component :is="item.icon" class="size-4 shrink-0" aria-hidden="true" />
                {{ item.label }}
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

        <h1 class="truncate text-base font-semibold text-ink">
          {{ route.meta.title ?? 'Dashboard' }}
        </h1>

        <button
          type="button"
          class="ml-auto rounded-md p-2 text-ink-muted transition-colors hover:bg-default hover:text-ink"
          :aria-label="theme.theme === 'dark' ? 'Mode terang' : 'Mode gelap'"
          @click="theme.toggle()"
        >
          <Sun v-if="theme.theme === 'dark'" class="size-5" aria-hidden="true" />
          <Moon v-else class="size-5" aria-hidden="true" />
        </button>
      </header>

      <main class="p-4 sm:p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>
