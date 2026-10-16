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