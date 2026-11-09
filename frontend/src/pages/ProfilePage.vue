<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { Save, UserRound, Fingerprint } from 'lucide-vue-next'
import { useMutation } from '@tanstack/vue-query'

import BaseButton from '@/components/ui/BaseButton.vue'
import FormField from '@/components/ui/FormField.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import type { Toast } from '@/composables/useMasterCrud'
import { ApiError, http } from '@/lib/http'
import type { User } from '@/lib/types'
import { useAuthStore } from '@/stores/auth'
import { useBiometric } from '@/composables/useBiometric'

const auth = useAuthStore()
const { isAvailable, enroll } = useBiometric()
const fullName = ref(auth.user?.full_name ?? '')
const submitted = ref(false)
const toast = ref<Toast | null>(null)

const biometricEnrolling = ref(false)
const biometricSupported = ref<boolean | null>(null)

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

/** Kunci biometric didukung browser atau tidak. */
async function checkBiometricSupport(): Promise<void> {
  biometricSupported.value = await isAvailable()
}

/** Aktifkan atau nonaktifkan kunci sidik jari. */
async function toggleBiometric(enable: boolean): Promise<void> {
  if (enable) {
    biometricEnrolling.value = true
    try {
      await enroll()
      auth.setBiometricEnrolled(true)
      auth.lockBiometric()
      toast.value = {
        kind: 'success',
        title: 'Sidik jari terdaftar',
        description: 'Kunci biometric aktif. Anda akan diminta verifikasi sidik jari saat membuka aplikasi.',
      }
    } catch (err: any) {
      biometricEnrolling.value = false
      toast.value = {
        kind: 'error',
        title: 'Gagal mendaftarkan sidik jari',
        description: err?.message ?? 'Coba lagi nanti.',
      }
      return
    }
    biometricEnrolling.value = false
  } else {
    auth.clearBiometric()
    toast.value = {
      kind: 'info',
      title: 'Sidik jari dinonaktifkan',
      description: 'Kunci biometric dimatikan.',
    }
  }
}

onMounted(() => {
  checkBiometricSupport()
})
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

    <!-- Form nama lengkap -->
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

    <!-- Kunci biometric -->
    <div class="card space-y-4 p-5">
      <div class="flex items-center justify-between">
        <div>
          <div class="flex items-center gap-2 text-brand-600 mb-1">
            <Fingerprint class="size-4" aria-hidden="true" />
            <p class="text-sm font-semibold">Kunci Sidik Jari</p>
          </div>
          <p class="text-xs text-ink-muted">
            Buka aplikasi dengan sensor sidik jari (TouchID, FaceID, Windows Hello, atau sidik jari Android).
            <br />
            <span class="block mt-1">
              {{ biometricSupported === null ? 'Memeriksa dukungan…' : biometricSupported ? 'Didukung di browser ini.' : 'Tidak didukung di browser ini.' }}
            </span>
          </p>
        </div>
        <label for="biometric-toggle" class="relative inline-flex cursor-pointer items-center">
          <input
            id="biometric-toggle"
            type="checkbox"
            role="switch"
            :aria-checked="auth.biometricEnrolled"
            :checked="auth.biometricEnrolled"
            @change="toggleBiometric(($event.target as HTMLInputElement).checked)"
            :disabled="biometricSupported !== true || biometricEnrolling"
            class="sr-only peer"
          />
          <div
            class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-brand-600"
          />
        </label>
      </div>

      <p v-if="biometricSupported === false" class="text-xs text-ink-muted mt-1">
        Browser atau perangkat ini tidak mendukung autentikasi biometric (WebAuthn).
      </p>
    </div>

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
