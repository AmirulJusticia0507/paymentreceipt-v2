<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { requestWithMeta, qs } from '@/lib/http'

interface ReportRow {
  department_id: number
  department_kode: string
  department_nama: string
  status: string
  receipt_count: number
  total_amount: string
}

interface Department { id: number; kode: string; nama: string }

function localDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = new Date()
const dateFrom = ref(localDate(new Date(today.getFullYear(), today.getMonth(), 1)))
const dateTo = ref(localDate(today))
const departmentId = ref('')
const rows = ref<ReportRow[]>([])
const departments = ref<Department[]>([])
const loading = ref(false)
const initialLoading = ref(true)
const error = ref('')

const receiptCount = computed(() => rows.value.reduce((sum, row) => sum + row.receipt_count, 0))
const totalAmount = computed(() => rows.value.reduce((sum, row) => sum + Number(row.total_amount || 0), 0))

function formatRupiah(value: number | string): string {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0)
}

async function loadDepartments(): Promise<void> {
  try {
    const result = await requestWithMeta<Department[]>('/api/v1/departments?per_page=100')
    departments.value = result.data ?? []
  } catch {
    departments.value = []
  }
}

async function loadReport(): Promise<void> {
  if (!dateFrom.value || !dateTo.value || dateFrom.value > dateTo.value) {
    error.value = 'Pilih rentang tanggal yang valid.'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const params = qs({ date_from: dateFrom.value, date_to: dateTo.value, department_id: departmentId.value || undefined })
    const result = await requestWithMeta<ReportRow[]>(`/api/v1/reports/operational${params}`)
    rows.value = result.data ?? []
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Gagal memuat laporan.'
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadDepartments(), loadReport()])
})
</script>

<template>
  <section class="space-y-5">
    <header>
      <h1 class="text-xl font-bold tracking-tight text-ink sm:text-2xl">Laporan Operasional</h1>
      <p class="mt-1 text-sm text-ink-muted">Ringkasan jumlah receipt dan nilai transaksi menurut department dan status.</p>
    </header>

    <form class="card grid gap-3 rounded-xl p-4 sm:grid-cols-2 lg:grid-cols-[1fr_1fr_1.2fr_auto] lg:items-end" @submit.prevent="loadReport()">
      <label class="grid gap-1 text-sm text-ink">
        <span>Tanggal mulai</span>
        <input v-model="dateFrom" type="date" required class="rounded-lg border border-default bg-raised px-3 py-2 focus-visible:outline-2 focus-visible:outline-brand-500" />
      </label>
      <label class="grid gap-1 text-sm text-ink">
        <span>Tanggal akhir</span>
        <input v-model="dateTo" type="date" required class="rounded-lg border border-default bg-raised px-3 py-2 focus-visible:outline-2 focus-visible:outline-brand-500" />
      </label>
      <label class="grid gap-1 text-sm text-ink">
        <span>Department</span>
        <select v-model="departmentId" class="rounded-lg border border-default bg-raised px-3 py-2 focus-visible:outline-2 focus-visible:outline-brand-500">
          <option value="">Semua department</option>
          <option v-for="department in departments" :key="department.id" :value="String(department.id)">{{ department.kode }} — {{ department.nama }}</option>
        </select>
      </label>
      <button type="submit" :disabled="loading" class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600 disabled:cursor-not-allowed disabled:opacity-60">
        {{ loading ? 'Memuat…' : 'Tampilkan' }}
      </button>
    </form>

    <p v-if="error" role="alert" class="rounded-lg border border-status-cancelled/30 bg-status-cancelled/10 p-3 text-sm text-status-cancelled">{{ error }}</p>

    <div v-if="initialLoading" class="grid gap-3 sm:grid-cols-2">
      <div v-for="n in 2" :key="n" class="card h-24 animate-pulse rounded-xl bg-default" />
    </div>
    <template v-else>
      <div class="grid gap-3 sm:grid-cols-2">
        <article class="card rounded-xl p-4">
          <p class="text-sm text-ink-muted">Jumlah receipt</p>
          <p class="mt-1 text-2xl font-semibold text-ink">{{ receiptCount.toLocaleString('id-ID') }}</p>
        </article>
        <article class="card rounded-xl p-4">
          <p class="text-sm text-ink-muted">Total nilai transaksi</p>
          <p class="mt-1 text-2xl font-semibold text-ink">{{ formatRupiah(totalAmount) }}</p>
        </article>
      </div>

      <div v-if="!loading && rows.length === 0" class="card rounded-xl p-8 text-center text-sm text-ink-muted">
        Tidak ada receipt pada filter yang dipilih.
      </div>
      <div v-else class="card overflow-hidden rounded-xl">
        <ul class="divide-y divide-default lg:hidden">
          <li v-for="(row, index) in rows" :key="`mobile-${row.department_id}-${row.status}-${index}`" class="space-y-2 p-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="truncate font-semibold text-ink">{{ row.department_nama }}</p>
                <p class="text-xs text-ink-muted">{{ row.department_kode }}</p>
              </div>
              <span class="shrink-0 rounded-full bg-default px-2.5 py-1 text-xs font-medium capitalize text-ink">{{ row.status }}</span>
            </div>
            <dl class="grid grid-cols-2 gap-3 text-sm">
              <div><dt class="text-xs text-ink-muted">Receipt</dt><dd class="mt-0.5 font-medium tabular-nums text-ink">{{ row.receipt_count }}</dd></div>
              <div><dt class="text-xs text-ink-muted">Total transaksi</dt><dd class="mt-0.5 font-medium tabular-nums text-ink">{{ formatRupiah(row.total_amount) }}</dd></div>
            </dl>
          </li>
        </ul>
        <div class="overflow-x-auto">
          <table class="hidden w-full min-w-[700px] text-left text-sm lg:table">
            <thead class="border-b border-default bg-raised text-xs uppercase tracking-wide text-ink-muted">
              <tr><th class="px-4 py-3">Department</th><th class="px-4 py-3">Status</th><th class="px-4 py-3 text-right">Receipt</th><th class="px-4 py-3 text-right">Total transaksi</th></tr>
            </thead>
            <tbody class="divide-y divide-default">
              <tr v-for="(row, index) in rows" :key="`${row.department_id}-${row.status}-${index}`" class="hover:bg-raised/60">
                <td class="px-4 py-3"><span class="font-medium text-ink">{{ row.department_nama }}</span><span class="ml-2 text-xs text-ink-muted">{{ row.department_kode }}</span></td>
                <td class="px-4 py-3 capitalize text-ink">{{ row.status }}</td>
                <td class="px-4 py-3 text-right tabular-nums text-ink">{{ row.receipt_count }}</td>
                <td class="px-4 py-3 text-right tabular-nums text-ink">{{ formatRupiah(row.total_amount) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </section>
</template>
