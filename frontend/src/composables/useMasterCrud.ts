/**
 * Composable CRUD generik untuk halaman tabel master.
 *
 * Tiga halaman (department, satuan, kategori) punya perilaku yang identik:
 * daftar + pencarian + create + update + delete + toast. Menulis ulang pola itu
 * di tiga halaman berarti tiga tempat untuk memperbaiki bug yang sama, jadi
 * logikanya ditarik ke sini.
 *
 * Yang TIDAK dilakukan di sini: menentukan endpoint, bentuk kolom tabel, atau
 * field form. Itu perbedaan antar halaman dan harus terlihat di halamannya.
 */

import { computed, ref, watch } from 'vue'
import { useQuery, useQueryClient } from '@tanstack/vue-query'

import { ApiError, requestWithMeta } from '@/lib/http'
import type { PageMeta } from '@/lib/types'

/** Bentuk data tabel master - sama untuk department, satuan, dan kategori. */
export interface MasterItem {
  id: number
  kode: string
  nama: string
  keterangan?: string
}

export interface MasterFormValues {
  kode: string
  nama: string
  keterangan?: string
}

/** Toast yang sedang tampil, atau null. */
export interface Toast {
  kind: 'success' | 'error' | 'info'
  title: string
  description?: string
}

export interface MasterCrudConfig {
  /** Prefix path API, mis. "/api/v1/departments". */
  endpoint: string
  /** Nama sumber data di cache, mis. "departments". */
  resourceKey: string
  /** Label tunggal untuk pesan, mis. "Department". */
  label: string
  /** Field yang dipakai untuk mengidentifikasi baris di pesan konfirmasi. */
  displayField?: 'kode' | 'nama'
}

export function useMasterCrud(config: MasterCrudConfig) {
  const queryClient = useQueryClient()

  const search = ref('')
  const page = ref(1)
  const perPage = ref(20)

  /** Item yang sedang diedit, atau null kalau mode create. */
  const editing = ref<MasterItem | null>(null)
  const formOpen = ref(false)
  const saving = ref(false)

  /** Baris yang menunggu konfirmasi hapus. */
  const deleting = ref<MasterItem | null>(null)
  const deleteBusy = ref(false)

  const toast = ref<Toast | null>(null)

  const displayField = config.displayField ?? 'nama'

  const listKey = computed(() => [
    config.resourceKey,
    'list',
    { q: search.value, page: page.value, per_page: perPage.value },
  ] as const)

  const query = useQuery({
    queryKey: listKey,
    queryFn: async () => {
      const params = new URLSearchParams({
        page: String(page.value),
        per_page: String(perPage.value),
      })
      if (search.value.trim()) params.set('q', search.value.trim())

      const res = await requestWithMeta<MasterItem[]>(`${config.endpoint}?${params}`)
      return { items: res.data, meta: res.meta as PageMeta }
    },
  })

  // Ganti kata kunci pencarian selalu kembali ke halaman 1. Kalau tidak, user
  // yang sedang di halaman 5 akan melihat "tidak ada hasil" padahal ada -
  // hasil cuma ada di halaman 1.
  watch(search, () => {
    page.value = 1
  })

  const rows = computed<MasterItem[]>(() => query.data.value?.items ?? [])
  const meta = computed(() => query.data.value?.meta)

  /** Keadaan tabel untuk BaseTable. */
  const state = computed<'idle' | 'loading' | 'error' | 'ready'>(() => {
    if (query.error.value) return 'error'
    if (query.isPending.value) return 'loading'
    return 'ready'
  })

  const errorMessage = computed(() => {
    const err = query.error.value
    if (err instanceof ApiError) return err.message
    if (err) return 'Tidak bisa menghubungi server.'
    return null
  })

  function invalidate(): void {
    void queryClient.invalidateQueries({ queryKey: [config.resourceKey] })
  }

  // --- Form ------------------------------------------------------------

  function openCreate(): void {
    editing.value = null
    formOpen.value = true
  }

  function openEdit(item: MasterItem): void {
    editing.value = item
    formOpen.value = true
  }

  function closeForm(): void {
    formOpen.value = false
    editing.value = null
  }

  async function save(values: MasterFormValues): Promise<boolean> {
    saving.value = true
    try {
      if (editing.value) {
        await requestWithMeta(`${config.endpoint}/${editing.value.id}`, {
          method: 'PUT',
          body: values,
        })
        toast.value = { kind: 'success', title: `${config.label} diperbarui` }
      } else {
        await requestWithMeta(config.endpoint, { method: 'POST', body: values })
        toast.value = { kind: 'success', title: `${config.label} ditambahkan` }
      }
      closeForm()
      invalidate()
      return true
    } catch (err) {
      toast.value = {
        kind: 'error',
        title: 'Gagal menyimpan',
        description: err instanceof ApiError ? err.message : 'Terjadi kesalahan.',
      }
      // Dibiarkan false supaya form tetap terbuka dan isian user tidak hilang.
      return false
    } finally {
      saving.value = false
    }
  }

  // --- Hapus -----------------------------------------------------------

  function askDelete(item: MasterItem): void {
    deleting.value = item
  }

  function cancelDelete(): void {
    deleting.value = null
  }

  async function confirmDelete(): Promise<void> {
    if (!deleting.value) return

    deleteBusy.value = true
    try {
      await requestWithMeta(`${config.endpoint}/${deleting.value.id}`, {
        method: 'DELETE',
      })
      toast.value = { kind: 'success', title: `${config.label} dihapus` }
      deleting.value = null
      invalidate()
    } catch (err) {
      // 409 di sini berarti data masih dipakai. Pesannya dari server sudah
      // menjelaskan detailnya, jadi cukup diteruskan apa adanya.
      toast.value = {
        kind: 'error',
        title: 'Tidak bisa dihapus',
        description: err instanceof ApiError ? err.message : 'Terjadi kesalahan.',
      }
    } finally {
      deleteBusy.value = false
    }
  }

  return {
    // label dipakai template untuk judul halaman, tombol, dan pesan. Dikembalikan
    // dari sini supaya teks itu punya satu sumber kebenaran - menulis label
    // langsung di template pernah membuat tombol menampilkan "undefined".
    label: config.label,
    // data
    rows,
    meta,
    state,
    errorMessage,
    search,
    page,
    // form
    formOpen,
    editing,
    saving,
    save,
    openCreate,
    openEdit,
    closeForm,
    // delete
    deleting,
    deleteBusy,
    askDelete,
    cancelDelete,
    confirmDelete,
    // misc
    toast,
    displayField,
    retry: () => query.refetch(),
  }
}