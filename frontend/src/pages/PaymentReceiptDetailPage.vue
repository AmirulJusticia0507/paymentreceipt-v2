<script setup lang="ts">
/**
 * Halaman review & persetujuan Payment Receipt untuk Approver / GA Staff.
 *
 * Layout dua kolom:
 *  - Kiri  : embedded PDF viewer (iframe → /api/v1/payment-receipts/:id/pdf)
 *            dengan fallback tombol unduh jika browser memblokir iframe PDF.
 *  - Kanan : panel info + riwayat status + input catatan + tombol approve/tolak.
 *
 * Hanya user dengan role admin, ga_staff, atau approver yang melihat
 * tombol aksi — viewer hanya bisa membaca PDF.
 */
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  CheckCircle,
  Download,
  ExternalLink,
  Loader2,
  XCircle,
} from 'lucide-vue-next'

import Badge from '@/components/ui/Badge.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import { requestWithMeta } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'

// ─── types ─────────────────────────────────────────────────────────────────

interface Receipt {
  id: number
  kode: string
  status: string
  department_nama: string
  department_kode: string
  request_kode: string
  request_nama: string
  deskripsi_items: string
  jumlah_items: number
  harga_items: string
  total_harga: string
  tanggal_pembelian: string
  usage: string
  keterangan: string
  category_nama: string
  unit_nama: string
  source_nama_barang: string
  created_at: string
  updated_at: string
}

interface StatusHistory {
  id: number
  from_status: string | null
  to_status: string
  changed_by_nama: string
  note: string
  created_at: string
}

// ─── legal transitions (sama dengan domain backend) ────────────────────────

const NEXT_STATUS: Record<string, string[]> = {
  pending:   ['checked'],
  checked:   ['processed', 'cancelled'],
  processed: ['finished', 'cancelled'],
  finished:  [],
  cancelled: [],
}

/** Label tombol approve per transisi. */
const APPROVE_LABEL: Record<string, string> = {
  checked:   'Setujui (→ Checked)',
  processed: 'Proses (→ Processed)',
  finished:  'Selesaikan (→ Finished)',
}

// ─── state ─────────────────────────────────────────────────────────────────

const route  = useRoute()
const router = useRouter()
const auth   = useAuthStore()

const receipt       = ref<Receipt | null>(null)
const history       = ref<StatusHistory[]>([])
const pdfBlobUrl    = ref<string | null>(null)
const pdfLoading    = ref(true)
const pdfError      = ref(false)
const catatan       = ref('')
const approving     = ref(false)
const cancelling    = ref(false)
const toast         = ref<{ kind: 'success' | 'error' | 'info'; title: string; description?: string } | null>(null)

/** Hanya admin, ga_staff, approver yang boleh mengubah status. */
const canAct = computed(() => auth.hasRole('admin', 'ga_staff', 'approver'))

/** Status transisi yang tersedia dari status saat ini. */
const nextStatuses = computed<string[]>(() =>
  receipt.value ? (NEXT_STATUS[receipt.value.status] ?? []) : [],
)

/** Status approve berikutnya (bukan cancelled). */
const approveStatus = computed<string | null>(() =>
  nextStatuses.value.find(s => s !== 'cancelled') ?? null,
)

/** Apakah cancelled tersedia dari status saat ini. */
const canCancel = computed<boolean>(() =>
  nextStatuses.value.includes('cancelled'),
)

// ─── helpers ───────────────────────────────────────────────────────────────

function formatRupiah(val: string | number | undefined | null): string {
  if (!val) return 'Rp 0'
  const n = typeof val === 'string' ? parseFloat(val.replace(/[^0-9.-]/g, '')) : val
  if (isNaN(n)) return String(val)
  return 'Rp ' + n.toLocaleString('id-ID')
}

function formatDate(iso: string | undefined | null): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleDateString('id-ID', { day: '2-digit', month: 'long', year: 'numeric' })
}

function formatDateTime(iso: string | undefined | null): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString('id-ID', {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

// ─── load data ─────────────────────────────────────────────────────────────

async function load(): Promise<void> {
  const id = route.params.id as string
  try {
    const [resReceipt, resHistory] = await Promise.all([
      requestWithMeta<Receipt>(`/api/v1/payment-receipts/${id}`),
      requestWithMeta<StatusHistory[]>(`/api/v1/payment-receipts/${id}/status-history`),
    ])
    receipt.value = resReceipt.data
    history.value = resHistory.data ?? []
  } catch (e: any) {
    toast.value = { kind: 'error', title: 'Gagal memuat data', description: e?.message ?? 'Error' }
  }
}

async function loadPdf(): Promise<void> {
  const id = route.params.id as string
  pdfLoading.value = true
  pdfError.value   = false
  try {
    const res = await fetch(`/api/v1/payment-receipts/${id}/pdf`, { credentials: 'include' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const blob  = await res.blob()
    pdfBlobUrl.value = URL.createObjectURL(blob)
  } catch {
    pdfError.value = true
  } finally {
    pdfLoading.value = false
  }
}

onMounted(async () => {
  await Promise.all([load(), loadPdf()])
})

// ─── actions ───────────────────────────────────────────────────────────────

async function changeStatus(newStatus: string): Promise<void> {
  if (!receipt.value) return
  const isCancelling = newStatus === 'cancelled'
  if (isCancelling) cancelling.value = true
  else approving.value = true

  try {
    const res = await requestWithMeta<Receipt>(`/api/v1/payment-receipts/${receipt.value.id}/status`, {
      method: 'PATCH',
      body: { status: newStatus, note: catatan.value.trim() },
    })
    receipt.value = res.data
    catatan.value = ''
    // Reload history setelah update status
    const resH = await requestWithMeta<StatusHistory[]>(`/api/v1/payment-receipts/${receipt.value.id}/status-history`)
    history.value = resH.data ?? []
    toast.value = {
      kind: 'success',
      title: isCancelling ? 'Dokumen dibatalkan' : 'Status diperbarui',
      description: `Status sekarang: ${newStatus}`,
    }
  } catch (e: any) {
    toast.value = { kind: 'error', title: 'Gagal mengubah status', description: e?.message ?? 'Error' }
  } finally {
    approving.value  = false
    cancelling.value = false
  }
}

function downloadPdf(): void {
  const id = route.params.id as string
  window.open(`/api/v1/payment-receipts/${id}/pdf`, '_blank')
}

function openPdfNewTab(): void {
  if (pdfBlobUrl.value) window.open(pdfBlobUrl.value, '_blank')
}

function goBack(): void {
  router.push({ name: 'payment-receipts' })
}
</script>

<template>
  <div class="space-y-4">
    <!-- ── Top bar ───────────────────────────────────────────────────────── -->
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="flex items-center gap-3">
        <button
          type="button"
          class="rounded-lg p-1.5 text-ink-muted transition-colors hover:bg-default hover:text-ink focus-visible:outline-2 focus-visible:outline-brand-500"
          aria-label="Kembali ke daftar"
          @click="goBack()"
        >
          <ArrowLeft class="size-5" aria-hidden="true" />
        </button>
        <div>
          <div class="flex items-center gap-2 flex-wrap">
            <h1 class="text-lg font-semibold tracking-tight text-ink font-mono">
              {{ receipt?.kode ?? '...' }}
            </h1>
            <Badge v-if="receipt" :variant="(receipt.status as any)">
              {{ receipt.status }}
            </Badge>
          </div>
          <p class="mt-0.5 text-xs text-ink-muted">
            {{ receipt?.department_nama }} · {{ receipt?.request_kode }} {{ receipt?.request_nama }}
          </p>
        </div>
      </div>

      <div class="flex gap-2">
        <BaseButton variant="secondary" size="sm" @click="downloadPdf()">
          <template #icon><Download class="size-4" aria-hidden="true" /></template>
          Unduh PDF
        </BaseButton>
        <BaseButton
          v-if="pdfBlobUrl"
          variant="ghost"
          size="sm"
          :aria-label="'Buka PDF di tab baru'"
          @click="openPdfNewTab()"
        >
          <template #icon><ExternalLink class="size-4" aria-hidden="true" /></template>
        </BaseButton>
      </div>
    </div>

    <!-- ── Main layout: PDF kiri, panel kanan ───────────────────────────── -->
    <div class="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_380px]">

      <!-- PDF Viewer -->
      <div class="card overflow-hidden rounded-xl border border-default shadow-xs">
        <!-- Loading -->
        <div
          v-if="pdfLoading"
          class="flex h-[650px] flex-col items-center justify-center gap-3 text-ink-muted"
        >
          <Loader2 class="size-8 animate-spin text-brand-500" aria-hidden="true" />
          <span class="text-sm">Memuat dokumen PDF…</span>
        </div>

        <!-- Error fallback -->
        <div
          v-else-if="pdfError"
          class="flex h-[650px] flex-col items-center justify-center gap-3 p-6 text-center"
        >
          <p class="text-sm font-medium text-ink">Gagal memuat PDF di browser</p>
          <p class="text-xs text-ink-muted">
            Ini bisa terjadi jika backend tidak aktif atau browser memblokir PDF inline.
          </p>
          <BaseButton variant="secondary" size="sm" @click="downloadPdf()">
            <template #icon><Download class="size-4" aria-hidden="true" /></template>
            Buka / Unduh PDF
          </BaseButton>
        </div>

        <!-- iframe PDF -->
        <iframe
          v-else-if="pdfBlobUrl"
          :src="pdfBlobUrl"
          class="h-[650px] w-full"
          title="Dokumen Purchase Requisition"
          aria-label="Pratinjau dokumen PDF Purchase Requisition"
        />
      </div>

      <!-- Review & Approval panel -->
      <aside class="flex flex-col gap-4">

        <!-- Info dokumen -->
        <div class="card rounded-xl border border-default p-4 space-y-3">
          <h2 class="text-sm font-semibold text-ink">Informasi Dokumen</h2>
          <dl class="grid grid-cols-2 gap-x-3 gap-y-2 text-xs">
            <div>
              <dt class="text-ink-muted">Deskripsi Item</dt>
              <dd class="mt-0.5 font-medium text-ink">{{ receipt?.deskripsi_items || '-' }}</dd>
            </div>
            <div>
              <dt class="text-ink-muted">Kategori</dt>
              <dd class="mt-0.5 text-ink">{{ receipt?.category_nama || '-' }}</dd>
            </div>
            <div>
              <dt class="text-ink-muted">Jumlah</dt>
              <dd class="mt-0.5 text-ink">{{ receipt?.jumlah_items ?? '-' }} {{ receipt?.unit_nama || '' }}</dd>
            </div>
            <div>
              <dt class="text-ink-muted">Harga Satuan</dt>
              <dd class="mt-0.5 tabular-nums text-ink">{{ formatRupiah(receipt?.harga_items) }}</dd>
            </div>
            <div class="col-span-2">
              <dt class="text-ink-muted">Total</dt>
              <dd class="mt-0.5 text-base font-bold tabular-nums text-ink">
                {{ formatRupiah(receipt?.total_harga) }}
              </dd>
            </div>
            <div>
              <dt class="text-ink-muted">Tanggal Pembelian</dt>
              <dd class="mt-0.5 text-ink">{{ formatDate(receipt?.tanggal_pembelian) }}</dd>
            </div>
            <div>
              <dt class="text-ink-muted">Sumber</dt>
              <dd class="mt-0.5 text-ink">{{ receipt?.source_nama_barang || '-' }}</dd>
            </div>
            <div v-if="receipt?.usage" class="col-span-2">
              <dt class="text-ink-muted">Usage</dt>
              <dd class="mt-0.5 text-ink">{{ receipt.usage }}</dd>
            </div>
            <div v-if="receipt?.keterangan" class="col-span-2">
              <dt class="text-ink-muted">Keterangan</dt>
              <dd class="mt-0.5 text-ink">{{ receipt.keterangan }}</dd>
            </div>
          </dl>
        </div>

        <!-- Blok TTD (Tanda Tangan) -->
        <div class="card rounded-xl border border-default p-4 space-y-3">
          <h2 class="text-sm font-semibold text-ink">Tanda Tangan Dokumen</h2>
          <div class="grid grid-cols-3 gap-2 text-xs text-center">
            <div class="rounded-lg border border-dashed border-default p-3">
              <p class="text-ink-muted">User</p>
              <div class="mt-3 mb-1 h-px w-full bg-default"></div>
              <p class="text-ink font-medium">{{ receipt?.department_nama || '—' }}</p>
            </div>
            <div class="rounded-lg border border-dashed border-brand-500/40 bg-brand-500/5 p-3">
              <p class="text-ink-muted">General Affairs</p>
              <div class="mt-3 mb-1 h-px w-full bg-brand-200"></div>
              <p class="text-ink font-medium">GA Department</p>
            </div>
            <div class="rounded-lg border border-dashed border-default p-3">
              <p class="text-ink-muted">Dir. Operasional</p>
              <div class="mt-3 mb-1 h-px w-full bg-default"></div>
              <p class="text-ink font-medium text-[10px] leading-tight">Geoffrey Meyssonier</p>
            </div>
          </div>
          <p class="text-[10px] text-ink-muted">
            Tanda tangan fisik tercantum pada dokumen PDF yang dicetak.
          </p>
        </div>

        <!-- Riwayat status -->
        <div class="card rounded-xl border border-default p-4 space-y-2">
          <h2 class="text-sm font-semibold text-ink">Riwayat Status</h2>
          <ol v-if="history.length" class="space-y-2">
            <li
              v-for="h in history"
              :key="h.id"
              class="flex gap-2 text-xs"
            >
              <span class="mt-0.5 size-1.5 shrink-0 rounded-full bg-brand-500 self-start translate-y-1"></span>
              <div class="min-w-0">
                <p class="text-ink">
                  <span class="font-medium">{{ h.from_status ?? '(baru)' }}</span>
                  <span class="mx-1 text-ink-muted">→</span>
                  <Badge :variant="(h.to_status as any)" class="text-[10px]">{{ h.to_status }}</Badge>
                </p>
                <p class="text-ink-muted mt-0.5">
                  {{ h.changed_by_nama }} · {{ formatDateTime(h.created_at) }}
                </p>
                <p v-if="h.note" class="text-ink mt-0.5 italic">{{ h.note }}</p>
              </div>
            </li>
          </ol>
          <p v-else class="text-xs text-ink-muted">Belum ada riwayat.</p>
        </div>

        <!-- Approval actions (hanya untuk yang punya akses) -->
        <div
          v-if="canAct && nextStatuses.length > 0"
          class="card rounded-xl border border-default p-4 space-y-3"
        >
          <h2 class="text-sm font-semibold text-ink">Tindakan Persetujuan</h2>

          <!-- Catatan -->
          <div>
            <label for="catatan-approval" class="block text-xs font-medium text-ink mb-1">
              Catatan <span class="text-ink-muted font-normal">(opsional)</span>
            </label>
            <textarea
              id="catatan-approval"
              v-model="catatan"
              rows="3"
              placeholder="Tambahkan catatan persetujuan atau alasan penolakan…"
              class="block w-full rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink placeholder:text-ink-muted focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none resize-none transition-[var(--transition-fast)] disabled:opacity-60"
              :disabled="approving || cancelling"
            />
          </div>

          <!-- Tombol -->
          <div class="flex flex-col gap-2">
            <!-- Approve / next step -->
            <BaseButton
              v-if="approveStatus"
              :loading="approving"
              :disabled="cancelling"
              full
              @click="changeStatus(approveStatus!)"
            >
              <template #icon>
                <CheckCircle class="size-4" aria-hidden="true" />
              </template>
              {{ APPROVE_LABEL[approveStatus] ?? `Setujui (→ ${approveStatus})` }}
            </BaseButton>

            <!-- Tolak / batalkan -->
            <BaseButton
              v-if="canCancel"
              variant="danger"
              :loading="cancelling"
              :disabled="approving"
              full
              @click="changeStatus('cancelled')"
            >
              <template #icon>
                <XCircle class="size-4" aria-hidden="true" />
              </template>
              Tolak / Batalkan
            </BaseButton>
          </div>
        </div>

        <!-- Status terminal: tidak ada aksi -->
        <div
          v-else-if="receipt && nextStatuses.length === 0"
          class="rounded-xl border border-default bg-raised px-4 py-3 text-xs text-ink-muted text-center"
        >
          Status <strong class="text-ink">{{ receipt.status }}</strong> adalah status final. Tidak ada tindakan lanjutan.
        </div>

        <!-- Viewer-only: tidak tampilkan aksi -->
        <div
          v-else-if="!canAct"
          class="rounded-xl border border-default bg-raised px-4 py-3 text-xs text-ink-muted text-center"
        >
          Anda hanya memiliki akses baca. Hubungi GA Staff atau Admin untuk mengubah status.
        </div>

      </aside>
    </div>
  </div>

  <!-- Toast -->
  <div class="fixed right-4 bottom-4 z-40 w-full max-w-sm">
    <Transition name="toast">
      <ToastNotification
        v-if="toast"
        :kind="toast.kind"
        :title="toast.title"
        :description="toast.description"
        @close="toast = null"
      />
    </Transition>
  </div>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all var(--transition-fast);
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(0.5rem);
}
</style>
