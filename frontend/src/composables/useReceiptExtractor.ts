/**
 * Composable untuk ekstraksi data struk/faktur melalui AI.
 *
 * Endpoint menerima multipart/form-data (bukan JSON), jadi tidak bisa pakai
 * `http` dari lib/http yang selalu set Content-Type: application/json. Kita
 * pakai fetch langsung dengan FormData, tapi tetap kirim credentials
 * (cookie HttpOnly) supaya endpoint terlindungi auth bisa dipakai.
 */
import type { ExtractedReceipt, ExtractReceiptResponse } from '@/lib/types'

const BASE_URL = import.meta.env.VITE_API_URL ?? ''

/** Error yang dikembalikan saat ekstraksi gagal. */
export class ReceiptExtractError extends Error {
  status: number
  code: string

  constructor(message: string, status: number, code: string) {
    super(message)
    this.name = 'ReceiptExtractError'
    this.status = status
    this.code = code
  }
}

export function useReceiptExtractor() {
  /**
   * Unggah berkas struk/faktur dan terima data terstruktur dari AI.
   *
   * File yang didukung: JPG, PNG, WEBP. Maksimal 10 MB (dibatasi server).
   */
  async function extract(file: File): Promise<ExtractedReceipt> {
    const formData = new FormData()
    formData.append('file', file)

    const res = await fetch(`${BASE_URL}/api/v1/ai/extract-receipt`, {
      method: 'POST',
      credentials: 'include',
      body: formData,
    })

    if (!res.ok) {
      let detail = `Ekstraksi gagal (status ${res.status})`
      let code = 'extract_failed'
      try {
        const problem = await res.json()
        detail = problem.detail || detail
        code = problem.code || code
      } catch {
        // Respons bukan JSON, abaikan.
      }
      throw new ReceiptExtractError(detail, res.status, code)
    }

    const result: ExtractReceiptResponse = await res.json()
    return result.data
  }

  return { extract }
}
