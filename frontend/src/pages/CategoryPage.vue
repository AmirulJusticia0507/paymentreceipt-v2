<script setup lang="ts">
/**
 * Halaman daftar Kategori.
 *
 * Seluruh logika CRUD ada di useMasterCrud; halaman ini hanya menyusun tampilan
 * dan kolom. Pola yang sama dipakai ulang oleh halaman satuan dan kategori, jadi
 * perbedaan antar keduanya nanti tinggal beberapa baris konfigurasi.
 */
import { computed, useId } from 'vue'
import { Pencil, Plus, Trash2 } from 'lucide-vue-next'

import BaseTable from '@/components/ui/BaseTable.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseModal from '@/components/ui/BaseModal.vue'
import SearchInput from '@/components/ui/SearchInput.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import MasterEditForm from '@/components/MasterEditForm.vue'
import type { Column } from '@/components/ui/tableTypes'
import { useMasterCrud } from '@/composables/useMasterCrud'
import type { MasterItem } from '@/composables/useMasterCrud'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const crud = useMasterCrud({
  endpoint: '/api/v1/categories',
  resourceKey: 'categories',
  label: 'Kategori',
  displayField: 'kode',
})

// Id form dibuat di halaman, lalu dipakai oleh form DAN tombol Simpan di footer
// modal. Dengan begitu keduanya terhubung lewat atribut form bawaan HTML,
// tanpa template ref yang harus menyeberangi batas Teleport.
const formId = useId()

// Tipe baris memakai MasterItem dari composable, bukan tipe yang ditulis ulang
// di sini. Menyalin bentuk datanya membuat halaman ikut harus diubah setiap
// kali field di server bertambah - dan kalau lupa, errornya baru muncul saat
// build, jauh dari tempat perubahannya.
const columns = computed<Column<MasterItem>[]>(() => [
  { key: 'kode', label: 'Kode', width: 'w-28', sortable: false },
  { key: 'nama', label: 'Nama' },
  { key: 'keterangan', label: 'Keterangan', hideOnMobile: true },
  { key: 'aksi', label: 'Aksi', align: 'right', width: 'w-24' },
])

const deleteTarget = computed(() => crud.deleting.value)
const canManage = computed(() => auth.hasRole('admin'))
</script>

<template>
  <div class="space-y-4">
    <!-- Header halaman -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold tracking-tight text-ink">{{ crud.label }}</h1>
        <p class="mt-0.5 text-sm text-ink-muted">Kategori barang: ATK, elektronik, furnitur, dan sejenisnya.</p>
      </div>

      <!-- Tombol tambah hanya untuk admin. Disembunyikan, bukan hanya
           dinonaktifkan: user yang tidak punya izin akan lebih jelas dengan
           tidak melihat tombolnya sama sekali. -->
      <BaseButton v-if="canManage" @click="crud.openCreate()">
        <template #icon><Plus class="size-4" aria-hidden="true" /></template>
        {{ 'Tambah ' + crud.label }}
      </BaseButton>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <SearchInput v-model="crud.search.value" placeholder="Cari kode atau nama..." />
    </div>

    <!-- Tabel -->
    <BaseTable
      :columns="columns"
      :rows="crud.rows.value"
      :meta="crud.meta.value"
      :state="crud.state.value"
      :error-message="crud.errorMessage.value"
      @update:page="crud.page.value = $event"
      @retry="crud.retry()"
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
            @click="crud.openEdit(row)"
          >
            <Pencil class="size-4" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-status-cancelled/10 hover:text-status-cancelled"
            :aria-label="`Hapus ${row.nama}`"
            @click="crud.askDelete(row)"
          >
            <Trash2 class="size-4" aria-hidden="true" />
          </button>
        </div>
        <span v-else class="text-xs text-ink-muted/50">-</span>
      </template>

      <template #empty>
        <p class="text-sm font-medium text-ink">
          {{
            crud.search.value
              ? 'Tidak ada ' + crud.label.toLowerCase() + ' yang cocok'
              : 'Belum ada ' + crud.label.toLowerCase()
          }}
        </p>
        <p class="mt-1 text-sm text-ink-muted">
          {{
            crud.search.value
              ? 'Coba kata kunci lain atau bersihkan pencarian.'
              : 'Tambahkan kategori barang pertama untuk memulai.'
          }}
        </p>
      </template>
    </BaseTable>

    <!-- Modal form -->
    <BaseModal
      :open="crud.formOpen.value"
      :title="crud.editing.value ? 'Ubah ' + crud.label : 'Tambah ' + crud.label"
      :description="
        crud.editing.value
          ? 'Perubahan langsung berlaku untuk pengajuan baru.'
          : 'Kategori mengelompokkan jenis barang pada payment receipt.'
      "
      :busy="crud.saving.value"
      @close="crud.closeForm()"
    >
      <MasterEditForm :form-id="formId" :item="crud.editing.value" @submit="crud.save" />

      <template #footer>
        <BaseButton variant="secondary" :disabled="crud.saving.value" @click="crud.closeForm()"> Batal </BaseButton>
        <BaseButton type="submit" :form="formId" :loading="crud.saving.value"> Simpan </BaseButton>
      </template>
    </BaseModal>

    <!-- Konfirmasi hapus -->
    <ConfirmDialog
      :open="!!deleteTarget"
      :title="'Hapus ' + crud.label + '?'"
      :message="`Department ${deleteTarget?.nama ?? ''} akan dinonaktifkan dan tidak muncul lagi di daftar.`"
      :item-name="deleteTarget?.kode"
      :busy="crud.deleteBusy.value"
      confirm-label="Hapus"
      @confirm="crud.confirmDelete()"
      @cancel="crud.cancelDelete()"
    >
      <p class="mt-3 rounded-lg bg-default p-3 text-xs text-ink-muted">
        Ini soft delete: data tidak hilang permanen dan masih bisa direstore. Department yang masih dipakai transaksi
        akan ditolak.
      </p>
    </ConfirmDialog>

    <!-- Toast -->
    <div class="fixed right-4 bottom-4 z-40 w-full max-w-sm">
      <Transition name="toast">
        <ToastNotification
          v-if="crud.toast.value"
          :kind="crud.toast.value.kind"
          :title="crud.toast.value.title"
          :description="crud.toast.value.description"
          @close="crud.toast.value = null"
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
