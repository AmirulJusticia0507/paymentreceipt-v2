/**
 * Composable untuk biometric fingerprint lewat WebAuthn Platform Authenticator
 * (TouchID, FaceID, Windows Hello, sidik jari Android, dll).
 *
 * Digunakan sebagai "local app lock": setelah sesi JWT dari server valid,
 * pengguna harus memvalidasi sidik jari lewat browser sebelum UI aplikasi
 * dibuka. Ini TIDAK menggantikan autentikasi server — cookie HttpOnly
 * tetap menjadi sumber kebenaran sesi.
 */
import { computed } from 'vue'

const STORAGE_KEY = 'pr_biometric_credential'

interface StoredCredential {
  id: string
  publicKey: string
}

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary)
}

function base64ToArrayBuffer(base64: string): Uint8Array {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}

export function useBiometric() {
  const supported = computed(() =>
    typeof window !== 'undefined' &&
    'credentials' in navigator &&
    typeof (PublicKeyCredential as any).isUserVerifyingPlatformAuthenticatorAvailable === 'function'
  )

  /** Browser + platform authenticator (sensor sidik jari) tersedia atau tidak. */
  async function isAvailable(): Promise<boolean> {
    if (!supported.value) return false
    try {
      return await (PublicKeyCredential as any).isUserVerifyingPlatformAuthenticatorAvailable()
    } catch {
      return false
    }
  }

  /** Daftarkan sidik jari pengguna (create credential). Dipanggil sekali. */
  async function enroll(): Promise<void> {
    if (!(await isAvailable())) {
      throw new Error('Biometric tidak didukung di browser ini')
    }

    const challenge = new Uint8Array(32)
    crypto.getRandomValues(challenge)

    const credential = await navigator.credentials.create({
      publicKey: {
        challenge,
        rp: { name: 'Payment Receipt v2', id: location.hostname },
        user: {
          id: new Uint8Array(16),
          name: location.hostname,
          displayName: 'Pengguna',
        },
        pubKeyCredParams: [{ alg: -7, type: 'public-key' }],
        authenticatorSelection: {
          authenticatorAttachment: 'platform',
          userVerification: 'required',
        },
        timeout: 60_000,
      },
    }) as PublicKeyCredential

    const response = credential.response as AuthenticatorAttestationResponse
    const publicKeyBuffer = response.getPublicKey?.() ?? new ArrayBuffer(0)
    const stored: StoredCredential = {
      id: arrayBufferToBase64(credential.rawId),
      publicKey: arrayBufferToBase64(publicKeyBuffer),
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(stored))
  }

  /** Verifikasi sidik jari (get assertion). Jika resolve, sidik jari valid. */
  async function authenticate(): Promise<boolean> {
    const storedRaw = localStorage.getItem(STORAGE_KEY)
    if (!storedRaw) return false

    const stored: StoredCredential = JSON.parse(storedRaw)
    const challenge = new Uint8Array(32)
    crypto.getRandomValues(challenge)

    await navigator.credentials.get({
      publicKey: {
        challenge,
        allowCredentials: [
          {
            id: base64ToArrayBuffer(stored.id) as BufferSource,
            type: 'public-key',
          },
        ],
        userVerification: 'required',
        timeout: 60_000,
      },
    })

    return true
  }

  /** Sudah ada credential sidik jari yang tersimpan secara lokal. */
  function hasEnrolled(): boolean {
    return !!localStorage.getItem(STORAGE_KEY)
  }

  /** Hapus credential sidik jari (mis. saat logout). */
  function remove(): void {
    localStorage.removeItem(STORAGE_KEY)
  }

  return { supported, isAvailable, enroll, authenticate, hasEnrolled, remove }
}
