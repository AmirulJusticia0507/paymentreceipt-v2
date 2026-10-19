<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Pencil, Plus, Trash2 } from 'lucide-vue-next'

import BaseTable from '@/components/ui/BaseTable.vue'
import type { Column } from '@/components/ui/tableTypes'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseModal from '@/components/ui/BaseModal.vue'
import SearchInput from '@/components/ui/SearchInput.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import { requestWithMeta, qs } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import type { TableState } from '@/components/ui/tableTypes'

const auth = useAuthStore()

const search = ref('')
const dateFrom = ref('')
const dateTo = ref('')
const page = ref(1)
const perPage = ref(20)

const rowsValue = ref<any[]>([])
const metaValue = ref<any>(null)
const rows = computed(() => rowsValue.value)
const meta = computed(() => metaValue.value)

const deleteBusy = ref(false)
const deleting = ref<any>(null)

const formOpen = ref(false)
const saving = ref(false)
const editing = ref<any>(null)

const toast = ref<{
  kind: 'success' | 'error' | 'info'
  title: string
  description?: string
} | null>(null)

const canManage = computed(() => auth.hasRole('admin'))

const bulkOpen = ref(false)
const bulkReceipts = ref<any[]>([])
const bulkSelected = ref<Set<number>>(new Set())
const bulkSaving = ref(false)

const columns: Column<any>[] = [
  { key: 'kode', label: 'Kode', width: 'w-32', sortable: false },
  { key: 'nama_request', label: 'Nama Request' },
  { key: 'receipt_kode', label: 'Receipt Asal', width: 'w-32' },
  { key: 'harga_aktual', label: 'Harga Aktual', align: 'right', width: 'w-32' },
  { key: 'tanggal_selesai', label: 'Tanggal', width: 'w-32' },
  { key: 'aksi', label: 'Aksi', align: 'right', width: 'w-24' },
]

const state: TableState = 'ready'

async function fetchList(): Promise<void> {
  const params = qs({
    page: page.value,
    per_page: perPage.value,
    q: search.value.trim() || undefined,
    date_from: dateFrom.value || undefined,
    date_to: dateTo.value || undefined,
  })
  try {
    const res = await requestWithMeta<any[]>(`/api/v1/pr-archives${params}`)
    rowsValue.value = res.data
    metaValue.value = res.meta
  } catch (e) {
    rowsValue.value = []
    metaValue.value = null
  }
}

onMounted(fetchList)

async function loadBulkReceipts(): Promise<void> {
  try {
    const res = await requestWithMeta<any[]>(`/api/v1/payment-receipts?status=finished&per_page=200`)
    bulkReceipts.value = res.data ?? []
  } catch {
    bulkReceipts.value = []
  }
  bulkSelected.value = new Set()
}

function toggleBulk(id: number): void {
  if (bulkSelected.value.has(id)) bulkSelected.value.delete(id)
  else bulkSelected.value.add(id)
}

const bulkTotal = computed(() => {
  let total = 0
  for (const id of bulkSelected.value) {
    const r = bulkReceipts.value.find((x) => x.id === id)
    if (r) total += r.total_harga ? parseFloat(r.total_harga) : 0
  }
  return total
})

async function submitBulk(): Promise<void> {
  bulkSaving.value = true
  try {
    await requestWithMeta('/api/v1/pr-archives/bulk-archive', {
      method: 'POST',
      body: {
        tanggal_selesai: new Date().toISOString(),
        nama_request: 'ARSIP PR',
        manager_approved: '',
        description: '',
        harga_rencana: '0',
        kode: '',
        payment_receipt_ids: [...bulkSelected.value],
      },
    })
    toast.value = { kind: 'success', title: 'Bulk archive berhasil' }
    bulkOpen.value = false
    await fetchList()
  } catch (err: any) {
    toast.value = {
      kind: 'error',
      title: 'Gagal bulk archive',
      description: err?.message ?? 'Error',
    }
  } finally {
    bulkSaving.value = false
  }
}

function openBulk(): void {
  bulkOpen.value = true
  loadBulkReceipts()
}

function askDelete(item: any): void {
  deleting.value = item
}

function cancelDelete(): void {
  deleting.value = null
}

async function confirmDelete(): Promise<void> {
  if (!deleting.value) return
  deleteBusy.value = true
  try {
    await requestWithMeta(`/api/v1/pr-archives/${deleting.value.id}`, { method: 'DELETE' })
    toast.value = { kind: 'success', title: 'Arsip dihapus' }
    deleting.value = null
    await fetchList()
  } catch (err: any) {
    toast.value = { kind: 'error', title: 'Gagal hapus', description: err?.message ?? 'Error' }
  } finally {
    deleteBusy.value = false
  }
}

function openCreate(): void {
  editing.value = {
    id: 0,
    kode: '',
    payment_receipt_id: 0,
    nama_request: '',
    manager_approved: '',
    description: '',
    quantity: 0,
    harga_rencana: '0',
    harga_aktual: '0',
    tanggal_selesai: new Date().toISOString(),
  }
  formOpen.value = true
}

function closeForm(): void {
  formOpen.value = false
  editing.value = null
}

async function saveForm(): Promise<void> {
  if (!editing.value) return
  saving.value = true
  try {
    const body = { ...editing.value }
    if (editing.value.id) {
      await requestWithMeta(`/api/v1/pr-archives/${editing.value.id}`, { method: 'PUT', body })
      toast.value = { kind: 'success', title: 'Arsip diperbarui' }
    } else {
      await requestWithMeta('/api/v1/pr-archives', { method: 'POST', body })
      toast.value = { kind: 'success', title: 'Arsip ditambahkan' }
    }
    closeForm()
    await fetchList()
  } catch (err: any) {
    toast.value = { kind: 'error', title: 'Gagal menyimpan', description: err?.message ?? 'Error' }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold tracking-tight text-ink">Arsip PR</h1>
        <p class="mt-0.5 text-sm text-ink-muted">
          Arsip Purchase Requisition dari payment receipt yang sudah finished.
        </p>
      </div>
      <div class="flex gap-2">
        <BaseButton variant="secondary" @click="openBulk()">Bulk Archive</BaseButton>
        <BaseButton v-if="canManage" @click="openCreate()">
          <template #icon><Plus class="size-4" aria-hidden="true" /></template>
          Tambah Arsip
        </BaseButton>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <SearchInput v-model="search" placeholder="Cari kode, nama request, approver..." />
      <input
        v-model="dateFrom"
        type="date"
        class="rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none"
      />
      <input
        v-model="dateTo"
        type="date"
        class="rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none"
      />
      <BaseButton variant="secondary" size="sm" @click="fetchList()">Cari</BaseButton>
    </div>

    <BaseTable
      :columns="columns"
      :rows="rows"
      :meta="meta"
      :state="state"
      :error-message="null"
      @update:page="
        (e: number) => {
          page = e
          fetchList()
        }
      "
    >
      <template #cell-kode="{ row }">
        <span class="font-mono text-xs font-medium text-ink">{{ row.kode }}</span>
      </template>

      <template #cell-nama_request="{ row }">
        <span class="font-medium">{{ row.nama_request }}</span>
      </template>

      <template #cell-receipt_kode="{ row }">
        <span class="font-mono text-xs text-ink-muted">{{ row.receipt_kode }}</span>
      </template>

      <template #cell-harga_aktual="{ row }">
        <span class="tabular-nums">{{ row.harga_aktual }}</span>
      </template>

      <template #cell-tanggal_selesai="{ row }">
        <span class="text-xs text-ink-muted">{{
          row.tanggal_selesai ? new Date(row.tanggal_selesai).toLocaleDateString('id-ID') : '-'
        }}</span>
      </template>

      <template #cell-aksi="{ row }">
        <div v-if="canManage" class="flex justify-end gap-1">
          <button
            type="button"
            class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-default hover:text-brand-600"
            :aria-label="`Ubah ${row.kode}`"
            @click="() => {}"
          >
            <Pencil class="size-4" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-status-cancelled/10 hover:text-status-cancelled"
            :aria-label="`Hapus ${row.kode}`"
            @click="askDelete(row)"
          >
            <Trash2 class="size-4" aria-hidden="true" />
          </button>
        </div>
        <span v-else class="text-xs text-ink-muted/50">-</span>
      </template>

      <template #empty>
        <p class="text-sm font-medium text-ink">Belum ada arsip PR</p>
        <p class="mt-1 text-sm text-ink-muted">Arsip dibuat dari receipt yang sudah finished.</p>
      </template>
    </BaseTable>

    <!-- Modal create -->
    <BaseModal
      :open="formOpen"
      :title="editing?.id ? 'Ubah Arsip PR' : 'Tambah Arsip PR'"
      :busy="saving"
      @close="closeForm()"
    >
      <div class="space-y-4">
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Kode</label>
          <input
            v-model="editing.kode"
            type="text"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          />
        </div>
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Payment Receipt ID</label>
          <input
            v-model.number="editing.payment_receipt_id"
            type="number"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          />
        </div>
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Nama Request</label>
          <input
            v-model="editing.nama_request"
            type="text"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          />
        </div>
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Manager Approved</label>
          <input
            v-model="editing.manager_approved"
            type="text"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          />
        </div>
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Deskripsi</label>
          <textarea
            v-model="editing.description"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          />
        </div>
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Qty</label>
          <input
            v-model.number="editing.quantity"
            type="number"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          />
        </div>
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Harga Rencana</label>
          <input
            v-model="editing.harga_rencana"
            type="text"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          />
        </div>
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Harga Aktual</label>
          <input
            v-model="editing.harga_aktual"
            type="text"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          />
        </div>
      </div>
      <template #footer>
        <BaseButton variant="secondary" :disabled="saving" @click="closeForm()">Batal</BaseButton>
        <BaseButton :loading="saving" @click="saveForm()">Simpan</BaseButton>
      </template>
    </BaseModal>

    <!-- Bulk archive modal -->
    <BaseModal :open="bulkOpen" title="Bulk Archive" :busy="bulkSaving" @close="bulkOpen = false">
      <div class="space-y-3">
        <p class="text-sm text-ink-muted">Pilih receipt yang sudah berstatus finished dan belum diarsipkan.</p>
        <div class="max-h-60 overflow-y-auto space-y-2">
          <label v-for="r in bulkReceipts" :key="r.id" class="flex items-center gap-2 text-sm">
            <input type="checkbox" :value="r.id" @change="toggleBulk(r.id)" :checked="bulkSelected.has(r.id)" />
            <span class="font-mono text-xs">{{ r.kode }}</span>
            <span class="flex-1 truncate text-ink">{{ r.deskripsi_items }}</span>
            <span class="tabular-nums text-ink-muted text-xs">{{ r.total_harga }}</span>
          </label>
        </div>
        <div class="text-sm font-semibold text-ink">
          Total: {{ bulkTotal.toFixed(2) }} ({{ bulkSelected.size }} receipt)
        </div>
      </div>
      <template #footer>
        <BaseButton variant="secondary" :disabled="bulkSaving" @click="bulkOpen = false">Batal</BaseButton>
        <BaseButton :loading="bulkSaving" @click="submitBulk()">Arsipkan</BaseButton>
      </template>
    </BaseModal>

    <!-- Delete confirm -->
    <ConfirmDialog
      :open="!!deleting"
      title="Hapus Arsip PR?"
      :message="`Arsip ${deleting?.kode ?? ''} akan dihapus permanen.`"
      :item-name="deleting?.kode"
      :busy="deleteBusy"
      confirm-label="Hapus"
      @confirm="confirmDelete()"
      @cancel="cancelDelete()"
    />

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
