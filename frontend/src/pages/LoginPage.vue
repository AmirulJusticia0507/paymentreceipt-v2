<script setup lang="ts">
/**
 * Halaman masuk.
 *
 * Catatan keamanan yang relevan untuk halaman ini:
 * - Password di-type="password" dan tidak pernah disimpan di Pinia/localStorage.
 * - Tombol "lihat password" disediakan karena menyalin password manual dari
 *   password manager sering jadi sumber error ketik, dan password manager
 *   modern sudah tidak bisa di-paste ke field type=password di beberapa browser.
 */
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { AlertCircle, Loader2, Eye, EyeOff, LogIn } from 'lucide-vue-next'

import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/lib/http'

const schema = toTypedSchema(
  z.object({
    username: z.string().min(1, 'Username wajib diisi').max(50, 'Maksimal 50 karakter'),
    password: z.string().min(1, 'Password wajib diisi').max(200, 'Maksimal 200 karakter'),
  }),
)

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const { handleSubmit, setFieldError, isSubmitting } = useForm({
  validationSchema: schema,
  initialValues: { username: '', password: '' },
})

// useField, bukan v-model biasa: vee-validate butuh nilai DAN error-nya
// terikat ke field yang sama supaya pesan validasi muncul tepat di bawah input.
const { value: username, errorMessage: usernameError } = useField<string>('username')
const { value: password, errorMessage: passwordError } = useField<string>('password')

const formError = ref<string | null>(null)
const showPassword = ref(false)

/**
 * Penyerang event submit yang eksplisit.
 *
 * Menulis @submit="handleSubmit(onSubmit)" di template tampak benar tapi salah:
 * Vue mengompilasi ekspresi BER-TANDA KURUNG sebagai statement, jadi $event
 * dibuang dan vee-validate tidak pernah menerima objek Event. Akibatnya
 * preventDefault tidak pernah dipanggil dan form jatuh ke submit GET bawaan
 * browser - password ikut masuk query string URL.
 *
 * Di sini preventDefault dipanggil eksplisit oleh kode kita sendiri, jadi
 * perilaku ini tidak bergantung pada cara compiler memahami ekspresi template
 * maupun pada implementasi internal library.
 */
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
        // Error per-field dari server dipetakan ke form supaya user melihat
        // pesan di tempat yang sedang diketik, bukan hanya di atas.
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
        formError.value = 'Tidak bisa menghubungi server. Periksa koneksi Anda.'
      } else {
        formError.value = 'Terjadi kesalahan tak terduga. Coba lagi.'
      }
      // Password dikosongkan supaya tidak tertinggal di DOM kalau form
      // di-render ulang. Username sengaja dipertahankan - user tidak perlu
      // mengetiknya lagi setelah gagal.
      password.value = ''
    })
}

// Demo credentials sengaja ditampilkan: akun ini hanya ada di seed
// development.sql. Di production tidak ada kredensial demo seperti ini.
const isDemo = import.meta.env.DEV
</script>

<template>
  <main class="flex min-h-dvh items-center justify-center bg-surface px-4 py-12">
    <div class="w-full max-w-sm">
      <!-- Brand -->
      <div class="mb-8 text-center">
        <div class="mx-auto mb-4 flex size-12 items-center justify-center rounded-xl bg-brand-600 text-white shadow-sm">
          <LogIn class="size-6" aria-hidden="true" />
        </div>
        <h1 class="text-xl font-semibold tracking-tight text-ink">Payment Receipt</h1>
        <p class="mt-1 text-sm text-ink-muted">Sistem Purchase Requisition</p>
      </div>

      <!-- Card -->
      <div class="card p-6 shadow-sm">
        <form novalidate class="space-y-4" @submit="onSubmitEvent">
          <!-- Error global -->
          <div
            v-if="formError"
            role="alert"
            class="flex gap-2.5 rounded-lg border border-status-cancelled/30 bg-status-cancelled/10 p-3 text-sm text-status-cancelled"
          >
            <AlertCircle class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
            <p>{{ formError }}</p>
          </div>

          <!-- Username -->
          <div class="space-y-1.5">
            <label for="username" class="block text-sm font-medium text-ink">Username</label>
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
              class="block w-full rounded-lg border border-default bg-raised px-3 py-2 text-sm text-ink placeholder:text-ink-muted/60 transition-[var(--transition-fast)] focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none aria-[invalid=true]:border-status-cancelled"
              placeholder="Masukkan username"
            />
            <p v-if="usernameError" id="username-error" class="text-xs text-status-cancelled">
              {{ usernameError }}
            </p>
          </div>

          <!-- Password -->
          <div class="space-y-1.5">
            <label for="password" class="block text-sm font-medium text-ink">Password</label>
            <div class="relative">
              <input
                id="password"
                v-model="password"
                name="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                required
                :aria-invalid="!!passwordError"
                :aria-describedby="passwordError ? 'password-error' : undefined"
                class="block w-full rounded-lg border border-default bg-raised px-3 py-2 pr-10 text-sm text-ink placeholder:text-ink-muted/60 transition-[var(--transition-fast)] focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none aria-[invalid=true]:border-status-cancelled"
                placeholder="Masukkan password"
              />
              <button
                type="button"
                class="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-ink-muted transition-colors hover:text-ink"
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

          <button
            type="submit"
            :disabled="isSubmitting"
            class="flex w-full items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm transition-[var(--transition-fast)] hover:bg-brand-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500 disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Loader2 v-if="isSubmitting" class="size-4 animate-spin" aria-hidden="true" />
            <span>{{ isSubmitting ? 'Memproses...' : 'Masuk' }}</span>
          </button>
        </form>
      </div>

      <!-- Kredensial demo, hanya di development -->
      <div v-if="isDemo" class="mt-4 rounded-lg border border-default bg-raised/60 p-3 text-xs text-ink-muted">
        <p class="mb-1.5 font-medium text-ink">Akun demo (lokal saja)</p>
        <ul class="space-y-0.5">
          <li><code class="font-mono">admin</code> / <code class="font-mono">admin123</code></li>
          <li><code class="font-mono">ga_staff</code> / <code class="font-mono">ga_staff123</code></li>
          <li><code class="font-mono">approver</code> / <code class="font-mono">approver123</code></li>
          <li><code class="font-mono">viewer</code> / <code class="font-mono">approver123</code></li>
        </ul>
      </div>
    </div>
  </main>
</template>
