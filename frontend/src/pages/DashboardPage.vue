<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Bar, Doughnut, Line } from 'vue-chartjs'
import {
  ArcElement,
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
} from 'chart.js'
import Badge from '@/components/ui/Badge.vue'
import { requestWithMeta } from '@/lib/http'

ChartJS.register(ArcElement, BarElement, CategoryScale, LinearScale, Legend, LineElement, PointElement, Tooltip)

const summary = ref<any>(null)
const recent = ref<any[]>([])
const statusBreakdown = ref<any[]>([])
const monthlySpend = ref<any[]>([])
const topDepartments = ref<any[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

async function load(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const [s, r, b, m, t] = await Promise.all([
      requestWithMeta<any>('/api/v1/dashboard/summary'),
      requestWithMeta<any[]>('/api/v1/dashboard/recent'),
      requestWithMeta<any[]>('/api/v1/dashboard/status-breakdown'),
      requestWithMeta<any[]>('/api/v1/dashboard/monthly-spend'),
      requestWithMeta<any[]>('/api/v1/dashboard/top-departments'),
    ])
    summary.value = s.data
    recent.value = r.data ?? []
    statusBreakdown.value = b.data ?? []
    monthlySpend.value = m.data ?? []
    topDepartments.value = t.data ?? []
  } catch (e: any) {
    error.value = e?.message ?? 'Gagal memuat dashboard'
  } finally {
    loading.value = false
  }
}

onMounted(load)

const spendChartData = computed(() => ({
  labels: monthlySpend.value.map((m: any) => m.label),
  datasets: [
    {
      label: 'Total belanja (Rp)',
      data: monthlySpend.value.map((m: any) => Number(m.spend) || 0),
      backgroundColor: 'rgba(59, 130, 246, 0.5)',
      borderColor: 'rgb(59, 130, 246)',
      tension: 0.3,
      fill: true,
    },
  ],
}))

const statusChartData = computed(() => ({
  labels: statusBreakdown.value.map((s: any) => s.status),
  datasets: [
    {
      data: statusBreakdown.value.map((s: any) => s.count),
      backgroundColor: ['#facc15', '#3b82f6', '#8b5cf6', '#22c55e', '#ef4444'],
    },
  ],
}))

const topDeptChartData = computed(() => ({
  labels: topDepartments.value.map((d: any) => d.nama),
  datasets: [
    {
      label: 'Total belanja',
      data: topDepartments.value.map((d: any) => Number(d.total) || 0),
      backgroundColor: 'rgba(139, 92, 246, 0.6)',
    },
  ],
}))
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold tracking-tight text-ink">Dashboard</h1>
      <button class="text-sm text-brand-600 hover:underline" @click="load()">Muat ulang</button>
    </div>

    <div v-if="loading">
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div v-for="i in 4" :key="i" class="card p-4 animate-pulse">
          <div class="h-4 w-1/2 bg-default rounded mb-2"></div>
          <div class="h-6 w-1/3 bg-default rounded"></div>
        </div>
      </div>
    </div>

    <div v-else-if="error" class="card p-4 text-sm text-status-cancelled">{{ error }}</div>

    <template v-else>
      <!-- Kartu statistik -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="card p-4">
          <p class="text-xs text-ink-muted">Total Request</p>
          <p class="text-2xl font-bold text-ink tabular-nums">{{ summary?.total_requests ?? 0 }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-ink-muted">Total Payment Receipt</p>
          <p class="text-2xl font-bold text-ink tabular-nums">{{ summary?.total_payment_receipts ?? 0 }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-ink-muted">Total Base</p>
          <p class="text-2xl font-bold text-ink tabular-nums">{{ summary?.total_bases ?? 0 }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-ink-muted">Total Department</p>
          <p class="text-2xl font-bold text-ink tabular-nums">{{ summary?.total_departments ?? 0 }}</p>
        </div>
      </div>

      <!-- Charts -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div class="card p-4">
          <h2 class="text-sm font-semibold text-ink mb-3">Trend Belanja Bulanan</h2>
          <Line :data="spendChartData" :options="{ responsive: true, plugins: { legend: { display: false } } }" />
        </div>

        <div class="card p-4">
          <h2 class="text-sm font-semibold text-ink mb-3">Status Breakdown</h2>
          <div class="max-w-xs mx-auto">
            <Doughnut :data="statusChartData" :options="{ responsive: true }" />
          </div>
        </div>

        <div class="card p-4">
          <h2 class="text-sm font-semibold text-ink mb-3">Top Department (Total Belanja)</h2>
          <Bar :data="topDeptChartData" :options="{ indexAxis: 'y' as const, responsive: true, plugins: { legend: { display: false } } }" />
        </div>
      </div>

      <!-- Recent transactions -->
      <div class="card p-4">
        <h2 class="text-sm font-semibold text-ink mb-3">10 Transaksi Terakhir</h2>
        <table class="w-full border-collapse text-sm">
          <thead>
            <tr class="border-b border-default">
              <th class="px-3 py-2 text-left text-xs font-semibold uppercase text-ink-muted">Kode</th>
              <th class="px-3 py-2 text-left text-xs font-semibold uppercase text-ink-muted">Departemen</th>
              <th class="px-3 py-2 text-left text-xs font-semibold uppercase text-ink-muted">Deskripsi</th>
              <th class="px-3 py-2 text-left text-xs font-semibold uppercase text-ink-muted">Status</th>
              <th class="px-3 py-2 text-right text-xs font-semibold uppercase text-ink-muted">Total</th>
              <th class="px-3 py-2 text-left text-xs font-semibold uppercase text-ink-muted">Tanggal</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in recent" :key="r.id" class="border-b border-default/60 last:border-b-0">
              <td class="px-3 py-2 font-mono text-xs font-medium text-ink">{{ r.kode }}</td>
              <td class="px-3 py-2 text-ink-muted">{{ r.department_nama }}</td>
              <td class="px-3 py-2 text-ink truncate max-w-xs">{{ r.deskripsi_items }}</td>
              <td class="px-3 py-2"><Badge :variant="r.status">{{ r.status }}</Badge></td>
              <td class="px-3 py-2 text-right tabular-nums text-ink">{{ r.total_harga }}</td>
              <td class="px-3 py-2 text-xs text-ink-muted">{{ r.tanggal_pembelian ? new Date(r.tanggal_pembelian).toLocaleDateString('id-ID') : '-' }}</td>
            </tr>
          </tbody>
        </table>
        <p v-if="recent.length === 0" class="text-sm text-ink-muted text-center py-6">Belum ada transaksi.</p>
      </div>
    </template>
  </div>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: all var(--transition-fast); }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateY(0.5rem); }
</style>