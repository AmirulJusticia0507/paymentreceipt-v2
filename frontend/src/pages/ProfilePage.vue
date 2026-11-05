<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Save, UserRound } from 'lucide-vue-next'
import { useMutation } from '@tanstack/vue-query'

import BaseButton from '@/components/ui/BaseButton.vue'
import FormField from '@/components/ui/FormField.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import type { Toast } from '@/composables/useMasterCrud'
import { ApiError, http } from '@/lib/http'
import type { User } from '@/lib/types'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const fullName = ref(auth.user?.full_name ?? '')
const submitted = ref(false)
const toast = ref<Toast | null>(null)

watch(
  () => auth.user?.full_name,
  (value) => {
    fullName.value = value ?? ''
  },
)

const fullNameError = computed(() => {
  if (!submitted.value) return undefined
  const value = fullName.value.trim()
  if (!value) return 'Nama lengkap wajib diisi.'
  if ([...value].length > 150) return 'Nama lengkap maksimal 150 karakter.'
  return undefined
})

const mutation = useMutation({
  mutationFn: () => http.patch<User>('/api/v1/auth/me', { full_name: fullName.value }),
  onSuccess: (user) => {
    auth.setUser(user)
    toast.value = { kind: 'success', title: 'Profil diperbarui', description: 'Nama lengkap berhasil disimpan.' }
  },
  onError: (error) => {
    toast.value = {
      kind: 'error',
      title: 'Profil gagal diperbarui',
      description: error instanceof ApiError ? error.message : 'Tidak bisa menghubungi server.',
    }
  },
})

function submit(): void {
  submitted.value = true
  if (fullNameError.value) return
  mutation.mutate()
}
</script>

<template>
  <section class="mx-auto max-w-xl space-y-6">
    <div>
      <div class="flex items-center gap-2 text-brand-600">
        <UserRound class="size-5" aria-hidden="true" />
        <p class="text-sm font-semibold">Akun</p>
      </div>
      <h1 class="mt-1 text-2xl font-bold text-ink">Profil Saya</h1>
      <p class="mt-1 text-sm text-ink-muted">Ubah nama lengkap yang tampil pada aplikasi.</p>
    </div>

    <form class="card space-y-5 p-5" @submit.prevent="submit">
      <FormField
        v-model="fullName"
        label="Nama lengkap"
        placeholder="Masukkan nama lengkap"
        autocomplete="name"
        maxlength="150"
        required
        :error="fullNameError"
        hint="Nama ini dipakai sebagai nama tampilan akun Anda."
      />
      <div class="flex justify-end">
        <BaseButton type="submit" :loading="mutation.isPending.value">
          <template #icon><Save class="size-4" aria-hidden="true" /></template>
          Simpan perubahan
        </BaseButton>
      </div>
    </form>

    <div class="fixed right-4 bottom-4 z-40 w-full max-w-sm">
      <ToastNotification
        v-if="toast"
        :kind="toast.kind"
        :title="toast.title"
        :description="toast.description"
        @close="toast = null"
      />
    </div>
  </section>
</template>
