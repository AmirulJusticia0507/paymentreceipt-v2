<script setup lang="ts">
import { ref } from 'vue'
import { Sparkles, Trash2, Send } from 'lucide-vue-next'

import PageHeader from '@/components/ui/PageHeader.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import Badge from '@/components/ui/Badge.vue'
import FormField from '@/components/ui/FormField.vue'
import SearchInput from '@/components/ui/SearchInput.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import BaseModal from '@/components/ui/BaseModal.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonTable from '@/components/ui/SkeletonTable.vue'
import BaseTable from '@/components/ui/BaseTable.vue'
import type { Column } from '@/components/ui/tableTypes'

// State untuk demo FormField & Search
const inputValue = ref('')
const inputError = ref('Format isian tidak sesuai contoh')
const searchKeyword = ref('')

// State untuk demo Modal & Confirm
const showConfirm = ref(false)
const showModal = ref(false)

// State untuk demo Table
interface SampleItem {
  id: number
  kode: string
  nama: string
  status: 'pending' | 'finished' | 'processed'
}

const sampleColumns: Column<SampleItem>[] = [
  { key: 'kode', label: 'Kode', sortable: true },
  { key: 'nama', label: 'Nama Komponen' },
  { key: 'status', label: 'Status' },
]

const sampleRows: SampleItem[] = [
  { id: 1, kode: 'CMP-01', nama: 'Header Navigasi', status: 'finished' },
  { id: 2, kode: 'CMP-02', nama: 'Dialog Konfirmasi', status: 'processed' },
  { id: 3, kode: 'CMP-03', nama: 'Filter Pencarian', status: 'pending' },
]
</script>

<template>
  <div class="space-y-8 pb-12">
    <PageHeader
      title="Playground Komponen UI"
      description="Katalog dan pengujian komponen UI generik untuk sistem Payment Receipt v2."
    />

    <!-- Bagian 1: Tombol -->
    <section class="rounded-xl border border-default bg-surface p-6 shadow-xs">
      <h2 class="text-base font-semibold text-ink">1. BaseButton</h2>
      <p class="mt-1 text-sm text-ink-muted">Variasi ukuran, status pemuatan, dan tema tombol.</p>

      <div class="mt-4 flex flex-wrap items-center gap-3">
        <BaseButton variant="primary">
          <template #icon><Sparkles class="size-4" /></template>
          Primary Button
        </BaseButton>
        <BaseButton variant="secondary">Secondary</BaseButton>
        <BaseButton variant="danger">
          <template #icon><Trash2 class="size-4" /></template>
          Danger
        </BaseButton>
        <BaseButton variant="ghost">Ghost Button</BaseButton>
        <BaseButton variant="primary" loading>Loading</BaseButton>
        <BaseButton variant="secondary" disabled>Disabled</BaseButton>
        <BaseButton variant="primary" size="sm">Small Size</BaseButton>
      </div>
    </section>

    <!-- Bagian 2: Badge Status -->
    <section class="rounded-xl border border-default bg-surface p-6 shadow-xs">
      <h2 class="text-base font-semibold text-ink">2. Badge Status</h2>
      <p class="mt-1 text-sm text-ink-muted">Label semantik sesuai siklus hidup transaksi payment receipt.</p>

      <div class="mt-4 flex flex-wrap items-center gap-3">
        <Badge variant="pending">Pending</Badge>
        <Badge variant="checked">Checked</Badge>
        <Badge variant="processed">Processed</Badge>
        <Badge variant="finished">Finished</Badge>
        <Badge variant="cancelled">Cancelled</Badge>
        <Badge variant="neutral">Neutral Label</Badge>
      </div>
    </section>

    <!-- Bagian 3: Input Form & Pencarian -->
    <section class="rounded-xl border border-default bg-surface p-6 shadow-xs">
      <h2 class="text-base font-semibold text-ink">3. FormField & SearchInput</h2>
      <p class="mt-1 text-sm text-ink-muted">Input dengan keterbacaan aksesibilitas, pesan error, dan debounce pencarian.</p>

      <div class="mt-4 grid gap-4 md:grid-cols-2">
        <FormField
          v-model="inputValue"
          label="Nama Pengguna"
          placeholder="Masukkan nama..."
          hint="Gunakan nama lengkap sesuai identitas."
          required
        />

        <FormField
          label="Email Karyawan"
          model-value="format-salah@"
          :error="inputError"
          placeholder="nama@susiair.com"
          required
        />

        <div class="md:col-span-2">
          <label class="mb-1.5 block text-sm font-medium text-ink">SearchInput (Debounce 300ms)</label>
          <SearchInput
            v-model="searchKeyword"
            placeholder="Cari sesuatu di sini..."
          />
          <p class="mt-2 text-xs text-ink-muted">
            Nilai kata kunci saat ini: <span class="font-mono font-medium text-ink">{{ searchKeyword || '(kosong)' }}</span>
          </p>
        </div>
      </div>
    </section>

    <!-- Bagian 4: Modal & Dialog -->
    <section class="rounded-xl border border-default bg-surface p-6 shadow-xs">
      <h2 class="text-base font-semibold text-ink">4. Modal & ConfirmDialog</h2>
      <p class="mt-1 text-sm text-ink-muted">Komponen dialog interaktif untuk aksi sensitif dan formulir popup.</p>

      <div class="mt-4 flex flex-wrap gap-3">
        <BaseButton variant="secondary" @click="showModal = true">
          Buka BaseModal
        </BaseButton>
        <BaseButton variant="danger" @click="showConfirm = true">
          Buka ConfirmDialog
        </BaseButton>
      </div>

      <!-- Preview Modal Biasa -->
      <BaseModal
        :open="showModal"
        title="Contoh Dialog Modal"
        @close="showModal = false"
      >
        <p class="text-sm text-ink">
          Ini adalah konten di dalam BaseModal. Dilengkapi animasi transisi, tombol close, dan penutup backdrop.
        </p>
        <template #footer>
          <BaseButton variant="secondary" @click="showModal = false">Tutup</BaseButton>
          <BaseButton variant="primary" @click="showModal = false">
            <template #icon><Send class="size-4" /></template>
            Simpan
          </BaseButton>
        </template>
      </BaseModal>

      <!-- Preview ConfirmDialog -->
      <ConfirmDialog
        :open="showConfirm"
        title="Hapus Data Pengujian?"
        message="Data yang dihapus tidak dapat dipulihkan kembali ke kondisi semula."
        item-name="PR-2026-SAMPLE"
        @cancel="showConfirm = false"
        @confirm="showConfirm = false"
      />
    </section>

    <!-- Bagian 5: Tabel & Loading States -->
    <section class="rounded-xl border border-default bg-surface p-6 shadow-xs">
      <h2 class="text-base font-semibold text-ink">5. BaseTable & State Pemuatan</h2>
      <p class="mt-1 text-sm text-ink-muted">Penanganan state tabel: data normal, skeleton loading, dan empty state.</p>

      <div class="mt-4 space-y-6">
        <div>
          <h3 class="mb-2 text-sm font-medium text-ink">Data Tabel (Normal)</h3>
          <BaseTable
            :columns="sampleColumns"
            :rows="sampleRows"
            state="ready"
          >
            <template #cell-status="{ value }">
              <Badge :variant="(value as any)">{{ value }}</Badge>
            </template>
          </BaseTable>
        </div>

        <div>
          <h3 class="mb-2 text-sm font-medium text-ink">Skeleton Table (Loading)</h3>
          <SkeletonTable :columns="3" :rows="2" />
        </div>

        <div>
          <h3 class="mb-2 text-sm font-medium text-ink">EmptyState</h3>
          <div class="rounded-lg border border-dashed border-default p-4">
            <EmptyState
              title="Belum ada transaksi"
              message="Silakan buat transaksi payment receipt baru untuk memulai pengajuan."
            />
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
