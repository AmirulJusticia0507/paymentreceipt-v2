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
import {
  Building2,
  FileText,
  Layers,
  Moon,
  Receipt,
  RefreshCw,
  Sun,
  TrendingUp,
} from 'lucide-vue-next'
import Badge from '@/components/ui/Badge.vue'
import { useThemeStore } from '@/stores/theme'
import { requestWithMeta } from '@/lib/http'

ChartJS.register(ArcElement, BarElement, CategoryScale, LinearScale, Legend, LineElement, PointElement, Tooltip)

const theme = useThemeStore()

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

function formatRupiah(val: number | string | null | undefined): string {
  const n = Number(val) || 0
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)
}

const spendChartData = computed(() => ({
  labels: monthlySpend.value.map((m: any) => m.label),
  datasets: [
    {
      label: 'Total belanja (Rp)',
      data: monthlySpend.value.map((m: any) => Number(m.spend) || 0),
      backgroundColor: 'rgba(59, 130, 246, 0.15)',
      borderColor: 'rgb(59, 130, 246)',
      borderWidth: 2,
      pointBackgroundColor: 'rgb(59, 130, 246)',
      pointBorderColor: '#ffffff',
      pointHoverRadius: 6,
      tension: 0.35,
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
      borderWidth: 2,
      borderColor: theme.theme === 'dark' ? '#1e293b' : '#ffffff',
    },
  ],
}))

const topDeptChartData = computed(() => ({
  labels: topDepartments.value.map((d: any) => d.nama),
  datasets: [
    {
      label: 'Total belanja',
      data: topDepartments.value.map((d: any) => Number(d.total) || 0),
      backgroundColor: 'rgba(139, 92, 246, 0.75)',
      borderRadius: 6,
    },
  ],
}))
</script>

<template>
  <div class="space-y-6">
    <!-- Header Dashboard Modern dengan Tombol Dark Mode & Refresh -->
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-xl font-bold tracking-tight text-ink sm:text-2xl">Dashboard</h1>
        <p class="text-xs text-ink-muted sm:text-sm">
          Ringkasan aktivitas pengajuan & verifikasi pembelian PT. Asi Pudjiastuti Aviation
        </p>
      </div>

      <!-- Quick Action Toolbar -->
      <div class="flex items-center gap-2">
        <!-- Tombol Dark Mode Khusus Dashboard -->
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-xl border border-default bg-raised px-3.5 py-2 text-xs font-semibold text-ink shadow-2xs transition-all hover:bg-default hover:shadow-xs focus-visible:outline-2 focus-visible:outline-brand-500"
          :title="theme.theme === 'dark' ? 'Ganti ke Mode Terang' : 'Ganti ke Mode Gelap'"
          @click="theme.toggle()"
        >
          <Sun v-if="theme.theme === 'dark'" class="size-4 text-amber-400 transition-transform hover:rotate-45" />
          <Moon v-else class="size-4 text-brand-600 transition-transform hover:-rotate-12" />
          <span>{{ theme.theme === 'dark' ? 'Mode Terang' : 'Mode Gelap' }}</span>
        </button>

        <!-- Tombol Segarkan -->
        <button
          type="button"
          :disabled="loading"
          class="inline-flex items-center gap-2 rounded-xl border border-default bg-raised px-3.5 py-2 text-xs font-semibold text-ink shadow-2xs transition-all hover:bg-default hover:shadow-xs focus-visible:outline-2 focus-visible:outline-brand-500 disabled:opacity-50"
          @click="load()"
        >
          <RefreshCw class="size-3.5 text-brand-600 dark:text-brand-400" :class="{ 'animate-spin': loading }" />
          <span>{{ loading ? 'Memuat...' : 'Segarkan' }}</span>
        </button>
      </div>
    </div>

    <!-- Skeleton Loading -->
    <div v-if="loading">
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div v-for="i in 4" :key="i" class="card animate-pulse rounded-2xl p-5 shadow-sm">
          <div class="mb-3 h-4 w-1/2 rounded bg-default"></div>
          <div class="h-8 w-2/3 rounded bg-default"></div>
        </div>
      </div>
    </div>

    <!-- Error State -->
    <div
      v-else-if="error"
      class="rounded-2xl border border-status-cancelled/30 bg-status-cancelled/10 p-5 text-sm text-status-cancelled shadow-xs"
    >
      <p class="font-semibold">Gagal memuat data dashboard:</p>
      <p class="mt-1 text-xs opacity-90">{{ error }}</p>
      <button
        type="button"
        class="mt-3 rounded-lg bg-status-cancelled/20 px-3 py-1.5 text-xs font-medium hover:bg-status-cancelled/30"
        @click="load()"
      >
        Coba Lagi
      </button>
    </div>

    <!-- Dashboard Content -->
    <template v-else>
      <!-- Kartu Statistik Modern -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <!-- Card 1: Total Request -->
        <div
          class="card group relative overflow-hidden rounded-2xl border border-default bg-raised p-5 shadow-xs transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md"
        >
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-ink-muted">Total Request</span>
            <div class="flex size-9 items-center justify-center rounded-xl bg-blue-50 text-blue-600 transition-colors group-hover:bg-blue-600 group-hover:text-white dark:bg-blue-950/60 dark:text-blue-400">
              <FileText class="size-4.5" />
            </div>
          </div>
          <p class="mt-3 text-3xl font-extrabold tracking-tight text-ink tabular-nums">
            {{ summary?.total_requests ?? 0 }}
          </p>
          <p class="mt-1 text-[11px] text-ink-muted">Pengajuan departemen</p>
        </div>

        <!-- Card 2: Total Payment Receipt -->
        <div
          class="card group relative overflow-hidden rounded-2xl border border-default bg-raised p-5 shadow-xs transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md"
        >
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-ink-muted">Payment Receipt</span>
            <div class="flex size-9 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600 transition-colors group-hover:bg-emerald-600 group-hover:text-white dark:bg-emerald-950/60 dark:text-emerald-400">
              <Receipt class="size-4.5" />
            </div>
          </div>
          <p class="mt-3 text-3xl font-extrabold tracking-tight text-ink tabular-nums">
            {{ summary?.total_payment_receipts ?? 0 }}
          </p>
          <p class="mt-1 text-[11px] text-ink-muted">Transaksi pembelian</p>
        </div>

        <!-- Card 3: Total Base -->
        <div
          class="card group relative overflow-hidden rounded-2xl border border-default bg-raised p-5 shadow-xs transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md"
        >
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-ink-muted">Total Base</span>
            <div class="flex size-9 items-center justify-center rounded-xl bg-purple-50 text-purple-600 transition-colors group-hover:bg-purple-600 group-hover:text-white dark:bg-purple-950/60 dark:text-purple-400">
              <Layers class="size-4.5" />
            </div>
          </div>
          <p class="mt-3 text-3xl font-extrabold tracking-tight text-ink tabular-nums">
            {{ summary?.total_bases ?? 0 }}
          </p>
          <p class="mt-1 text-[11px] text-ink-muted">Kategori pengajuan</p>
        </div>

        <!-- Card 4: Total Department -->
        <div
          class="card group relative overflow-hidden rounded-2xl border border-default bg-raised p-5 shadow-xs transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md"
        >
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-ink-muted">Departemen</span>
            <div class="flex size-9 items-center justify-center rounded-xl bg-amber-50 text-amber-600 transition-colors group-hover:bg-amber-600 group-hover:text-white dark:bg-amber-950/60 dark:text-amber-400">
              <Building2 class="size-4.5" />
            </div>
          </div>
          <p class="mt-3 text-3xl font-extrabold tracking-tight text-ink tabular-nums">
            {{ summary?.total_departments ?? 0 }}
          </p>
          <p class="mt-1 text-[11px] text-ink-muted">Unit kerja aktif</p>
        </div>
      </div>

      <!-- Charts Section Modern -->
      <div class="grid grid-cols-1 gap-5 lg:grid-cols-2">
        <!-- Monthly Spend Trend -->
        <div class="card rounded-2xl border border-default bg-raised p-5 shadow-xs">
          <div class="mb-4 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <TrendingUp class="size-4 text-brand-600 dark:text-brand-400" />
              <h2 class="text-sm font-bold text-ink">Trend Belanja Bulanan</h2>
            </div>
            <span class="text-[11px] font-medium text-ink-muted">12 bulan terakhir</span>
          </div>
          <Line
            :data="spendChartData"
            :options="{
              responsive: true,
              plugins: { legend: { display: false } },
              scales: {
                y: { grid: { color: theme.theme === 'dark' ? '#334155' : '#f1f5f9' } },
                x: { grid: { display: false } },
              },
            }"
          />
        </div>

        <!-- Status Breakdown -->
        <div class="card rounded-2xl border border-default bg-raised p-5 shadow-xs">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-sm font-bold text-ink">Status Breakdown</h2>
            <span class="text-[11px] font-medium text-ink-muted">Distribusi status</span>
          </div>
          <div class="mx-auto max-w-xs py-2">
            <Doughnut :data="statusChartData" :options="{ responsive: true, plugins: { legend: { position: 'bottom' } } }" />
          </div>
        </div>

        <!-- Top Department Spend -->
        <div class="card rounded-2xl border border-default bg-raised p-5 shadow-xs lg:col-span-2">
          <div class="mb-4 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Building2 class="size-4 text-purple-600 dark:text-purple-400" />
              <h2 class="text-sm font-bold text-ink">Top Departemen Berdasarkan Total Belanja</h2>
            </div>
          </div>
          <Bar
            :data="topDeptChartData"
            :options="{
              indexAxis: 'y' as const,
              responsive: true,
              plugins: { legend: { display: false } },
              scales: {
                x: { grid: { color: theme.theme === 'dark' ? '#334155' : '#f1f5f9' } },
                y: { grid: { display: false } },
              },
            }"
          />
        </div>
      </div>

      <!-- 10 Transaksi Terakhir Modern -->
      <div class="card overflow-hidden rounded-2xl border border-default bg-raised shadow-xs">
        <div class="border-b border-default px-5 py-4">
          <div class="flex items-center justify-between">
            <h2 class="text-sm font-bold text-ink">10 Transaksi Terakhir</h2>
            <RouterLink to="/payment-receipts" class="text-xs font-semibold text-brand-600 hover:underline dark:text-brand-400">
              Lihat Semua Transaksi &rarr;
            </RouterLink>
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full border-collapse text-left text-sm">
            <thead>
              <tr class="border-b border-default bg-surface/50 text-[11px] font-semibold uppercase tracking-wider text-ink-muted">
                <th class="px-5 py-3">Kode</th>
                <th class="px-5 py-3">Departemen</th>
                <th class="px-5 py-3">Deskripsi Item</th>
                <th class="px-5 py-3">Status</th>
                <th class="px-5 py-3 text-right">Total</th>
                <th class="px-5 py-3">Tanggal</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-default/60">
              <tr
                v-for="r in recent"
                :key="r.id"
                class="transition-colors hover:bg-default/40"
              >
                <td class="px-5 py-3.5 font-mono text-xs font-semibold text-brand-600 dark:text-brand-400">
                  {{ r.kode }}
                </td>
                <td class="px-5 py-3.5 font-medium text-ink">
                  {{ r.department_nama }}
                </td>
                <td class="max-w-xs truncate px-5 py-3.5 text-ink-muted">
                  {{ r.deskripsi_items || '-' }}
                </td>
                <td class="px-5 py-3.5">
                  <Badge :variant="r.status">{{ r.status }}</Badge>
                </td>
                <td class="px-5 py-3.5 text-right font-medium tabular-nums text-ink">
                  {{ formatRupiah(r.total_harga) }}
                </td>
                <td class="px-5 py-3.5 text-xs text-ink-muted whitespace-nowrap">
                  {{ r.tanggal_pembelian ? new Date(r.tanggal_pembelian).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }) : '-' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="recent.length === 0" class="py-10 text-center text-sm text-ink-muted">
          Belum ada transaksi pembelian tercatat.
        </div>
      </div>
    </template>
  </div>
</template>
