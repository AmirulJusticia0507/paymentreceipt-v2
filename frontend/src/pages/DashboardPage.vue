<script setup lang="ts">
/**
 * Dashboard sementara.
 *
 * Angka di sini sengaja diambil langsung dari database lewat query agregat
 * pada T9.x, bukan dari endpoint dashboard yang belum ada. Yang penting
 * sekarang layout, tema, dan shell aplikasinya sudah benar sebelum modul
 * bisnis masuk.
 */
import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { FileText, FolderTree, Package, TrendingUp } from 'lucide-vue-next'

import type { PageMeta } from '@/lib/types'

// Query-nya di-disable sampai endpoint /dashboard benar-benar ada (Phase 9).
// Struktur placeholder ini supaya kerangkanya sudah benar sebelum angka
// asli berasal dari server.
const { data: meta } = useQuery<PageMeta | null>({
  queryKey: ['dashboard-placeholder'],
  queryFn: async () => null,
  enabled: false,
})

const stats = computed(() => [
  { label: 'Total Requests', value: meta.value?.total ?? '-', icon: FolderTree, accent: 'text-brand-600' },
  { label: 'Payment Receipt', value: '-', icon: FileText, accent: 'text-status-processed' },
  { label: 'Department', value: '-', icon: Package, accent: 'text-status-finished' },
])
</script>

<template>
  <div class="space-y-6">
    <!-- Kartu statistik -->
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="card p-5 shadow-sm transition-[var(--transition-fast)] hover:shadow-md"
      >
        <div class="flex items-start justify-between">
          <div>
            <p class="text-sm font-medium text-ink-muted">{{ stat.label }}</p>
            <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">{{ stat.value }}</p>
          </div>
          <div class="rounded-lg bg-default p-2">
            <component :is="stat.icon" class="size-5" :class="stat.accent" aria-hidden="true" />
          </div>
        </div>
      </div>
    </div>

    <!-- Placeholder chart -->
    <div class="card p-6">
      <div class="mb-4 flex items-center gap-2">
        <TrendingUp class="size-5 text-ink-muted" aria-hidden="true" />
        <h2 class="text-sm font-semibold text-ink">Tren Pengeluaran</h2>
      </div>
      <div
        class="flex h-48 items-center justify-center rounded-lg border border-dashed border-default text-sm text-ink-muted"
      >
        Chart akan tersedia setelah endpoint dashboard selesai (T9.7)
      </div>
    </div>

    <!-- Empty state -->
    <div class="card p-6">
      <h2 class="text-sm font-semibold text-ink">Transaksi Terbaru</h2>
      <div
        class="mt-4 flex h-32 items-center justify-center rounded-lg border border-dashed border-default text-sm text-ink-muted"
      >
        Belum ada data
      </div>
    </div>
  </div>
</template>