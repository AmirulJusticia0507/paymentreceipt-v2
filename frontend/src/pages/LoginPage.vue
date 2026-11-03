<script setup lang="ts">
/**
 * Halaman masuk modern dengan branding adaptif Susi Air dan pemilih tema.
 *
 * Catatan keamanan:
 * - Password di-type="password" dan tidak pernah disimpan di Pinia/localStorage.
 * - Tombol "lihat password" disediakan untuk mempermudah pengecekan tanpa membuka risiko.
 */
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import {
  AlertCircle,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  Lock,
  Moon,
  Sun,
  User,
} from 'lucide-vue-next'

import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { ApiError } from '@/lib/http'

const schema = toTypedSchema(
  z.object({
    username: z.string().min(1, 'Username wajib diisi').max(50, 'Maksimal 50 karakter'),
    password: z.string().min(1, 'Password wajib diisi').max(200, 'Maksimal 200 karakter'),
  }),
)

const auth = useAuthStore()
const theme = useThemeStore()
const router = useRouter()
const route = useRoute()

const { handleSubmit, setFieldError, isSubmitting } = useForm({
  validationSchema: schema,
  initialValues: { username: '', password: '' },
})

const { value: username, errorMessage: usernameError } = useField<string>('username')
const { value: password, errorMessage: passwordError } = useField<string>('password')

const formError = ref<string | null>(null)
const showPassword = ref(false)

function onSubmitEvent(event: Event): void {
  event.preventDefault()
  void handleSubmit(onSubmit)(event)
}

function onSubmit(values: { username: string; password: string }) {
  formError.value = null

  return auth
    .login(values.username, values.password)
    .then(() => {
      const redirect = route.query.redirect
      const target = typeof redirect === 'string' && redirect.startsWith('/') ? redirect : '/'
      router.push(target)
    })
    .catch((err: unknown) => {
      if (err instanceof ApiError) {
        let mapped = false
        for (const field of ['username', 'password'] as const) {
          const message = err.fieldMessage(field)
          if (message) {
            setFieldError(field, message)
            mapped = true
          }
        }
        if (!mapped) formError.value = err.message
      } else if (err instanceof TypeError) {
        formError.value = 'Tidak bisa menghubungi server API. Pastikan server backend sedang aktif.'
      } else {
        formError.value = 'Terjadi kesalahan tak terduga. Coba lagi.'
      }
      password.value = ''
    })
}

function fillDemoAccount(user: string, pass: string): void {
  username.value = user
  password.value = pass
}

const isDemo = import.meta.env.DEV
</script>

<template>
  <main
    class="relative flex min-h-dvh items-center justify-center overflow-hidden bg-surface px-4 py-12 transition-colors duration-200"
  >
    <!-- Background subtle ambient glow blobs -->
    <div
      class="pointer-events-none absolute -top-32 -right-32 size-96 rounded-full bg-brand-500/10 blur-3xl dark:bg-brand-500/15"
      aria-hidden="true"
    />
    <div
      class="pointer-events-none absolute -bottom-32 -left-32 size-96 rounded-full bg-brand-600/10 blur-3xl dark:bg-brand-600/15"
      aria-hidden="true"
    />

    <!-- Tombol Dark Mode di sudut kanan atas -->
    <header class="absolute top-4 right-4 z-20 sm:top-6 sm:right-6">
      <button
        type="button"
        class="inline-flex items-center gap-2 rounded-xl border border-default bg-raised/80 px-3.5 py-2 text-xs font-semibold text-ink shadow-xs backdrop-blur-md transition-all hover:bg-default hover:shadow-sm focus-visible:outline-2 focus-visible:outline-brand-500"
        :aria-label="theme.theme === 'dark' ? 'Ganti ke mode terang' : 'Ganti ke mode gelap'"
        @click="theme.toggle()"
      >
        <Sun v-if="theme.theme === 'dark'" class="size-4 text-amber-400 transition-transform hover:rotate-45" />
        <Moon v-else class="size-4 text-brand-600 transition-transform hover:-rotate-12" />
        <span>{{ theme.theme === 'dark' ? 'Mode Terang' : 'Mode Gelap' }}</span>
      </button>
    </header>

    <div class="relative z-10 w-full max-w-sm">
      <!-- Brand Logo & Title -->
      <div class="mb-8 text-center">
        <div
          class="inline-flex items-center justify-center rounded-2xl bg-white p-3.5 shadow-md ring-1 ring-black/5 transition-transform duration-200 hover:scale-105 dark:bg-white dark:ring-white/20"
        >
          <img
            src="/susi-air-logo.png"
            alt="Susi Air Logo"
            class="h-12 w-auto max-w-[240px] object-contain sm:h-14"
          />
        </div>
        <h1 class="mt-4 text-2xl font-extrabold tracking-tight text-ink">Payment Receipt</h1>
        <p class="mt-1 text-sm font-semibold text-brand-600 dark:text-brand-400">
          PT. Asi Pudjiastuti Aviation
        </p>
        <p class="text-xs text-ink-muted">General Affairs Department</p>
      </div>

      <!-- Login Card -->
      <div class="card rounded-2xl border border-default/80 bg-raised/95 p-6 shadow-xl backdrop-blur-md sm:p-7">
        <form novalidate class="space-y-4" @submit="onSubmitEvent">
          <!-- Error global -->
          <div
            v-if="formError"
            role="alert"
            class="flex items-start gap-2.5 rounded-xl border border-status-cancelled/30 bg-status-cancelled/10 p-3 text-sm text-status-cancelled"
          >
            <AlertCircle class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
            <p class="leading-snug">{{ formError }}</p>
          </div>

          <!-- Username Field -->
          <div class="space-y-1.5">
            <label for="username" class="block text-xs font-semibold uppercase tracking-wider text-ink-muted">
              Username
            </label>
            <div class="relative">
              <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-ink-muted">
                <User class="size-4" aria-hidden="true" />
              </span>
              <input
                id="username"
                v-model="username"
                name="username"
                type="text"
                autocomplete="username"
                autocapitalize="none"
                spellcheck="false"
                required
                :aria-invalid="!!usernameError"
                :aria-describedby="usernameError ? 'username-error' : undefined"
                class="block w-full rounded-xl border border-default bg-surface py-2.5 pr-3 pl-9.5 text-sm text-ink placeholder:text-ink-muted/50 transition-all focus:border-brand-500 focus:ring-3 focus:ring-brand-500/20 focus:outline-none aria-[invalid=true]:border-status-cancelled"
                placeholder="Masukkan username Anda"
              />
            </div>
            <p v-if="usernameError" id="username-error" class="text-xs text-status-cancelled">
              {{ usernameError }}
            </p>
          </div>

          <!-- Password Field -->
          <div class="space-y-1.5">
            <label for="password" class="block text-xs font-semibold uppercase tracking-wider text-ink-muted">
              Password
            </label>
            <div class="relative">
              <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-ink-muted">
                <Lock class="size-4" aria-hidden="true" />
              </span>
              <input
                id="password"
                v-model="password"
                name="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                required
                :aria-invalid="!!passwordError"
                :aria-describedby="passwordError ? 'password-error' : undefined"
                class="block w-full rounded-xl border border-default bg-surface py-2.5 pr-10 pl-9.5 text-sm text-ink placeholder:text-ink-muted/50 transition-all focus:border-brand-500 focus:ring-3 focus:ring-brand-500/20 focus:outline-none aria-[invalid=true]:border-status-cancelled"
                placeholder="Masukkan password Anda"
              />
              <button
                type="button"
                class="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-ink-muted transition-colors hover:text-ink focus-visible:outline-brand-500"
                :aria-label="showPassword ? 'Sembunyikan password' : 'Tampilkan password'"
                @click="showPassword = !showPassword"
              >
                <EyeOff v-if="showPassword" class="size-4" aria-hidden="true" />
                <Eye v-else class="size-4" aria-hidden="true" />
              </button>
            </div>
            <p v-if="passwordError" id="password-error" class="text-xs text-status-cancelled">
              {{ passwordError }}
            </p>
          </div>

          <!-- Submit Button -->
          <button
            type="submit"
            :disabled="isSubmitting"
            class="mt-2 flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-brand-600 to-brand-500 py-2.5 px-4 text-sm font-semibold text-white shadow-md shadow-brand-500/20 transition-all hover:from-brand-700 hover:to-brand-600 hover:shadow-lg hover:shadow-brand-500/30 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500 active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Loader2 v-if="isSubmitting" class="size-4 animate-spin" aria-hidden="true" />
            <span>{{ isSubmitting ? 'Memproses Masuk...' : 'Masuk ke Sistem' }}</span>
          </button>
        </form>
      </div>

      <!-- Kredensial demo (Quick Fill) -->
      <div
        v-if="isDemo"
        class="mt-5 rounded-xl border border-default/70 bg-raised/70 p-3.5 text-xs text-ink-muted backdrop-blur-xs"
      >
        <div class="mb-2 flex items-center gap-1.5 font-semibold text-ink">
          <KeyRound class="size-3.5 text-brand-600 dark:text-brand-400" />
          <span>Akun Demo Cepat (Klik untuk isi)</span>
        </div>
        <div class="grid grid-cols-2 gap-1.5">
          <button
            type="button"
            class="rounded-lg border border-default bg-surface/80 px-2 py-1.5 text-left transition-colors hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-950/50 dark:hover:text-brand-300"
            @click="fillDemoAccount('admin', 'admin123')"
          >
            <p class="font-medium text-ink">admin</p>
            <p class="text-[10px] text-ink-muted">admin123 (Administrator)</p>
          </button>
          <button
            type="button"
            class="rounded-lg border border-default bg-surface/80 px-2 py-1.5 text-left transition-colors hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-950/50 dark:hover:text-brand-300"
            @click="fillDemoAccount('ga_staff', 'ga_staff123')"
          >
            <p class="font-medium text-ink">ga_staff</p>
            <p class="text-[10px] text-ink-muted">ga_staff123 (GA Staff)</p>
          </button>
          <button
            type="button"
            class="rounded-lg border border-default bg-surface/80 px-2 py-1.5 text-left transition-colors hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-950/50 dark:hover:text-brand-300"
            @click="fillDemoAccount('approver', 'approver123')"
          >
            <p class="font-medium text-ink">approver</p>
            <p class="text-[10px] text-ink-muted">approver123 (Manager)</p>
          </button>
          <button
            type="button"
            class="rounded-lg border border-default bg-surface/80 px-2 py-1.5 text-left transition-colors hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-950/50 dark:hover:text-brand-300"
            @click="fillDemoAccount('viewer', 'approver123')"
          >
            <p class="font-medium text-ink">viewer</p>
            <p class="text-[10px] text-ink-muted">approver123 (Viewer)</p>
          </button>
        </div>
      </div>
    </div>
  </main>
</template>
