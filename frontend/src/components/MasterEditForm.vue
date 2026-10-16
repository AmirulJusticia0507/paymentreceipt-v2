<script setup lang="ts">
/**
 * Form create/edit untuk tabel master.
 *
 * Dipakai department, satuan, dan kategori. Field keterangan disembunyikan saat
 * `withKeterangan` false, karena tabel satuan memang tidak punya kolom itu -
 * menampilkan input yang diam-diam tidak disimpan adalah pembohongan UI.
 */
import { watch } from 'vue'
import { useField, useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'

import FormField from '@/components/ui/FormField.vue'
import type { MasterFormValues, MasterItem } from '@/composables/useMasterCrud'

const props = defineProps<{
  item: MasterItem | null
  withKeterangan?: boolean
  /** Error per-field dari server (422), dikunci ke nama field. */
  serverErrors?: Record<string, string>
  /**
   * Id untuk elemen <form>.
   *
   * Tombol Simpan berada di footer modal, di luar elemen ini. Daripada
   * mengambil referensi component dari luar lalu memanggil metodenya - yang rapuh
   * kalau ref belum terisi atau component-nya berada di dalam Teleport -
   * assosiasi form bawaan HTML lebih andal: tombol cukup menunjuk form lewat
   * atribut form. Bonusnya, Enter di dalam input tetap memicu submit.
   */
  formId?: string
}>()

const emit = defineEmits<{ submit: [values: MasterFormValues] }>()

const schema = toTypedSchema(
  z.object({
    kode: z
      .string()
      .trim()
      .min(2, 'Kode minimal 2 karakter')
      .max(30, 'Kode maksimal 30 karakter')
      .regex(/^[A-Za-z0-9-]+$/, 'Kode hanya boleh huruf, angka, dan tanda hubung'),
    nama: z
      .string()
      .trim()
      .min(1, 'Nama wajib diisi')
      .max(100, 'Nama maksimal 100 karakter'),
    keterangan: z.string().trim().max(255, 'Keterangan maksimal 255 karakter'),
  }),
)

const { handleSubmit, setFieldError } = useForm({
  validationSchema: schema,
  initialValues: { kode: '', nama: '', keterangan: '' },
})

const { value: kode, errorMessage: kodeError } = useField<string>('kode')
const { value: nama, errorMessage: namaError } = useField<string>('nama')
const { value: keterangan, errorMessage: keteranganError } = useField<string>('keterangan')

// Form diisi ulang tiap kali modal dibuka dengan item yang berbeda. Inilah yang
// membedakan mode create dan edit - bukan dua form terpisah.
watch(
  () => props.item,
  (item) => {
    kode.value = item?.kode ?? ''
    nama.value = item?.nama ?? ''
    keterangan.value = item?.keterangan ?? ''
  },
  { immediate: true },
)

// Error dari server dipetakan ke field yang sama supaya pesan muncul di tempat
// yang sedang diketik, bukan melayang di atas form.
watch(
  () => props.serverErrors,
  (errs) => {
    if (!errs) return
    for (const [field, message] of Object.entries(errs)) {
      if (field === 'kode' || field === 'nama' || field === 'keterangan') {
        setFieldError(field, message)
      }
    }
  },
)

function onSubmit(values: { kode: string; nama: string; keterangan: string }): void {
  emit('submit', {
    kode: values.kode,
    nama: values.nama,
    keterangan: props.withKeterangan === false ? undefined : values.keterangan,
  })
}


</script>

<template>
  <!--
    Tag <form> dipakai supaya Enter di dalam input memicu submit. Penangan
    submit-nya memanggil preventDefault secara eksplisit: menulis
    @submit="handleSubmit(onSubmit)" di template compilation Vue memperlakukan
    ekspresi bertanda kurung sebagai statement sehingga $event dibuang dan form
    jatuh ke submit GET bawaan browser.
  -->
  <form :id="formId" novalidate class="space-y-4" @submit.prevent="handleSubmit(onSubmit)">
    <FormField
      v-model="kode"
      label="Kode"
      :error="kodeError"
      hint="Huruf besar, angka, atau tanda hubung. Contoh: GA"
      required
      maxlength="30"
      autocomplete="off"
      autocapitalize="characters"
      placeholder="GA"
    />

    <FormField
      v-model="nama"
      label="Nama"
      :error="namaError"
      required
      maxlength="100"
      :placeholder="item ? item.nama : 'General Affairs'"
    />

    <FormField
      v-if="withKeterangan !== false"
      v-model="keterangan"
      label="Keterangan"
      :error="keteranganError"
      maxlength="255"
      placeholder="Opsional"
    />
  </form>
</template>