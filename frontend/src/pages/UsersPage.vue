<script setup lang="ts">
/**
 * Halaman Pengguna (admin).
 *
 * Daftar semua akun + reset password. Sengaja BUKAN CRUD penuh: pembuatan dan
 * penghapusan akun tetap urusan admin sistem di sisi server. Halaman ini hanya
 * mengambil alih satu titik nyeri - mereset password orang lain, yang dulu
 * harus menyentuh database langsung.
 *
 * Reset mencabut semua refresh token target: ia keluar dari seluruh perangkat
 * dan harus login dengan password baru.
 */
import { computed, ref } from 'vue'
import { KeyRound } from 'lucide-vue-next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'

import Badge from '@/components/ui/Badge.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseModal from '@/components/ui/BaseModal.vue'
import BaseTable from '@/components/ui/BaseTable.vue'
import FormField from '@/components/ui/FormField.vue'
import SearchInput from '@/components/ui/SearchInput.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import type { Column } from '@/components/ui/tableTypes'
import type { Toast } from '@/composables/useMasterCrud'
import { ApiError, http } from '@/lib/http'
import type { AdminUser } from '@/lib/types'

/** Query key stabil untuk data pengguna. */
const userKeys = {
  all: ['users'] as const,
  list: ['users', 'list'] as const,
}

const ROLE_LABELS: Record<AdminUser['role'], string> = {
  admin: 'Administrator',
  ga_staff: 'GA Staff',
  approver: 'Approver',
  viewer: 'Viewer',
}

const queryClient = useQueryClient()
const search = ref('')
const toast = ref<Toast | null>(null)

const query = useQuery({
  queryKey: userKeys.list,
  queryFn: () => http.get<AdminUser[]>('/api/v1/users/'),
})

const rows = computed(() => {
  const items = query.data.value ?? []
  const q = search.value.trim().toLowerCase()
  if (!q) return items
  // Pencarian lokal: daftar user organisasi kecil dan seluruhnya sudah ada
  // di halaman, tidak perlu bolak-balik ke server.
  return items.filter((u) =>
    [u.username, u.email, u.full_name].some((field) => field.toLowerCase().includes(q)),
  )
})

const state = computed<'idle' | 'loading' | 'error' | 'ready'>(() => {
  if (query.error.value) return 'error'
  if (query.isPending.value) return 'loading'
  return 'ready'
})

const errorMessage = computed(() => {
  const err = query.error.value
  if (err instanceof ApiError) return err.message
  if (err) return 'Tidak bisa menghubungi server.'
  return null
})

// --- Reset password ---------------------------------------------------------

const resetTarget = ref<AdminUser | null>(null)
const newPassword = ref('')
const confirmPassword = ref('')
const submitted = ref(false)

const passwordError = computed(() => {
  if (!submitted.value) return undefined
  if (newPassword.value.length < 8) return 'Password minimal 8 karakter.'
  if (newPassword.value !== confirmPassword.value) return 'Konfirmasi password tidak sama.'
  return undefined
})

const resetMutation = useMutation({
  mutationFn: (user: AdminUser) =>
    http.post<{ id: number; username: string; reset: boolean }>(
      `/api/v1/users/${user.id}/reset-password`,
      { password: newPassword.value },
    ),
  onSuccess: (_data, user) => {
    toast.value = {
      kind: 'success',
      title: 'Password direset',
      description: `${user.username} sekarang harus login ulang di semua perangkat.`,
    }
    closeReset()
    void queryClient.invalidateQueries({ queryKey: userKeys.all })
  },
  onError: (err) => {
    toast.value = {
      kind: 'error',
      title: 'Gagal mereset password',
      description: err instanceof ApiError ? err.message : 'Tidak bisa menghubungi server.',
    }
  },
})

function openReset(user: AdminUser): void {
  resetTarget.value = user
  newPassword.value = ''
  confirmPassword.value = ''
  submitted.value = false
}

function closeReset(): void {
  if (resetMutation.isPending.value) return
  resetTarget.value = null
}

function submitReset(): void {
  submitted.value = true
  if (passwordError.value || !resetTarget.value) return
  resetMutation.mutate(resetTarget.value)
}

const columns: Column<AdminUser>[] = [
  { key: 'username', label: 'Username', width: 'w-40' },
  { key: 'email', label: 'Email', hideOnMobile: true },
  { key: 'full_name', label: 'Nama', hideOnMobile: true },
  { key: 'role', label: 'Role', width: 'w-36' },
  { key: 'is_active', label: 'Status', width: 'w-24' },
  { key: 'aksi', label: 'Aksi', align: 'right', width: 'w-16' },
]
</script>

<template>
  <div class="space-y-4">
    <!-- Header halaman -->
    <div>
      <h1 class="text-lg font-semibold tracking-tight text-ink">Pengguna</h1>
      <p class="mt-0.5 text-sm text-ink-muted">
        Daftar akun dan reset password. Reset mencabut sesi user di semua perangkat.
      </p>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <SearchInput v-model="search" placeholder="Cari username, email, atau nama..." />
    </div>

    <!-- Tabel -->
    <BaseTable
      :columns="columns"
      :rows="rows"
      :state="state"
      :error-message="errorMessage"
      @retry="query.refetch()"
    >
      <template #cell-username="{ row }">
        <span class="font-mono text-xs font-medium text-ink">{{ row.username }}</span>
      </template>

      <template #cell-email="{ row }">
        <span class="text-sm text-ink-muted">{{ row.email }}</span>
      </template>

      <template #cell-full_name="{ row }">
        <span class="font-medium">{{ row.full_name }}</span>
      </template>

      <template #cell-role="{ row }">
        <Badge variant="neutral">{{ ROLE_LABELS[row.role] }}</Badge>
      </template>

      <template #cell-is_active="{ row }">
        <Badge :variant="row.is_active ? 'finished' : 'cancelled'">
          {{ row.is_active ? 'Aktif' : 'Nonaktif' }}
        </Badge>
      </template>

      <template #cell-aksi="{ row }">
        <button
          type="button"
          class="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-default hover:text-brand-600 focus-visible:ring-2 focus-visible:ring-brand-600"
          :aria-label="`Reset password ${row.username}`"
          @click="openReset(row)"
        >
          <KeyRound class="size-4" aria-hidden="true" />
        </button>
      </template>

      <template #empty>
        <p class="text-sm font-medium text-ink">
          {{ search ? 'Tidak ada pengguna yang cocok' : 'Belum ada pengguna' }}
        </p>
        <p class="mt-1 text-sm text-ink-muted">
          {{
            search
              ? 'Coba kata kunci lain atau bersihkan pencarian.'
              : 'Pengguna dibuat oleh admin sistem di sisi server.'
          }}
        </p>
      </template>
    </BaseTable>

    <!-- Modal reset password -->
    <BaseModal
      :open="!!resetTarget"
      title="Reset Password"
      description="User akan keluar dari semua perangkat dan harus login dengan password baru."
      :busy="resetMutation.isPending.value"
      @close="closeReset()"
    >
      <form :id="'reset-password-form'" class="space-y-4" @submit.prevent="submitReset">
        <p class="rounded-lg bg-default p-3 text-sm text-ink-muted">
          Mengubah password milik
          <span class="font-medium text-ink">{{ resetTarget?.username }}</span>
          ({{ resetTarget ? ROLE_LABELS[resetTarget.role] : '' }}).
        </p>
        <FormField
          v-model="newPassword"
          label="Password baru"
          type="password"
          autocomplete="new-password"
          required
          hint="Minimal 8 karakter."
          :error="passwordError"
        />
        <FormField
          v-model="confirmPassword"
          label="Ulangi password baru"
          type="password"
          autocomplete="new-password"
          required
        />
      </form>

      <template #footer>
        <BaseButton variant="secondary" :disabled="resetMutation.isPending.value" @click="closeReset()">
          Batal
        </BaseButton>
        <BaseButton type="submit" form="reset-password-form" :loading="resetMutation.isPending.value">
          Reset Password
        </BaseButton>
      </template>
    </BaseModal>

    <!-- Toast -->
    <div class="fixed right-4 bottom-4 z-40 w-full max-w-sm">
      <Transition name="toast">
        <ToastNotification
          v-if="toast"
          :kind="toast.kind"
          :title="toast.title"
          :description="toast.description"
          @close="toast = null"
        />
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all var(--transition-fast);
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(0.5rem);
}
</style>
