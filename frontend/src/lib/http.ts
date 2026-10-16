/**
 * HTTP client dengan auto-refresh token.
 *
 * Token auth disimpan di cookie HttpOnly, jadi JavaScript TIDAK bisa
 * membacanya. Konsekuensinya: setiap request memakai `credentials: 'include'`
 * dan browser yang mengurus pencocokan cookie-nya.
 *
 * Yang kita lakukan di sini hanya menangani SKENARIO: saat server membalas 401
 * karena access token kedaluwarsa, kita coba tukar refresh token sekali saja,
 * lalu mengulang request aslinya.
 */

import type { FieldProblem, Problem } from './types'

const BASE_URL = import.meta.env.VITE_API_URL ?? ''

/** Callback yang dipanggil ketika sesi benar-benar habis. */
let onUnauthorized: (() => void) | null = null

export function setUnauthorizedHandler(handler: () => void): void {
  onUnauthorized = handler
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly problem: Problem
  readonly fields: Record<string, string>

  constructor(problem: Problem, fallbackMessage: string) {
    super(problem.detail || fallbackMessage)
    this.name = 'ApiError'
    this.status = problem.status
    // code bisa kosong di Problem yang dibuat manual (mis. proxy error),
    // jadi default ke string kosong, bukan undefined.
    this.code = problem.code ?? ''
    this.problem = problem
    // Ringkas errors[] jadi peta field -> pesan supaya form tinggal menulis
    // error.messageOf('username').
    this.fields = Object.fromEntries((problem.errors ?? []).map((e: FieldProblem) => [e.field, e.message]))
  }

  /** Message untuk sebuah field, atau undefined kalau field itu valid. */
  fieldMessage(field: string): string | undefined {
    return this.fields[field]
  }

  get isUnauthorized(): boolean {
    return this.status === 401
  }

  get isForbidden(): boolean {
    return this.status === 403
  }

  get isNotFound(): boolean {
    return this.status === 404
  }

  get isConflict(): boolean {
    return this.status === 409
  }

  get isValidation(): boolean {
    return this.status === 422
  }

  get isRateLimited(): boolean {
    return this.status === 429
  }
}

/**
 * Refresh yang sedang berjalan.
 *
 * Kalau lima request gagal bersamaan, kita hanya mengirim SATU permintaan
 * refresh dan-fifth itu menunggu hasilnya. Tanpa ini, access token yang
 * kedaluwarsa akan memicu badai refresh yang justru memperburuk kondisi.
 */
let refreshInFlight: Promise<boolean> | null = null

async function refreshSession(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight

  refreshInFlight = (async () => {
    try {
      const res = await fetch(`${BASE_URL}/api/v1/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
      })
      return res.ok
    } catch {
      // Jaringan mati atau server mati - anggap sesi tidak bisa diselamatkan.
      return false
    } finally {
      // Bersihkan SETELAH promise selesai, supaya request lain yang datang
      // selama ini masih memakai promise yang sama (bukan membuat yang baru).
      queueMicrotask(() => {
        refreshInFlight = null
      })
    }
  })()

  return refreshInFlight
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown
  /** Lewati auto-refresh (dipakai oleh endpoint auth sendiri). */
  skipRefresh?: boolean
  /** Sinyal pembatalan, supaya component yang unmount tidak menorphan request. */
  signal?: AbortSignal
}

async function toProblem(res: Response): Promise<Problem> {
  try {
    return (await res.json()) as Problem
  } catch {
    // Server membalas non-JSON (mis. proxy error atau panic HTML).
    return {
      type: 'about:blank',
      title: 'Kesalahan',
      status: res.status,
      detail: `Permintaan gagal dengan status ${res.status}`,
    }
  }
}

async function execute(path: string, options: RequestOptions, isRetry = false): Promise<Response> {
  const { body, skipRefresh, headers, ...rest } = options

  const finalHeaders = new Headers(headers)
  if (body !== undefined) {
    finalHeaders.set('Content-Type', 'application/json')
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    ...rest,
    headers: finalHeaders,
    credentials: 'include',
    body: body === undefined ? undefined : JSON.stringify(body),
  })

  // 401 pada endpoint yang bukan auth = access token kemungkinan kedaluwarsa.
  // Coba refresh SEKALI. isRetry mencegah loop tak berujung.
  if (res.status === 401 && !skipRefresh && !isRetry && !path.includes('/auth/')) {
    const refreshed = await refreshSession()
    if (refreshed) {
      return execute(path, options, true)
    }
    // Refresh gagal berarti sesi benar-benar habis.
    onUnauthorized?.()
  }

  return res
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const res = await execute(path, options)

  if (res.status === 204) {
    return undefined as T
  }

  if (!res.ok) {
    throw new ApiError(await toProblem(res), `Permintaan ke ${path} gagal`)
  }

  const contentType = res.headers.get('Content-Type') ?? ''
  if (!contentType.includes('application/json')) {
    return (await res.text()) as T
  }

  const payload = await res.json()
  // Backend membungkus data dalam { data, meta }.
  return (payload as { data: T }).data
}

export const http = {
  get: <T>(path: string, options: RequestOptions = {}) =>
    request<T>(path, { ...options, method: 'GET' }),

  post: <T>(path: string, body?: unknown, options: RequestOptions = {}) =>
    request<T>(path, { ...options, method: 'POST', body }),

  put: <T>(path: string, body?: unknown, options: RequestOptions = {}) =>
    request<T>(path, { ...options, method: 'PUT', body }),

  patch: <T>(path: string, body?: unknown, options: RequestOptions = {}) =>
    request<T>(path, { ...options, method: 'PATCH', body }),

  delete: <T>(path: string, options: RequestOptions = {}) =>
    request<T>(path, { ...options, method: 'DELETE' }),
}

/** Versi yang mengembalikan respons penuh - untuk endpoint dengan metadata. */
export async function requestWithMeta<T>(
  path: string,
  options: RequestOptions = {},
): Promise<{ data: T; meta: unknown }> {
  const res = await execute(path, options)
  if (!res.ok) {
    throw new ApiError(await toProblem(res), `Permintaan ke ${path} gagal`)
  }
  return (await res.json()) as { data: T; meta: unknown }
}

/** Membangun query string, membuang nilai kosong. */
export function qs(params: Record<string, string | number | boolean | undefined | null>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    search.set(key, String(value))
  }
  const str = search.toString()
  return str ? `?${str}` : ''
}
