/** Bentuk error dari backend (RFC 9457 Problem Details). */
export interface Problem {
  type: string
  title: string
  status: number
  detail: string
  instance?: string
  code?: string
  errors?: FieldProblem[]
  retry_after?: number
}

export interface FieldProblem {
  field: string
  message: string
  rule?: string
}

/** Peran pengguna..Values harus sama dengan enum user_role di database. */
export type UserRole = 'admin' | 'ga_staff' | 'approver' | 'viewer'

export interface User {
  id: number
  username: string
  email: string
  full_name: string
  role: UserRole
  last_login_at?: string
}

/**
 * Profil user untuk halaman admin (Pengguna).
 *
 * Memuat kolom yang tidak dipakai login biasa - status aktif dan timestamp.
 * Seperti User, tidak pernah memuat password_hash: query backend sudah
 * menyaringnya di database.
 */
export interface AdminUser extends User {
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface LoginResponse {
  user: User
  expires_at: string
}

/** Metadata paginasi, sesuai dengan struct Pagination di backend. */
export interface PageMeta {
  page: number
  per_page: number
  total: number
  total_pages: number
}

/** Hasil ekstraksi satu barang dari struk/faktur. */
export interface ExtractedItem {
  name: string
  quantity: number
  unit_price: number
  total_price: number
  unit: string
}

/** Hasil ekstraksi struk/faktur dari AI. Semua field bisa kosong kalau
 * tidak terdeteksi di dokumen. */
export interface ExtractedReceipt {
  merchant_name: string
  merchant_address: string
  merchant_phone: string
  invoice_number: string
  invoice_date: string
  purchase_date: string
  payment_method: string
  currency: string
  subtotal: number
  tax: number
  discount: number
  total: number
  items: ExtractedItem[]
  raw_text: string
  confidence: number
}

/** Respons POST /api/v1/ai/extract-receipt. */
export interface ExtractReceiptResponse {
  data: ExtractedReceipt
  meta: {
    fields_missing: string[]
  }
}