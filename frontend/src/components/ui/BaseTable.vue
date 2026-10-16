<script setup lang="ts" generic="T extends { id: number | string }">
/**
 * Tabel data generik dengan paginasi server-side.
 *
 * Untuk T generic, `cell` pada Column<T> dikembalikan apa adanya. Kalau kolom
 * butuh tampilan kaya (badge, format rupiah), pakai slot #cell-<key> -
 * penyusun halaman lebih punya kontrol daripada logika di dalam tabel.
 *
 * Tabel ini SENGAJA tidak memuat data sendiri. Ia menerima rows, loading,
 * dan error, lalu merender. Pengambilan data tetap di feature masing-masing
 * lewat TanStack Query, supaya tabel tidak ikut tahu soal HTTP, cache, atau
 * invalidate.
 */
import { computed, ref } from 'vue'
import { AlertCircle, ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Inbox } from 'lucide-vue-next'

import BaseButton from './BaseButton.vue'
import SkeletonTable from './SkeletonTable.vue'
import EmptyState from './EmptyState.vue'
import type { Column, PageMeta, TableState } from './tableTypes'

const props = defineProps<{
  columns: Column<T>[]
  rows: T[]
  meta?: PageMeta
  state: TableState
  errorMessage?: string | null
  rowKey?: (row: T) => string | number
  /** Slot fallback untuk baris, dipakai tabel dengan tampilan khusus. */
  clickable?: boolean
}>()

const emit = defineEmits<{
  'update:page': [page: number]
  'update:sort': [sort: string]
  retry: []
}>()

const sort = ref('')

/** Tombol mana yang sedang aktif untuk sebuah kolom. */
function sortDirection(key: string): 'asc' | 'desc' | null {
  if (!sort.value) return null
  if (sort.value === key) return 'asc'
  if (sort.value === `-${key}`) return 'desc'
  return null
}

function toggleSort(column: Column<T>): void {
  if (!column.sortable) return

  const current = sortDirection(column.key)
  // Siklus asc -> desc -> lepas. Melepas penting supaya user bisa kembali ke
  // urutan default tanpa harus menekan tiga kali.
  let next: string
  if (current === null) next = column.key
  else if (current === 'asc') next = `-${column.key}`
  else next = ''

  sort.value = next
  emit('update:sort', next)
}

const alignClass = (align: Column<T>['align']): string =>
  align === 'right' ? 'text-right' : align === 'center' ? 'text-center' : 'text-left'

const canPrev = computed(() => (props.meta?.page ?? 1) > 1)
const canNext = computed(() => {
  if (!props.meta) return false
  return props.meta.page < props.meta.total_pages
})

const pageRange = computed(() => {
  if (!props.meta || props.meta.total === 0) return '0 data'
  const from = (props.meta.page - 1) * props.meta.per_page + 1
  const to = Math.min(props.meta.page * props.meta.per_page, props.meta.total)
  return `${from}-${to} dari ${props.meta.total}`
})

function keyFor(row: T, index: number): string | number {
  return props.rowKey ? props.rowKey(row) : (row as { id: string | number }).id ?? index
}
</script>

<template>
  <div class="card overflow-hidden shadow-sm">
    <!-- Error -->
    <div
      v-if="state === 'error'"
      class="flex flex-col items-center gap-3 px-6 py-12 text-center"
    >
      <AlertCircle class="size-8 text-status-cancelled" aria-hidden="true" />
      <div>
        <p class="text-sm font-medium text-ink">Gagal memuat data</p>
        <p class="mt-1 text-sm text-ink-muted">
          {{ errorMessage ?? 'Terjadi kesalahan yang tidak diketahui.' }}
        </p>
      </div>
      <BaseButton variant="secondary" size="sm" @click="emit('retry')">Coba lagi</BaseButton>
    </div>

    <!-- Tabel -->
    <div v-else class="relative">
      <!-- Loading: skeleton menutupi tabel, bukan spinner di tengah.
           Layout tidak bergeser saat data datang, jadi halaman terasa lebih tenang. -->
      <SkeletonTable v-if="state === 'loading'" :rows="6" :columns="columns.length" />

      <template v-else>
        <div v-if="rows.length === 0" class="px-6 py-12">
          <EmptyState>
            <template #icon><Inbox class="size-8" aria-hidden="true" /></template>
            <slot name="empty">
              <p class="text-sm font-medium text-ink">Belum ada data</p>
              <p class="mt-1 text-sm text-ink-muted">
                Data yang dibuat akan muncul di sini.
              </p>
            </slot>
          </EmptyState>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="w-full border-collapse text-sm">
            <thead>
              <tr class="border-b border-default">
                <th
                  v-for="column in columns"
                  :key="column.key"
                  scope="col"
                  :class="[
                    alignClass(column.align),
                    'px-4 py-3 text-xs font-semibold uppercase tracking-wide text-ink-muted',
                    column.width ?? '',
                    column.hideOnMobile ? 'hidden sm:table-cell' : '',
                  ]"
                  :aria-sort="
                    sortDirection(column.key) === 'asc'
                      ? 'ascending'
                      : sortDirection(column.key) === 'desc'
                        ? 'descending'
                        : 'none'
                  "
                >
                  <button
                    v-if="column.sortable"
                    type="button"
                    class="inline-flex items-center gap-1 rounded transition-colors hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500"
                    @click="toggleSort(column)"
                  >
                    {{ column.label }}
                    <ChevronUp
                      v-if="sortDirection(column.key) === 'asc'"
                      class="size-3.5"
                      aria-hidden="true"
                    />
                    <ChevronDown
                      v-else-if="sortDirection(column.key) === 'desc'"
                      class="size-3.5"
                      aria-hidden="true"
                    />
                    <ChevronDown v-else class="size-3.5 opacity-30" aria-hidden="true" />
                  </button>
                  <span v-else>{{ column.label }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, index) in rows"
                :key="keyFor(row, index)"
                class="border-b border-default/60 last:border-0 transition-colors hover:bg-default/40"
                :class="clickable ? 'cursor-pointer' : ''"
                @click="clickable && emit('retry')"
              >
                <td
                  v-for="column in columns"
                  :key="column.key"
                  :class="[
                    alignClass(column.align),
                    'px-4 py-3 text-ink',
                    column.hideOnMobile ? 'hidden sm:table-cell' : '',
                  ]"
                >
                  <!-- Slot khusus kolom, misalnya badge status atau rupiah. -->
                  <slot :name="`cell-${column.key}`" :row="row" :value="(row as Record<string, unknown>)[column.key]">
                    {{ column.cell ? column.cell(row) : (row as Record<string, unknown>)[column.key] }}
                  </slot>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </div>

    <!-- Paginasi -->
    <div
      v-if="state !== 'error' && meta && meta.total > 0"
      class="flex flex-wrap items-center justify-between gap-3 border-t border-default px-4 py-3"
    >
      <p class="text-xs text-ink-muted">{{ pageRange }}</p>

      <div class="flex items-center gap-1">
        <BaseButton
          variant="ghost"
          size="sm"
          :disabled="!canPrev"
          aria-label="Halaman sebelumnya"
          @click="emit('update:page', meta.page - 1)"
        >
          <ChevronLeft class="size-4" aria-hidden="true" />
        </BaseButton>

        <span class="px-2 text-xs tabular-nums text-ink-muted">
          {{ meta.page }} / {{ meta.total_pages }}
        </span>

        <BaseButton
          variant="ghost"
          size="sm"
          :disabled="!canNext"
          aria-label="Halaman berikutnya"
          @click="emit('update:page', meta.page + 1)"
        >
          <ChevronRight class="size-4" aria-hidden="true" />
        </BaseButton>
      </div>
    </div>
  </div>
</template>