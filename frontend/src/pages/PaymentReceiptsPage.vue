<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { FileText, Pencil, Plus, Trash2 } from 'lucide-vue-next'

import BaseTable from '@/components/ui/BaseTable.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseModal from '@/components/ui/BaseModal.vue'
import SearchInput from '@/components/ui/SearchInput.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import Badge from '@/components/ui/Badge.vue'
import { requestWithMeta, qs } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const STATUS_TRANSITIONS: Record<string, string[]> = {
  pending: ['checked'],
  checked: ['processed', 'cancelled'],
  processed: ['finished', 'cancelled'],
  finished: [],
  cancelled: [],
}

const search = ref('')
const statusFilter = ref('')
const departmentFilter = ref('')
const requestFilter = ref('')
const dateFrom = ref('')
const dateTo = ref('')
const page = ref(1)
const perPage = ref(20)

const rowsValue = ref<any[]>([])
const metaValue = ref<any>(null)
const rows = computed(() => rowsValue.value)
const meta = computed(() => metaValue.value)

const state = ref<'ready' | 'loading'>('ready')

const bulkSelected = ref<Set<number>>(new Set())
const bulkStatus = ref('')
const bulkBusy = ref(false)
function toggleRow(id: number): void {
  if (bulkSelected.value.has(id)) bulkSelected.value.delete(id)
  else bulkSelected.value.add(id)
}


const tableColumns = computed(() => {
  const base: any[] = [
    { key: 'kode', label: 'Kode', width: 'w-32', sortable: false },
    { key: 'deskripsi_items', label: 'Deskripsi' },
    { key: 'department_nama', label: 'Departemen', width: 'w-32' },
    { key: 'status', label: 'Status', width: 'w-28' },
    { key: 'total_harga', label: 'Total', align: 'right', width: 'w-32' },
    { key: 'tanggal_pembelian', label: 'Tanggal', width: 'w-28' },
    { key: 'aksi', label: 'Aksi', align: 'right', width: 'w-28' },
  ]
  if (canManage.value) base.unshift({ key: 'pilih', label: '', width: 'w-8', sortable: false })
  return base
})

const departments = ref<any[]>([])
const requests = ref<any[]>([])
const categories = ref<any[]>([])
const satuanList = ref<any[]>([])
const optionsLoaded = ref(false)

const editing = ref<any>(null)
const formOpen = ref(false)
const saving = ref(false)

const deleting = ref<any>(null)
const deleteBusy = ref(false)
const statusChanging = ref<number | null>(null)

const detailOpen = ref(false)
const detailItem = ref<any>(null)
const statusHistory = ref<any[]>([])

const toast = ref<{ kind: 'success' | 'error' | 'info'; title: string; description?: string } | null>(null)

const canManage = computed(() => auth.hasRole('admin', 'ga_staff'))


async function loadOptions(): Promise<void> {
  if (optionsLoaded.value) return
  try {
    const d = await requestWithMeta<any[]>(`/api/v1/departments?per_page=200`)
    departments.value = d.data ?? []
    const r = await requestWithMeta<any[]>(`/api/v1/requests?per_page=500`)
    requests.value = r.data ?? []
    const c = await requestWithMeta<any[]>(`/api/v1/categories?per_page=200`)
    categories.value = c.data ?? []
    const u = await requestWithMeta<any[]>(`/api/v1/satuan?per_page=200`)
    satuanList.value = u.data ?? []
  } finally {
    optionsLoaded.value = true
  }
}

function requestsByDepartment(dep: number): any[] {
  if (!dep) return requests.value
  return requests.value.filter((r: any) => r.department_id === dep)
}

async function fetchList(): Promise<void> {
  const params = qs({
    page: page.value,
    per_page: perPage.value,
    q: search.value.trim() || undefined,
    status: statusFilter.value || undefined,
    department_id: departmentFilter.value || undefined,
    request_id: requestFilter.value || undefined,
    date_from: dateFrom.value || undefined,
    date_to: dateTo.value || undefined,
  })
  state.value = 'loading'
  try {
    const res = await requestWithMeta<any[]>(`/api/v1/payment-receipts${params}`)
    rowsValue.value = res.data ?? []
    metaValue.value = res.meta
  } catch (e: any) {
    rowsValue.value = []
    toast.value = { kind: 'error', title: 'Gagal memuat daftar', description: e?.message ?? 'Error' }
  } finally {
    state.value = 'ready'
  }
}

onMounted(async () => {
  await loadOptions()
  await fetchList()
})

watch([search, statusFilter, departmentFilter, requestFilter, dateFrom, dateTo], () => { page.value = 1; fetchList() })
watch(page, () => fetchList())

const total = computed(() => {
  if (!editing.value) return 0
  const qty = Number(editing.value.jumlah_items) || 0
  const price = parseFloat(editing.value.harga_items) || 0
  return qty * price
})

function openCreate(): void {
  editing.value = {
    id: 0,
    kode: '',
    request_id: '',
    department_id: '',
    source_id: '',
    category_id: '',
    unit_id: '',
    tanggal_pembelian: new Date().toISOString().slice(0, 10),
    usage: '',
    deskripsi_items: '',
    jumlah_items: 1,
    harga_items: '',
    price_estimate: '',
    keterangan: '',
  }
  formOpen.value = true
}

function openEdit(item: any): void {
  editing.value = {
    id: item.id,
    kode: item.kode,
    request_id: item.request_id,
    department_id: item.department_id,
    source_id: item.source_id,
    category_id: item.category_id,
    unit_id: item.unit_id,
    tanggal_pembelian: (item.tanggal_pembelian ?? '').slice(0, 10),
    usage: item.usage ?? '',
    deskripsi_items: item.deskripsi_items ?? '',
    jumlah_items: item.jumlah_items,
    harga_items: item.harga_items,
    price_estimate: item.price_estimate ?? '',
    keterangan: item.keterangan ?? '',
  }
  formOpen.value = true
}

function closeForm(): void {
  formOpen.value = false
  editing.value = null
}

async function save(): Promise<void> {
  if (!editing.value) return
  saving.value = true
  const src = editing.value.source_id == '' ? null : Number(editing.value.source_id)
  const cat = editing.value.category_id == '' ? null : Number(editing.value.category_id)
  const unit = editing.value.unit_id == '' ? null : Number(editing.value.unit_id)
  const body: any = {
    request_id: Number(editing.value.request_id),
    department_id: Number(editing.value.department_id),
    source_id: src,
    category_id: cat,
    unit_id: unit,
    tanggal_pembelian: editing.value.tanggal_pembelian,
    usage: editing.value.usage,
    deskripsi_items: editing.value.deskripsi_items,
    jumlah_items: Number(editing.value.jumlah_items),
    harga_items: editing.value.harga_items,
    price_estimate: editing.value.price_estimate === '' ? null : editing.value.price_estimate,
    keterangan: editing.value.keterangan ?? '',
  }
  try {
    if (editing.value.id) {
      await requestWithMeta(`/api/v1/payment-receipts/${editing.value.id}`, { method: 'PUT', body })
      toast.value = { kind: 'success', title: 'Payment receipt diperbarui' }
    } else {
      await requestWithMeta('/api/v1/payment-receipts', { method: 'POST', body })
      toast.value = { kind: 'success', title: 'Payment receipt ditambahkan' }
    }
    closeForm()
    await fetchList()
  } catch (e: any) {
    toast.value = { kind: 'error', title: 'Gagal menyimpan', description: e?.message ?? 'Error' }
  } finally {
    saving.value = false
  }
}

function askDelete(item: any): void {
  deleting.value = item
}
function cancelDelete(): void { deleting.value = null }

async function confirmDelete(): Promise<void> {
  if (!deleting.value) return
  deleteBusy.value = true
  try {
    await requestWithMeta(`/api/v1/payment-receipts/${deleting.value.id}`, { method: 'DELETE' })
    toast.value = { kind: 'success', title: 'Payment receipt dihapus' }
    deleting.value = null
    await fetchList()
  } catch (e: any) {
    toast.value = { kind: 'error', title: 'Tidak bisa dihapus', description: e?.message ?? 'Error' }
  } finally {
    deleteBusy.value = false
  }
}

async function applyBulk(): Promise<void> {
  if (bulkSelected.value.size === 0 || !bulkStatus.value) return
  bulkBusy.value = true
  try {
    const res: any = await requestWithMeta('/api/v1/payment-receipts/bulk-status', {
      method: 'PATCH',
      body: { ids: [...bulkSelected.value], status: bulkStatus.value, note: '' },
    })
    toast.value = { kind: 'success', title: 'Bulk status berhasil', description: `Diubah: ${res.data?.changed ?? bulkSelected.value.size}` }
    bulkSelected.value = new Set()
    bulkStatus.value = ''
    await fetchList()
  } catch (e: any) {
    toast.value = { kind: 'error', title: 'Bulk gagal', description: e?.message ?? 'Error' }
  } finally {
    bulkBusy.value = false
  }
}

async function exportCsv(): Promise<void> {
  const params = qs({
    q: search.value.trim() || undefined,
    status: statusFilter.value || undefined,
    department_id: departmentFilter.value || undefined,
    request_id: requestFilter.value || undefined,
    date_from: dateFrom.value || undefined,
    date_to: dateTo.value || undefined,
  })
  window.location.href = `/api/v1/payment-receipts/export${params}`
}

async function changeStatus(item: any, newStatus: string): Promise<void> {
  statusChanging.value = item.id
  try {
    await requestWithMeta(`/api/v1/payment-receipts/${item.id}/status`, {
      method: 'PATCH',
      body: { status: newStatus, note: '' },
    })
    toast.value = { kind: 'success', title: `Status â†’ ${newStatus}` }
    await fetchList()
  } catch (e: any) {
    toast.value = { kind: 'error', title: 'Gagal ubah status', description: e?.message ?? 'Error' }
  } finally {
    statusChanging.value = null
  }
}

async function openDetail(item: any): Promise<void> {
  const res = await requestWithMeta<any>(`/api/v1/payment-receipts/${item.id}`)
  detailItem.value = res.data
  const sh = await requestWithMeta<any[]>(`/api/v1/payment-receipts/${item.id}/status-history`)
  statusHistory.value = sh.data ?? []
  detailOpen.value = true
}

function closeDetail(): void { detailOpen.value = false; detailItem.value = null; statusHistory.value = [] }
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold tracking-tight text-ink">Payment Receipt</h1>
        <p class="mt-0.5 text-sm text-ink-muted">Transaksi pembelian General Affairs.</p>
      </div>
      <BaseButton v-if="canManage" @click="openCreate()">
        <template #icon><Plus class="size-4" aria-hidden="true" /></template>
        Tambah
      </BaseButton>
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <SearchInput v-model="search" placeholder="Cari kode, deskripsi..." />
      <select v-model="statusFilter" class="rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none">
        <option value="">Semua status</option>
        <option value="pending">Pending</option>
        <option value="checked">Checked</option>
        <option value="processed">Processed</option>
        <option value="finished">Finished</option>
        <option value="cancelled">Cancelled</option>
      </select>
      <select v-model="departmentFilter" class="rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none">
        <option value="">Semua department</option>
        <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.nama }}</option>
      </select>
      <select v-model="requestFilter" class="rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none">
        <option value="">Semua request</option>
        <option v-for="r in requestsByDepartment(Number(departmentFilter))" :key="r.id" :value="r.id">{{ r.kode }} - {{ r.nama }}</option>
      </select>
      <input v-model="dateFrom" type="date" class="rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none" />
      <input v-model="dateTo" type="date" class="rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none" />
    </div>

    <div v-if="canManage && bulkSelected.size > 0" class="flex flex-wrap items-center gap-2 rounded-lg border border-default bg-raised px-3 py-2">
      <span class="text-sm text-ink">{{ bulkSelected.size }} terpilih</span>
      <select v-model="bulkStatus" class="rounded border border-default bg-raised px-2 py-1 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none">
        <option value="">Pilih status</option>
        <option value="checked">Checked</option>
        <option value="processed">Processed</option>
        <option value="finished">Finished</option>
        <option value="cancelled">Cancelled</option>
      </select>
      <BaseButton size="sm" :loading="bulkBusy" :disabled="!bulkStatus" @click="applyBulk()">Terapkan</BaseButton>
    </div>

    <BaseButton variant="secondary" size="sm" @click="exportCsv()">Export CSV</BaseButton>

    <BaseTable
      :columns="tableColumns"
      :rows="rows"
      :meta="meta"
      :state="state"
      :error-message="null"
      @update:page="(e: number) => { page = e; fetchList() }"
    >
      <template v-if="canManage" #cell-pilih="{ row }">
        <input type="checkbox" :checked="bulkSelected.has(row.id)" @change="toggleRow(row.id)" />
      </template>

      <template #cell-kode="{ row }">
        <span class="font-mono text-xs font-medium text-ink">{{ row.kode }}</span>
      </template>

      <template #cell-deskripsi_items="{ row }">
        <span>{{ row.deskripsi_items }}</span>
      </template>

      <template #cell-department_nama="{ row }">
        <span class="text-xs text-ink-muted">{{ row.department_nama }}</span>
      </template>

      <template #cell-status="{ row }">
        <div class="flex items-center gap-1">
          <Badge :variant="(row.status as any)">{{ row.status }}</Badge>
          <select
            v-if="canManage"
            class="text-xs rounded border border-default bg-raised px-1 py-0.5"
            :disabled="statusChanging === row.id"
            @change="changeStatus(row, ($event.target as HTMLSelectElement).value)"
          >
            <option value="">Ã—</option>
            <option v-for="s in STATUS_TRANSITIONS[row.status] ?? []" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>
      </template>

      <template #cell-total_harga="{ row }">
        <span class="tabular-nums">{{ row.total_harga }}</span>
      </template>

      <template #cell-tanggal_pembelian="{ row }">
        <span class="text-xs text-ink-muted">{{ row.tanggal_pembelian ? new Date(row.tanggal_pembelian).toLocaleDateString('id-ID') : '-' }}</span>
      </template>

      <template #cell-aksi="{ row }">
        <div class="flex justify-end gap-1">
          <button type="button" class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-default hover:text-brand-600" :aria-label="`Detail ${row.kode}`" @click="openDetail(row)">
            <FileText class="size-4" aria-hidden="true" />
          </button>
          <button v-if="canManage" type="button" class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-default hover:text-brand-600" :aria-label="`Ubah ${row.kode}`" @click="openEdit(row)">
            <Pencil class="size-4" aria-hidden="true" />
          </button>
          <button v-if="auth.hasRole('admin')" type="button" class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-status-cancelled/10 hover:text-status-cancelled" :aria-label="`Hapus ${row.kode}`" @click="askDelete(row)">
            <Trash2 class="size-4" aria-hidden="true" />
          </button>
        </div>
      </template>

      <template #empty>
        <p class="text-sm font-medium text-ink">Belum ada payment receipt</p>
        <p class="mt-1 text-sm text-ink-muted">Tambah transaksi pembelian.</p>
      </template>
    </BaseTable>

    <!-- Create/Edit modal -->
    <BaseModal :open="formOpen" :title="editing?.id ? 'Ubah Payment Receipt' : 'Tambah Payment Receipt'" :busy="saving" @close="closeForm()">
      <div class="space-y-4">
        <div class="relative">
          <label class="block text-sm font-medium text-ink">Departemen <span class="text-status-cancelled" aria-hidden="true">*</span></label>
          <select v-model="editing.department_id" class="block w-full rounded-lg border bg-raised py-2 px-3 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none">
            <option value="">Pilih department</option>
            <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.nama }}</option>
          </select>
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink">Request <span class="text-status-cancelled" aria-hidden="true">*</span></label>
          <select v-model="editing.request_id" class="block w-full rounded-lg border bg-raised py-2 px-3 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none">
            <option value="">Pilih request</option>
            <option v-for="r in requestsByDepartment(Number(editing.department_id))" :key="r.id" :value="r.id">{{ r.kode }} - {{ r.nama }}</option>
          </select>
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink">Kategori</label>
          <select v-model="editing.category_id" class="block w-full rounded-lg border bg-raised py-2 px-3 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none">
            <option value="">-</option>
            <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.nama }}</option>
          </select>
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink">Satuan</label>
          <select v-model="editing.unit_id" class="block w-full rounded-lg border bg-raised py-2 px-3 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none">
            <option value="">-</option>
            <option v-for="u in satuanList" :key="u.id" :value="u.id">{{ u.nama }}</option>
          </select>
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink">Tanggal Pembelian <span class="text-status-cancelled" aria-hidden="true">*</span></label>
          <input v-model="editing.tanggal_pembelian" type="date" class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20" />
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink">Deskripsi Item <span class="text-status-cancelled" aria-hidden="true">*</span></label>
          <input v-model="editing.deskripsi_items" type="text" maxlength="255" class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20" />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div class="relative">
            <label class="block text-sm font-medium text-ink">Jumlah</label>
            <input v-model.number="editing.jumlah_items" type="number" min="1" class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20" />
          </div>
          <div class="relative">
            <label class="block text-sm font-medium text-ink">Harga</label>
            <input v-model="editing.harga_items" type="text" class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20" />
          </div>
        </div>

        <div class="text-sm text-ink-muted">
          Total (auto): <span class="font-semibold text-ink">Rp {{ total.toLocaleString('id-ID') }}</span>
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink">Usage</label>
          <input v-model="editing.usage" type="text" maxlength="150" placeholder="Opsional" class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20" />
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink">Keterangan</label>
          <textarea v-model="editing.keterangan" class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20" />
        </div>
      </div>
      <template #footer>
        <BaseButton variant="secondary" :disabled="saving" @click="closeForm()">Batal</BaseButton>
        <BaseButton :loading="saving" @click="save()">Simpan</BaseButton>
      </template>
    </BaseModal>

    <!-- Delete confirm -->
    <ConfirmDialog
      :open="!!deleting"
      title="Hapus Payment Receipt?"
      :message="`Receipt ${deleting?.kode ?? ''} akan dihapus permanen (jika belum processed/finished).`"
      :item-name="deleting?.kode"
      :busy="deleteBusy"
      confirm-label="Hapus"
      @confirm="confirmDelete()"
      @cancel="cancelDelete()"
    />

    <!-- Detail modal with status history -->
    <BaseModal :open="detailOpen" title="Detail Payment Receipt" @close="closeDetail()">
      <div class="space-y-3 text-sm">
        <pre class="whitespace-pre-wrap text-ink">{{ detailItem }}</pre>
        <div>
          <h3 class="font-semibold text-ink mb-2">Riwayat Status</h3>
          <ol class="space-y-1 list-disc pl-4 text-ink-muted">
            <li v-for="h in statusHistory" :key="h.id">
              {{ h.from_status ?? '-' }} â†’ {{ h.to_status }} ({{ h.changed_by_nama }}, {{ new Date(h.created_at).toLocaleString('id-ID') }}){{ h.note ? ' - ' + h.note : '' }}
            </li>
          </ol>
          <p v-if="statusHistory.length === 0" class="text-xs text-ink-muted">Belum ada riwayat.</p>
        </div>
      </div>
    </BaseModal>

    <!-- Toast -->
    <div class="fixed right-4 bottom-4 z-40 w-full max-w-sm">
      <Transition name="toast">
        <ToastNotification v-if="toast" :kind="toast.kind" :title="toast.title" :description="toast.description" @close="toast = null" />
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: all var(--transition-fast); }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateY(0.5rem); }
</style>
