<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import { ApiError, http } from '@/lib/http'

interface DepartmentOption {
  id: number
  kode: string
  nama: string
}

const router = useRouter()
const departments = ref<DepartmentOption[]>([])
const username = ref('')
const password = ref('')
const phone = ref('')
const departmentId = ref<number | ''>('')
const loadingDepartments = ref(true)
const submitting = ref(false)
const error = ref('')
const success = ref('')

onMounted(async () => {
  try {
    departments.value = await http.get<DepartmentOption[]>('/api/v1/auth/signup/departments')
  } catch {
    error.value = 'Daftar department belum dapat dimuat. Coba muat ulang halaman.'
  } finally {
    loadingDepartments.value = false
  }
})

async function submit(): Promise<void> {
  error.value = ''
  success.value = ''
  submitting.value = true
  try {
    const result = await http.post<{ message: string }>('/api/v1/auth/signup', {
      username: username.value,
      password: password.value,
      phone: phone.value,
      department_id: Number(departmentId.value),
    })
    success.value = result.message
    password.value = ''
    window.setTimeout(() => void router.push('/login'), 1800)
  } catch (cause) {
    error.value = cause instanceof ApiError ? cause.message : 'Pendaftaran gagal. Coba lagi.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="flex min-h-dvh items-center justify-center bg-surface px-4 py-10">
    <section class="w-full max-w-md rounded-2xl border border-default bg-raised p-6 shadow-lg sm:p-8">
      <h1 class="text-xl font-bold text-ink">Daftar akun requester</h1>
      <p class="mt-1 text-sm text-ink-muted">Akun akan aktif setelah disetujui admin.</p>
      <p v-if="error" role="alert" class="mt-4 rounded-lg bg-status-cancelled/10 p-3 text-sm text-status-cancelled">{{ error }}</p>
      <p v-if="success" role="status" class="mt-4 rounded-lg bg-status-finished/10 p-3 text-sm text-ink">{{ success }}</p>
      <form class="mt-6 space-y-4" @submit.prevent="submit">
        <label class="block space-y-1 text-sm font-medium text-ink">Username
          <input v-model.trim="username" required minlength="3" maxlength="50" autocomplete="username" pattern="[A-Za-z0-9._-]+" class="mt-1 w-full rounded-lg border border-default bg-surface px-3 py-2 focus-visible:outline-2 focus-visible:outline-brand-500" />
        </label>
        <label class="block space-y-1 text-sm font-medium text-ink">Password
          <input v-model="password" required minlength="8" maxlength="200" type="password" autocomplete="new-password" class="mt-1 w-full rounded-lg border border-default bg-surface px-3 py-2 focus-visible:outline-2 focus-visible:outline-brand-500" />
        </label>
        <label class="block space-y-1 text-sm font-medium text-ink">Nomor HP
          <input v-model.trim="phone" required minlength="8" maxlength="30" type="tel" autocomplete="tel" pattern="[0-9+() -]+" class="mt-1 w-full rounded-lg border border-default bg-surface px-3 py-2 focus-visible:outline-2 focus-visible:outline-brand-500" />
        </label>
        <label class="block space-y-1 text-sm font-medium text-ink">Department
          <select v-model="departmentId" required :disabled="loadingDepartments || departments.length === 0" class="mt-1 w-full rounded-lg border border-default bg-surface px-3 py-2 focus-visible:outline-2 focus-visible:outline-brand-500 disabled:opacity-60">
            <option value="" disabled>{{ loadingDepartments ? 'Memuat department…' : 'Pilih department' }}</option>
            <option v-for="department in departments" :key="department.id" :value="department.id">{{ department.kode }} — {{ department.nama }}</option>
          </select>
        </label>
        <button type="submit" :disabled="submitting || loadingDepartments || departments.length === 0" class="w-full rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white transition-colors hover:bg-brand-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500 disabled:cursor-not-allowed disabled:opacity-60">{{ submitting ? 'Mengirim…' : 'Daftar' }}</button>
      </form>
      <p class="mt-5 text-center text-sm text-ink-muted">Sudah punya akun? <RouterLink to="/login" class="font-semibold text-brand-600 hover:underline focus-visible:outline-brand-500">Masuk</RouterLink></p>
    </section>
  </main>
</template>
