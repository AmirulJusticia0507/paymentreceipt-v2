<script setup lang="ts">
/**
 * Modal kunci biometric. Ditampilkan sebagai overlay full-screen ketika
 * sesi valid tapi pengguna harus memvalidasi sidik jari terlebih dahulu.
 */
import { ref } from 'vue'
import { Fingerprint, Lock } from 'lucide-vue-next'
import BaseButton from '@/components/ui/BaseButton.vue'
import { useBiometric } from '@/composables/useBiometric'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  locked: boolean
}>()

const emit = defineEmits<{
  unlock: []
  logout: []
}>()

const auth = useAuthStore()
const { authenticate, enroll, hasEnrolled } = useBiometric()
const unlocking = ref(false)
const enrolling = ref(false)

async function handleUnlock(): Promise<void> {
  unlocking.value = true
  try {
    const ok = await authenticate()
    if (ok) {
      unlocking.value = false
      emit('unlock')
    } else {
      unlocking.value = false
    }
  } catch {
    unlocking.value = false
  }
}

async function handleEnroll(): Promise<void> {
  enrolling.value = true
  try {
    await enroll()
    auth.biometricEnrolled = true
    enrolling.value = false
    emit('unlock')
  } catch {
    enrolling.value = false
  }
}

async function handleLogout(): Promise<void> {
  await auth.logout()
  emit('logout')
}
</script>

<template>
  <transition name="fade">
    <div v-if="locked" class="fixed inset-0 z-[100] bg-white dark:bg-gray-900 flex flex-col items-center justify-center">
      <div class="text-center p-6">
        <div class="mb-6">
          <Fingerprint class="size-32 text-brand-500 dark:text-brand-400" aria-hidden="true" />
        </div>
        <h2 class="text-lg font-semibold text-ink dark:text-white mb-2">
          {{ hasEnrolled() ? 'Verifikasi Sidik Jari' : 'Daftarkan Sidik Jari' }}
        </h2>
        <p class="text-ink-muted dark:text-gray-400 mb-8 text-sm max-w-xs mx-auto leading-relaxed">
          {{ hasEnrolled()
            ? 'Sentuh sensor sidik jari untuk membuka aplikasi.'
            : 'Anda belum mendaftarkan sidik jari. Daftarkan terlebih dahulu untuk membuka aplikasi dengan cepat.' }}
        </p>

        <div class="flex flex-col gap-3 w-full max-w-sm">
          <BaseButton
            v-if="hasEnrolled()"
            :loading="unlocking"
            full
            @click="handleUnlock"
          >
            <template #icon><Lock class="size-4" aria-hidden="true" /></template>
            Verifikasi dengan Sidik Jari
          </BaseButton>
          <BaseButton
            v-else
            :loading="enrolling"
            variant="secondary"
            full
            @click="handleEnroll"
          >
            <template #icon><Fingerprint class="size-4" aria-hidden="true" /></template>
            Daftarkan Sidik Jari
          </BaseButton>

          <button
            @click="handleLogout"
            class="w-full py-2 text-xs text-ink-muted dark:text-gray-500 underline underline-offset-4 hover:text-brand-600 dark:hover:text-brand-400"
          >
            Atau masuk dengan password
          </button>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
