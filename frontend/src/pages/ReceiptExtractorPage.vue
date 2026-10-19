<script setup lang="ts">
/**
 * Halaman ekstraksi data struk/faktur dengan AI.
 *
 * User mengunggah gambar struk/faktur, AI mengekstrak data terstruktur
 * (nama toko, tanggal, total, daftar barang, dst.), lalu hasilnya ditampilkan
 * untuk ditinjau dan diedit sebelum disimpan/dipakai.
 */
import { computed, ref } from 'vue'
import { Upload, FileText, Edit2, Save, Plus, AlertCircle } from 'lucide-vue-next'

import BaseButton from '@/components/ui/BaseButton.vue'
import FormField from '@/components/ui/FormField.vue'
import { useReceiptExtractor } from '@/composables/useReceiptExtractor'
import type { ExtractedReceipt } from '@/lib/types'

const { extract } = useReceiptExtractor()

// --- State ---
const file = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const previewUrl = ref<string | null>(null)
const extracting = ref(false)
const extracted = ref<ExtractedReceipt | null>(null)
const fieldsMissing = ref<string[]>([])
const editMode = ref(false)

const form = ref<ExtractedReceipt>({
  merchant_name: '',
  merchant_address: '',
  merchant_phone: '',
  invoice_number: '',
  invoice_date: '',
  purchase_date: '',
  payment_method: '',
  currency: 'IDR',
  subtotal: 0,
  tax: 0,
  discount: 0,
  total: 0,
  items: [],
  raw_text: '',
  confidence: 0,
})

const uploadError = ref<string | null>(null)

// --- Fungsi ---
async function handleExtract(): Promise<void> {
  const selected = file.value
  if (!selected) return

  // Validasi tipe file di klien sebelum kirim.
  const allowed = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp']
  if (!allowed.includes(selected.type)) {
    uploadError.value = 'Tipe berkas tidak didukung. Gunakan JPG, PNG, atau WEBP.'
    return
  }
  // Maksimal 10 MB (sama dengan batas server).
  if (selected.size > 10 * 1024 * 1024) {
    uploadError.value = 'Berkas terlalu besar. Maksimal 10 MB.'
    return
  }

  previewUrl.value = URL.createObjectURL(selected)
  uploadError.value = null
  extracted.value = null
  fieldsMissing.value = []

  try {
    extracting.value = true
    const result = await extract(selected)
    extracted.value = result
    form.value = { ...result }
    // Backend mengirim daftar field yang belum terdeteksi di meta.fields_missing.
    fieldsMissing.value = (result as any).meta?.fields_missing ?? []
  } catch (err) {
    if (err instanceof Error) {
      uploadError.value = err.message
    } else {
      uploadError.value = 'Terjadi kesalahan saat mengekstrak data.'
    }
  } finally {
    extracting.value = false
  }
}

async function handleFileSelected(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const selected = input.files?.[0]
  if (!selected) return

  file.value = selected
  await handleExtract()
}

function openFileInput(): void {
  fileInput.value?.click()
}

function removeFile(): void {
  file.value = null
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = null
  }
  extracted.value = null
  fieldsMissing.value = []
  uploadError.value = null
  form.value = {
    merchant_name: '',
    merchant_address: '',
    merchant_phone: '',
    invoice_number: '',
    invoice_date: '',
    purchase_date: '',
    payment_method: '',
    currency: 'IDR',
    subtotal: 0,
    tax: 0,
    discount: 0,
    total: 0,
    items: [],
    raw_text: '',
    confidence: 0,
  }
}

function toggleEdit(): void {
  editMode.value = !editMode.value
}

function addItem(): void {
  form.value.items.push({
    name: '',
    quantity: 0,
    unit_price: 0,
    total_price: 0,
    unit: 'pcs',
  })
}

function removeItem(index: number): void {
  form.value.items.splice(index, 1)
  recalcTotals()
}

function recalcTotals(): void {
  let subtotal = 0
  for (const item of form.value.items) {
    subtotal += item.total_price || 0
  }
  form.value.subtotal = subtotal
  form.value.total = subtotal + (form.value.tax || 0) - (form.value.discount || 0)
}

function onTotalChange(): void {
  form.value.total = (form.value.subtotal || 0) + (form.value.tax || 0) - (form.value.discount || 0)
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value || 0)
}

function hasMissingFields(): boolean {
  return fieldsMissing.value.length > 0
}

const isFormValid = computed(() => {
  return form.value.merchant_name.trim().length > 0 && form.value.total > 0 && form.value.items.length > 0
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header halaman -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold tracking-tight text-ink">Ekstraksi Struk &amp; Faktur</h1>
        <p class="mt-0.5 text-sm text-ink-muted">
          Unggah gambar struk atau faktur. AI akan menarik data pembelian secara otomatis untuk dicoret dan diedit
          sebelum disimpan.
        </p>
      </div>

      <BaseButton v-if="file && !extracting" @click="removeFile" variant="ghost" size="sm">
        <template #icon><FileText class="size-4" aria-hidden="true" /></template>
        Bersihkan
      </BaseButton>
    </div>

    <!-- Error -->
    <div
      v-if="uploadError"
      class="rounded-lg bg-status-cancelled/10 border border-status-cancelled/20 p-3 text-sm text-status-cancelled"
    >
      <AlertCircle class="size-4 mr-2" aria-hidden="true" />
      {{ uploadError }}
    </div>

    <!-- Area unggah + hasil -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
      <div class="lg:col-span-7 space-y-4">
        <!-- Drop zone (belum ada file) -->
        <div
          v-if="!file"
          class="border-2 border-dashed border-default rounded-xl p-8 text-center transition-colors hover:border-brand-500 hover:bg-brand-50/50 cursor-pointer"
          @click="openFileInput()"
          role="button"
          tabindex="0"
          @keydown.enter="openFileInput()"
          @keydown.space="openFileInput()"
          :aria-label="file ? 'Berkas sudah dipilih' : 'Pilih berkas struk/faktur'"
        >
          <Upload class="mx-auto mb-3 size-6 text-ink-muted" aria-hidden="true" />
          <p class="text-sm font-medium text-ink">Klik untuk memilih berkas</p>
          <p class="mt-1 text-xs text-ink-muted">JPG, PNG, WEBP · maksimal 10 MB</p>
          <input
            id="receipt-file"
            ref="fileInput"
            type="file"
            accept="image/jpeg,image/png,image/webp"
            class="hidden"
            @change="handleFileSelected"
          />
        </div>

        <!-- Pratinjau berkas yang dipilih -->
        <div v-else class="rounded-xl border border-default bg-raised p-4">
          <div class="flex items-start gap-3">
            <img
              v-if="previewUrl"
              :src="previewUrl"
              alt="Pratinjau struk"
              class="w-40 h-40 object-cover rounded-lg border border-default"
            />
            <div class="flex-1">
              <p class="font-medium text-ink truncate">{{ file?.name }}</p>
              <p class="text-xs text-ink-muted">{{ (file?.size / 1024).toFixed(0) }} KB · {{ file?.type }}</p>
            </div>
            <BaseButton @click="removeFile" variant="danger" size="sm" class="ml-auto" :loading="extracting">
              Hapus
            </BaseButton>
          </div>
        </div>

        <!-- Tombol ekstraksi (muncul setelah file dipilih, sebelum hasil) -->
        <div v-if="file && !extracted && !extracting" class="pt-2">
          <BaseButton @click="handleExtract" variant="primary" :loading="extracting">
            <template #icon><FileText class="size-4" aria-hidden="true" /></template>
            Ekstrak dengan AI
          </BaseButton>
        </div>

        <!-- Loading -->
        <div v-if="extracting" class="flex flex-col items-center justify-center py-8 text-ink-muted">
          <p class="animate-pulse">AI sedang membaca struk...</p>
        </div>
      </div>

      <!-- Hasil ekstraksi -->
      <div v-if="extracted" class="lg:col-span-5 space-y-4">
        <div class="rounded-xl border border-default bg-raised p-4">
          <div class="flex items-center justify-between mb-3">
            <h2 class="text-sm font-semibold text-ink">Hasil ekstraksi</h2>
            <BaseButton @click="toggleEdit" variant="ghost" size="sm">
              <template #icon><Edit2 class="size-4" aria-hidden="true" /></template>
              {{ editMode ? 'Selesai edit' : 'Edit data' }}
            </BaseButton>
          </div>

          <!-- Peringatan field yang belum terdeteksi -->
          <div
            v-if="hasMissingFields()"
            class="rounded-lg bg-status-cancelled/10 border border-status-cancelled/20 p-2 mb-3"
          >
            <AlertCircle class="size-4 mr-2 text-status-cancelled" aria-hidden="true" />
            <span class="text-xs text-status-cancelled"> Perlu diperbaiki: {{ fieldsMissing.join(', ') }} </span>
          </div>

          <div class="space-y-3">
            <div class="flex justify-between text-sm">
              <span class="text-ink-muted">Toko / Merchant</span>
              <span class="font-medium text-ink">{{ extracted.merchant_name || '-' }}</span>
            </div>
            <div class="flex justify-between text-sm">
              <span class="text-ink-muted">Nomor Faktur</span>
              <span class="font-medium text-ink">{{ extracted.invoice_number || '-' }}</span>
            </div>
            <div class="flex justify-between text-sm">
              <span class="text-ink-muted">Tanggal Pembelian</span>
              <span class="font-medium text-ink">
                {{ extracted.purchase_date || extracted.invoice_date || '-' }}
              </span>
            </div>
            <div class="flex justify-between text-sm">
              <span class="text-ink-muted">Metode Pembayaran</span>
              <span class="font-medium text-ink">{{ extracted.payment_method || '-' }}</span>
            </div>
            <div class="flex justify-between text-sm pt-2 border-t border-default/50">
              <span class="text-ink-muted font-medium">Total</span>
              <span class="font-semibold text-ink">
                {{ formatCurrency(extracted.total) }}
              </span>
            </div>
            <div class="flex justify-between text-xs text-ink-muted/70 mt-2">
              <span>Keyakinan AI</span>
              <span>{{ (extracted.confidence * 100).toFixed(0) }}%</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Form edit item barang -->
    <div v-if="extracted && editMode" class="space-y-4">
      <div class="rounded-xl border border-brand-200 bg-brand-50 p-4">
        <h3 class="text-sm font-semibold text-brand-900 flex items-center gap-2 mb-3">
          <Edit2 class="size-4" aria-hidden="true" />
          Edit hasil ekstraksi
        </h3>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormField label="Toko / Merchant" v-model="form.merchant_name" />
          <FormField label="Nomor Faktur" v-model="form.invoice_number" />
          <FormField label="Tanggal Pembelian" type="date" v-model="form.purchase_date" />
          <FormField label="Metode Pembayaran" v-model="form.payment_method" />
          <FormField label="Mata Uang" v-model="form.currency" />
          <FormField
            label="Subtotal"
            type="number"
            inputmode="decimal"
            v-model.number="form.subtotal"
            @blur="onTotalChange"
          />
          <FormField label="Pajak" type="number" inputmode="decimal" v-model.number="form.tax" @blur="onTotalChange" />
          <FormField
            label="Diskon"
            type="number"
            inputmode="decimal"
            v-model.number="form.discount"
            @blur="onTotalChange"
          />
          <FormField label="Total" type="number" inputmode="decimal" v-model.number="form.total" />
        </div>

        <div class="mt-4">
          <h4 class="text-xs font-semibold text-ink uppercase tracking-wider mb-2">Daftar Barang</h4>
          <div class="space-y-2">
            <div
              v-for="(item, idx) in form.items"
              :key="idx"
              class="grid grid-cols-1 md:grid-cols-6 gap-2 items-center"
            >
              <FormField :label="`Barang ${idx + 1}`" v-model="item.name" :hint="item.unit || '-'" />
              <FormField label="Qty" type="number" v-model.number="item.quantity" @blur="recalcTotals" />
              <FormField label="Harga Satuan" type="number" v-model.number="item.unit_price" @blur="recalcTotals" />
              <FormField label="Total" type="number" v-model.number="item.total_price" @blur="recalcTotals" />
              <FormField label="Satuan" v-model="item.unit" />
              <BaseButton
                variant="danger"
                size="sm"
                @click="removeItem(idx)"
                :aria-label="`Hapus ${item.name || 'barang'}`"
              >
                <span aria-hidden="true">×</span>
              </BaseButton>
            </div>
            <BaseButton variant="ghost" size="sm" @click="addItem">
              <template #icon><Plus class="size-4" aria-hidden="true" /></template>
              Tambah barang
            </BaseButton>
          </div>
        </div>

        <div class="mt-4 pt-4 border-t border-brand-100">
          <div class="flex justify-between text-sm font-medium text-ink">
            <span>Total akhir</span>
            <span>{{ formatCurrency(form.total) }}</span>
          </div>
        </div>

        <div class="mt-4 flex justify-end gap-2">
          <BaseButton variant="secondary" @click="toggleEdit">Batal</BaseButton>
          <BaseButton variant="primary" :disabled="!isFormValid" @click="toggleEdit"> Simpan perubahan </BaseButton>
        </div>
      </div>
    </div>

    <!-- Aksi selesai -->
    <div v-if="extracted && !editMode" class="pt-4 border-t border-default/50">
      <BaseButton variant="primary" :disabled="!isFormValid" @click="toggleEdit">
        <template #icon><Save class="size-4" aria-hidden="true" /></template>
        Gunakan data ini untuk payment receipt
      </BaseButton>
      <p class="mt-2 text-xs text-ink-muted">
        Modul pembuatan payment receipt masih dalam pengembangan (Phase 6). Saat ini hasil ekstraksi bisa disalin dan
        diedit di atas.
      </p>
    </div>
  </div>
</template>

<style scoped></style>
