<script setup lang="ts">
/**
 * Halaman daftar Request (pengajuan per departemen).
 *
 * Bedanya dengan halaman master data lain: request selalu miliki satu
 * departemen, jadi ada filter departemen di toolbar dan form tambah/ubah
 * memakai dropdown departemen. Data departemen diambil dari API master.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { Pencil, Plus, Trash2 } from 'lucide-vue-next'

import BaseTable from '@/components/ui/BaseTable.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseModal from '@/components/ui/BaseModal.vue'
import SearchInput from '@/components/ui/SearchInput.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import { requestWithMeta, qs } from '@/lib/http'
import type { MasterItem } from '@/composables/useMasterCrud'
import type { TableState } from '@/components/ui/tableTypes'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const search = ref('')
const departmentFilter = ref('')
const page = ref(1)
const perPage = ref(20)

const departments = ref<MasterItem[]>([])
const departmentsLoaded = ref(false)

const editing = ref<{
  id: number
  kode: string
  nama: string
  keterangan: string
  department_id: number
} | null>(null)
const formOpen = ref(false)
const saving = ref(false)

const deleting = ref<{ id: number; kode: string } | null>(null)
const deleteBusy = ref(false)

const toast = ref<{
  kind: 'success' | 'error' | 'info'
  title: string
  description?: string
} | null>(null)

const canManage = computed(() => auth.hasRole('admin'))

async function loadDepartments(): Promise<void> {
  if (departmentsLoaded.value) return
  try {
    const data = await requestWithMeta<MasterItem[]>(`/api/v1/departments?${qs({ page: 1, per_page: 200 })}`)
    departments.value = data.data
  } catch {
    departments.value = []
  } finally {
    departmentsLoaded.value = true
  }
}

async function fetchList(): Promise<void> {
  const params = qs({
    page: page.value,
    per_page: perPage.value,
    q: search.value.trim() || undefined,
    department_id: departmentFilter.value || undefined,
  })
  const res = await requestWithMeta<{ id: number; kode: string; nama: string; keterangan: string }[]>(
    `/api/v1/requests${params}`,
  )
  rowsValue.value = res.data
  metaValue.value = res.meta as any
}

const rowsValue = ref<{ id: number; kode: string; nama: string; keterangan: string }[]>([])
const metaValue = ref<any>(null)

const rows = computed(() => rowsValue.value)
const meta = computed(() => metaValue.value)

watch([search, departmentFilter], async () => {
  page.value = 1
  await fetchList()
})

watch(page, async () => {
  await fetchList()
})

onMounted(async () => {
  await loadDepartments()
  await fetchList()
})

const state: TableState = 'ready'

async function save(): Promise<void> {
  const item = editing.value!
  if (!item.kode || item.kode.trim().length < 2) {
    toast.value = {
      kind: 'error',
      title: 'Gagal menyimpan',
      description: 'Kode minimal 2 karakter.',
    }
    return
  }
  if (!item.nama || !item.nama.trim()) {
    toast.value = { kind: 'error', title: 'Gagal menyimpan', description: 'Nama wajib diisi.' }
    return
  }
  if (!item.department_id || item.department_id <= 0) {
    toast.value = {
      kind: 'error',
      title: 'Gagal menyimpan',
      description: 'Department wajib dipilih.',
    }
    return
  }

  saving.value = true
  try {
    if (editing.value?.id) {
      await requestWithMeta(`/api/v1/requests/${editing.value.id}`, {
        method: 'PUT',
        body: {
          kode: item.kode.trim(),
          nama: item.nama.trim(),
          keterangan: item.keterangan,
          department_id: item.department_id,
        },
      })
      toast.value = { kind: 'success', title: 'Request diperbarui' }
    } else {
      await requestWithMeta('/api/v1/requests', {
        method: 'POST',
        body: {
          kode: item.kode.trim(),
          nama: item.nama.trim(),
          keterangan: item.keterangan,
          department_id: item.department_id,
        },
      })
      toast.value = { kind: 'success', title: 'Request ditambahkan' }
    }
    closeForm()
    await fetchList()
  } catch (err: any) {
    toast.value = {
      kind: 'error',
      title: 'Gagal menyimpan',
      description: err?.message ?? 'Terjadi kesalahan.',
    }
  } finally {
    saving.value = false
  }
}

function openCreate(): void {
  editing.value = { id: 0, kode: '', nama: '', keterangan: '', department_id: 0 }
  formOpen.value = true
}

function openEdit(
  item: {
    id: number
    kode: string
    nama: string
    keterangan: string
    department_id?: number
  } | null,
): void {
  if (!item) return
  editing.value = {
    id: item.id,
    kode: item.kode,
    nama: item.nama,
    keterangan: item.keterangan ?? '',
    department_id: item.department_id ?? 0,
  }
  formOpen.value = true
}

function closeForm(): void {
  formOpen.value = false
  editing.value = null
}

function askDelete(item: { id: number; kode: string }): void {
  deleting.value = item
}

function cancelDelete(): void {
  deleting.value = null
}

async function confirmDelete(): Promise<void> {
  if (!deleting.value) return

  deleteBusy.value = true
  try {
    await requestWithMeta(`/api/v1/requests/${deleting.value.id}`, { method: 'DELETE' })
    toast.value = { kind: 'success', title: 'Request dihapus' }
    deleting.value = null
    await fetchList()
  } catch (err: any) {
    toast.value = {
      kind: 'error',
      title: 'Tidak bisa dihapus',
      description: err?.message ?? 'Terjadi kesalahan.',
    }
  } finally {
    deleteBusy.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <!-- Header halaman -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold tracking-tight text-ink">Request</h1>
        <p class="mt-0.5 text-sm text-ink-muted">
          Pengajuan pembelian milik satu departemen. Setiap request harus memilih departemen pemohon.
        </p>
      </div>

      <BaseButton v-if="canManage" @click="openCreate()">
        <template #icon><Plus class="size-4" aria-hidden="true" /></template>
        Tambah Request
      </BaseButton>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="relative w-full sm:max-w-xs">
        <select
          v-model="departmentFilter"
          class="block w-full rounded-lg border border-default bg-raised py-2 pr-9 pl-9 text-sm text-ink transition-[var(--transition-fast)] placeholder:text-ink-muted/60 focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none"
        >
          <option value="">Semua department</option>
          <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.nama }}</option>
        </select>
      </div>

      <SearchInput v-model="search" placeholder="Cari kode atau nama..." />
    </div>

    <!-- Tabel -->
    <BaseTable
      :columns="[]"
      :rows="rows"
      :meta="meta"
      :state="state"
      :error-message="null"
      @update:page="page = $event"
    >
      <template #cell-kode="{ row }">
        <span class="font-mono text-xs font-medium text-ink">{{ row.kode }}</span>
      </template>

      <template #cell-nama="{ row }">
        <span class="font-medium">{{ row.nama }}</span>
      </template>

      <template #cell-keterangan="{ row }">
        <span class="text-ink-muted">{{ row.keterangan || '-' }}</span>
      </template>

      <template #cell-aksi="{ row }">
        <div v-if="canManage" class="flex justify-end gap-1">
          <button
            type="button"
            class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-default hover:text-brand-600"
            :aria-label="`Ubah ${row.nama}`"
            @click="openEdit(row)"
          >
            <Pencil class="size-4" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-status-cancelled/10 hover:text-status-cancelled"
            :aria-label="`Hapus ${row.nama}`"
            @click="askDelete(row)"
          >
            <Trash2 class="size-4" aria-hidden="true" />
          </button>
        </div>
        <span v-else class="text-xs text-ink-muted/50">-</span>
      </template>

      <template #empty>
        <p class="text-sm font-medium text-ink">
          {{ search || departmentFilter ? 'Tidak ada request yang cocok' : 'Belum ada request' }}
        </p>
        <p class="mt-1 text-sm text-ink-muted">
          {{
            search || departmentFilter
              ? 'Coba kata kunci lain atau ubah filter.'
              : 'Tambahkan request pertama untuk memulai.'
          }}
        </p>
      </template>
    </BaseTable>

    <!-- Modal form -->
    <BaseModal
      :open="formOpen"
      :title="editing?.id ? 'Ubah Request' : 'Tambah Request'"
      :description="
        editing?.id
          ? 'Perubahan langsung berlaku untuk pembelian yang menggunakan request ini.'
          : 'Department pemohon wajib dipilih.'
      "
      :busy="saving"
      @close="closeForm()"
    >
      <div class="space-y-4">
        <div class="relative">
          <label class="block text-sm font-medium text-ink"
            >Kode <span class="text-status-cancelled" aria-hidden="true">*</span></label
          >
          <input
            v-model="editing!.kode"
            type="text"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
            maxlength="30"
            autocapitalize="characters"
            placeholder="REQ-GA"
          />
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink"
            >Nama <span class="text-status-cancelled" aria-hidden="true">*</span></label
          >
          <input
            v-model="editing!.nama"
            type="text"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
            maxlength="150"
            placeholder="Pengajuan ATK cabang Sinabung"
          />
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink"
            >Department <span class="text-status-cancelled" aria-hidden="true">*</span></label
          >
          <select
            v-model="editing!.department_id"
            class="block w-full rounded-lg border bg-raised py-2 pr-9 pl-9 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
          >
            <option value="">Pilih department</option>
            <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.nama }}</option>
          </select>
        </div>

        <div class="relative">
          <label class="block text-sm font-medium text-ink">Keterangan</label>
          <input
            v-model="editing!.keterangan"
            type="text"
            class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] focus:ring-2 focus:outline-none focus:border-brand-500 focus:ring-brand-500/20"
            placeholder="Opsional"
          />
        </div>
      </div>

      <template #footer>
        <BaseButton variant="secondary" :disabled="saving" @click="closeForm()"> Batal </BaseButton>
        <BaseButton :loading="saving" @click="save()"> Simpan </BaseButton>
      </template>
    </BaseModal>

    <!-- Konfirmasi hapus -->
    <ConfirmDialog
      :open="!!deleting"
      title="Hapus Request?"
      :message="`Request ${deleting?.kode ?? ''} akan dinonaktifkan.`"
      :item-name="deleting?.kode"
      :busy="deleteBusy"
      confirm-label="Hapus"
      @confirm="confirmDelete()"
      @cancel="cancelDelete()"
    >
      <p class="mt-3 rounded-lg bg-default p-3 text-xs text-ink-muted">
        Request yang masih dipakai payment receipt tidak bisa dihapus.
      </p>
    </ConfirmDialog>

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
