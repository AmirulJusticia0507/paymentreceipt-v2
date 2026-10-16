/** Store Pinia untuk sesi autentikasi. */

import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { ApiError, http } from '@/lib/http'
import type { LoginResponse, User, UserRole } from '@/lib/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const pending = ref(false)

  const isAuthenticated = computed(() => user.value !== null)

  const role = computed<UserRole | null>(() => user.value?.role ?? null)

  const isAdmin = computed(() => role.value === 'admin')

  /**
   * Loading session saat aplikasi dibuka.
   *
   * Cookie HttpOnly masih ada setelah refresh halaman, tapi kita tidak tahu
   * apakah masih valid. Jadi saat start, tanyakan ke /auth/me. Endpoint itu
   * aman dipanggil: kalau cookie sudah tidak berlaku, http client akan coba
   * refresh sekali lalu membiarkan kita logout dengan rapi.
   */
  async function restore(): Promise<void> {
    if (initialized.value) return

    try {
      user.value = await http.get<User>('/api/v1/auth/me', { skipRefresh: true })
    } catch (err) {
      // 401 di sini berarti memang tidak ada sesi - itu kondisi normal,
      // bukan error. Error lain (network, 500) juga berarti belum ada user
      // yang bisa ditampilkan, jadi perlakukan sama.
      if (!(err instanceof ApiError)) {
        console.warn('gagal memulihkan sesi:', err)
      }
      user.value = null
    } finally {
      initialized.value = true
    }
  }

  async function login(username: string, password: string): Promise<void> {
    pending.value = true
    try {
      const res = await http.post<LoginResponse>(
        '/api/v1/auth/login',
        { username, password },
        { skipRefresh: true },
      )
      user.value = res.user
      initialized.value = true
    } finally {
      pending.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      await http.post('/api/v1/auth/logout')
    } catch {
      // Logout yang gagal TIDAK boleh menahan user dari keluar. Cookie akan
      // kedaluwarsa sendiri, dan memaksa user tetap login karena logout
      // gagal adalah pengalaman yang lebih buruk daripada cookie yatim.
    } finally {
      user.value = null
    }
  }

  function hasRole(...roles: UserRole[]): boolean {
    return user.value !== null && roles.includes(user.value.role)
  }

  /** Dipanggil http client saat refresh gagal dan sesi benar-benar habis. */
  function clear(): void {
    user.value = null
    initialized.value = true
  }

  return {
    user,
    initialized,
    pending,
    isAuthenticated,
    role,
    isAdmin,
    login,
    logout,
    restore,
    hasRole,
    clear,
  }
})
