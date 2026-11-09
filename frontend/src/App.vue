<script setup lang="ts">
/**
 * Root component. RouterView ada di layout, jadi di sini cukup passthrough.
 * BiometricLock ditampilkan sebagai overlay full-screen ketika sesi valid
 * tapi pengguna harus memvalidasi sidik jari terlebih dahulu.
 */
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import BiometricLock from '@/components/ui/BiometricLock.vue'

const auth = useAuthStore()
const router = useRouter()

const biometricLocked = computed(() => auth.biometricLocked && auth.biometricEnrolled)

function routerPushLogin(): void {
  void router.push({ name: 'login' })
}
</script>

<template>
  <div>
    <BiometricLock
      v-if="biometricLocked"
      :locked="true"
      @unlock="auth.unlockBiometric()"
      @logout="routerPushLogin()"
    />
    <RouterView v-if="auth.initialized && !biometricLocked" />
  </div>
</template>
