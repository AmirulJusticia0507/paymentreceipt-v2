<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import BaseTable from '@/components/ui/BaseTable.vue'
import type { Column, PageMeta, TableState } from '@/components/ui/tableTypes'
import { requestWithMeta, qs } from '@/lib/http'

interface AuditLog {
  id: number
  actor_id: number
  username: string
  full_name: string
  http_method: string
  request_path: string
  status_code: number
  request_id: string
  created_at: string
}

const rows = ref<AuditLog[]>([])
const page = ref(1)
const perPage = 25
const meta = ref<PageMeta>({ page: 1, per_page: perPage, total: 0, total_pages: 0 })
const loading = ref(true)
const error = ref('')
const state = computed<TableState>(() => error.value ? 'error' : loading.value ? 'loading' : 'ready')
const columns: Column<AuditLog>[] = [
  { key: 'created_at', label: 'Waktu', width: 'w-44' },
  { key: 'actor', label: 'Pengguna', width: 'w-40' },
  { key: 'http_method', label: 'Aksi', width: 'w-24' },
  { key: 'request_path', label: 'Endpoint' },
  { key: 'status_code', label: 'HTTP', align: 'right', width: 'w-20' },
  { key: 'request_id', label: 'Request ID', width: 'w-40', hideOnMobile: true },
]

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })
}

async function load(nextPage = page.value): Promise<void> {
  page.value = nextPage
  loading.value = true
  error.value = ''
  try {
    const result = await requestWithMeta<AuditLog[]>(`/api/v1/audit-logs/${qs({ page: page.value, per_page: perPage })}`)
    rows.value = result.data ?? []
    meta.value = result.meta as PageMeta
  } catch (cause) {
    rows.value = []
    error.value = cause instanceof Error ? cause.message : 'Gagal memuat audit log.'
  } finally {
    loading.value = false
  }
}

onMounted(() => load(1))
</script>

<template>
  <section class="space-y-4">
    <header>
      <h1 class="text-xl font-bold tracking-tight text-ink sm:text-2xl">Audit Log</h1>
      <p class="mt-1 text-sm text-ink-muted">Riwayat mutasi berhasil oleh pengguna terautentikasi. Isi request tidak disimpan.</p>
    </header>

    <BaseTable :columns="columns" :rows="rows" :meta="meta" :state="state" :error-message="error" @update:page="load" @retry="load()">
      <template #cell-created_at="{ row }"><span class="text-xs text-ink-muted">{{ formatDateTime(row.created_at) }}</span></template>
      <template #cell-actor="{ row }"><span class="font-medium text-ink">{{ row.full_name || row.username }}</span><span class="block text-xs text-ink-muted">{{ row.username }}</span></template>
      <template #cell-http_method="{ row }"><span class="rounded bg-raised px-2 py-1 font-mono text-xs text-ink">{{ row.http_method }}</span></template>
      <template #cell-request_path="{ row }"><code class="break-all text-xs text-ink">{{ row.request_path }}</code></template>
      <template #cell-status_code="{ row }"><span class="tabular-nums text-ink-muted">{{ row.status_code }}</span></template>
      <template #cell-request_id="{ row }"><code class="text-xs text-ink-muted">{{ row.request_id }}</code></template>
    </BaseTable>
  </section>
</template>
