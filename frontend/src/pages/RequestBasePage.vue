<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Pencil, Plus, Trash2 } from 'lucide-vue-next'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseModal from '@/components/ui/BaseModal.vue'
import BaseTable from '@/components/ui/BaseTable.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import SearchInput from '@/components/ui/SearchInput.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import type { Column, TableState } from '@/components/ui/tableTypes'
import { ApiError, http, requestWithMeta, qs } from '@/lib/http'
import type { PageMeta } from '@/lib/types'
import { useAuthStore } from '@/stores/auth'
import { useAutocomplete, type AutocompleteOption } from '@/composables/useAutocomplete'

interface BaseItem {
  id: number
  kode: string
  nama: string
}
interface RequestBase {
  id: number
  kode: string
  nama: string
  jabatan: string
  keterangan: string
  base_id: number
}

const auth = useAuthStore()
const rows = ref<RequestBase[]>([]),
  bases = ref<BaseItem[]>([])
const meta = ref<PageMeta>(),
  state = ref<TableState>('loading'),
  errorMessage = ref<string | null>(null)
const search = ref(''),
  baseFilter = ref(''),
  page = ref(1)
const baseQuery = ref('')
const baseAutocomplete = useAutocomplete('/api/v1/bases/search', baseQuery)
const formOpen = ref(false),
  saving = ref(false),
  editing = ref<RequestBase | null>(null)
const deleting = ref<RequestBase | null>(null),
  deleteBusy = ref(false)
const toast = ref<{ kind: 'success' | 'error'; title: string; description?: string } | null>(null)
const canManage = computed(() => auth.hasRole('admin'))

/**
 * Pilih base dari hasil autocomplete.
 *
 * Ditaruh di fungsi, bukan ditulis langsung di atribut @click. Vue memang
 * mengizinkan beberapa pernyataan dalam satu handler, tapi pemisah `;`
 * antar pernyataan itu hilang saat Prettier membungkus handler ke beberapa
 * baris, dan hasilnya ekspresi yang tidak sah: build gagal di compiler Vue
 * dengan pesan "Unexpected token" yang menunjuk baris yang tidak jelas
 * penyebabnya. Satu fungsi tidak punya masalah itu.
 */
function pickBase(option: AutocompleteOption): void {
  if (!editing.value) return
  editing.value.base_id = option.value
  baseQuery.value = `${option.kode} — ${option.label}`
}
const columns: Column<RequestBase>[] = [
  { key: 'kode', label: 'Kode', width: 'w-28' },
  { key: 'nama', label: 'Nama' },
  { key: 'base', label: 'Base' },
  { key: 'jabatan', label: 'Jabatan', hideOnMobile: true },
  { key: 'aksi', label: 'Aksi', align: 'right', width: 'w-24' },
]
const baseName = (id: number) => bases.value.find((item) => item.id === id)?.nama ?? '-'

async function loadBases() {
  const response = await requestWithMeta<BaseItem[]>(`/api/v1/bases${qs({ page: 1, per_page: 200 })}`)
  bases.value = response.data
}
async function fetchList() {
  state.value = 'loading'
  errorMessage.value = null
  try {
    const response = await requestWithMeta<RequestBase[]>(
      `/api/v1/request-bases${qs({ page: page.value, per_page: 20, q: search.value.trim(), base_id: baseFilter.value })}`,
    )
    rows.value = response.data
    meta.value = response.meta as PageMeta
    state.value = 'ready'
  } catch (error) {
    state.value = 'error'
    errorMessage.value = error instanceof Error ? error.message : 'Gagal memuat data.'
  }
}
watch([search, baseFilter], () => {
  page.value = 1
  void fetchList()
})
watch(page, () => void fetchList())
onMounted(async () => {
  try {
    await loadBases()
  } catch {
    bases.value = []
  }
  await fetchList()
})
function openCreate() {
  editing.value = { id: 0, kode: '', nama: '', jabatan: '', keterangan: '', base_id: 0 }
  formOpen.value = true
}
function openEdit(item: RequestBase) {
  editing.value = { ...item }
  formOpen.value = true
}
function closeForm() {
  if (!saving.value) {
    formOpen.value = false
    editing.value = null
  }
}
async function save() {
  const item = editing.value
  if (!item) return
  if (item.kode.trim().length < 2 || !item.nama.trim() || item.base_id <= 0) {
    toast.value = {
      kind: 'error',
      title: 'Form belum lengkap',
      description: 'Kode, nama, dan base wajib diisi.',
    }
    return
  }
  saving.value = true
  try {
    await requestWithMeta(item.id ? `/api/v1/request-bases/${item.id}` : '/api/v1/request-bases', {
      method: item.id ? 'PUT' : 'POST',
      body: {
        kode: item.kode.trim().toUpperCase(),
        nama: item.nama.trim(),
        jabatan: item.jabatan.trim(),
        keterangan: item.keterangan.trim(),
        base_id: item.base_id,
      },
    })
    toast.value = {
      kind: 'success',
      title: item.id ? 'Request base diperbarui' : 'Request base ditambahkan',
    }
    closeForm()
    await fetchList()
  } catch (error) {
    toast.value = {
      kind: 'error',
      title: 'Gagal menyimpan',
      description: error instanceof ApiError ? error.message : 'Terjadi kesalahan.',
    }
  } finally {
    saving.value = false
  }
}
async function confirmDelete() {
  if (!deleting.value) return
  deleteBusy.value = true
  try {
    await http.delete(`/api/v1/request-bases/${deleting.value.id}`)
    deleting.value = null
    toast.value = { kind: 'success', title: 'Request base dihapus' }
    await fetchList()
  } catch (error) {
    toast.value = {
      kind: 'error',
      title: 'Tidak bisa dihapus',
      description: error instanceof Error ? error.message : 'Terjadi kesalahan.',
    }
  } finally {
    deleteBusy.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold text-ink">Request Base</h1>
        <p class="mt-0.5 text-sm text-ink-muted">Kelola pengajuan turunan untuk setiap base.</p>
      </div>
      <BaseButton v-if="canManage" @click="openCreate"
        ><template #icon><Plus class="size-4" /></template>Tambah Request Base</BaseButton
      >
    </div>
    <div class="flex flex-wrap gap-3">
      <select
        v-model="baseFilter"
        class="w-full rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none sm:max-w-xs"
      >
        <option value="">Semua base</option>
        <option v-for="item in bases" :key="item.id" :value="item.id">
          {{ item.nama }}
        </option></select
      ><SearchInput v-model="search" placeholder="Cari kode atau nama..." />
    </div>
    <BaseTable
      :columns="columns"
      :rows="rows"
      :meta="meta"
      :state="state"
      :error-message="errorMessage"
      @update:page="page = $event"
      @retry="fetchList"
      ><template #cell-kode="{ row }"
        ><span class="font-mono text-xs font-medium">{{ row.kode }}</span></template
      ><template #cell-base="{ row }">{{ baseName(row.base_id) }}</template
      ><template #cell-jabatan="{ row }">{{ row.jabatan || '-' }}</template
      ><template #cell-aksi="{ row }"
        ><div v-if="canManage" class="flex justify-end gap-1">
          <button
            class="rounded-md p-1.5 text-ink-muted hover:bg-default hover:text-brand-600 focus-visible:ring-2 focus-visible:ring-brand-500"
            :aria-label="`Ubah ${row.nama}`"
            @click="openEdit(row)"
          >
            <Pencil class="size-4" /></button
          ><button
            class="rounded-md p-1.5 text-ink-muted hover:bg-status-cancelled/10 hover:text-status-cancelled focus-visible:ring-2 focus-visible:ring-status-cancelled"
            :aria-label="`Hapus ${row.nama}`"
            @click="deleting = row"
          >
            <Trash2 class="size-4" />
          </button></div></template
    ></BaseTable>
    <BaseModal
      :open="formOpen"
      :title="editing?.id ? 'Ubah Request Base' : 'Tambah Request Base'"
      :busy="saving"
      @close="closeForm"
      ><form v-if="editing" id="request-base-form" class="space-y-4" @submit.prevent="save">
        <label class="block text-sm font-medium text-ink"
          >Kode *<input
            v-model="editing.kode"
            maxlength="30"
            class="mt-1 block w-full rounded-lg border border-default bg-raised px-3 py-2 focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none" /></label
        ><label class="block text-sm font-medium text-ink"
          >Nama *<input
            v-model="editing.nama"
            maxlength="150"
            class="mt-1 block w-full rounded-lg border border-default bg-raised px-3 py-2 focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none" /></label
        ><label class="block text-sm font-medium text-ink"
          >Cari Base *<input
            v-model="baseQuery"
            placeholder="Ketik kode atau nama base"
            class="mt-1 block w-full rounded-lg border border-default bg-raised px-3 py-2 focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none"
        /></label>
        <div v-if="baseQuery" class="max-h-40 overflow-auto rounded-lg border border-default">
          <p v-if="baseAutocomplete.loading.value" class="p-3 text-sm text-ink-muted">Mencari...</p>
          <button
            v-for="option in baseAutocomplete.options.value"
            :key="option.value"
            type="button"
            class="block w-full px-3 py-2 text-left text-sm hover:bg-default focus-visible:ring-2 focus-visible:ring-brand-500"
            @click="pickBase(option)"
          >
            {{ option.kode }} — {{ option.label }}
          </button>
        </div>
        <p v-if="editing.base_id" class="text-xs text-ink-muted">Base terpilih: {{ baseName(editing.base_id) }}</p>
        <label class="block text-sm font-medium text-ink"
          >Jabatan<input
            v-model="editing.jabatan"
            maxlength="100"
            class="mt-1 block w-full rounded-lg border border-default bg-raised px-3 py-2 focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none" /></label
        ><label class="block text-sm font-medium text-ink"
          >Keterangan<textarea
            v-model="editing.keterangan"
            class="mt-1 block w-full rounded-lg border border-default bg-raised px-3 py-2 focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none"
          />
        </label>
      </form>
      <template #footer
        ><BaseButton variant="secondary" :disabled="saving" @click="closeForm">Batal</BaseButton
        ><BaseButton type="submit" form="request-base-form" :loading="saving">Simpan</BaseButton></template
      ></BaseModal
    >
    <ConfirmDialog
      :open="!!deleting"
      title="Hapus Request Base?"
      :message="`${deleting?.kode ?? ''} akan dinonaktifkan.`"
      :busy="deleteBusy"
      confirm-label="Hapus"
      @confirm="confirmDelete"
      @cancel="deleting = null"
    />
    <div class="fixed right-4 bottom-4 z-40 w-full max-w-sm">
      <ToastNotification
        v-if="toast"
        :kind="toast.kind"
        :title="toast.title"
        :description="toast.description"
        @close="toast = null"
      />
    </div>
  </div>
</template>
