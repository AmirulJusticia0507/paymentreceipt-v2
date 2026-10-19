/**
 * Tipe bersama untuk komponen tabel.
 *
 * Dipisah dari komponen .vue supaya tipe bisa di-import tanpa menarik
 * sekaligus seluruh definisi komponen.
 */

/** Satu kolom tabel. */
export interface Column<T> {
  /** Harus sama dengan nama field di API. */
  key: string
  label: string
  /** Server Sorting harus didukung field ini. Kalau tidak, header tidak sortable. */
  sortable?: boolean
  align?: 'left' | 'center' | 'right'
  /** Kelas Tailwind untuk lebar kolom. */
  width?: string
  /** Sembunyikan kolom ini di layar kecil - untuk tabel yang padat. */
  hideOnMobile?: boolean
  /** Render kustom. Kalau diisi, `key` dipakai untuk key v-for saja. */
  cell?: (row: T) => string | number
}

/** Metadata paginasi dari backend. */
export interface PageMeta {
  page: number
  per_page: number
  total: number
  total_pages: number
}

/** Keadaan pemuatan tabel. */
export type TableState = 'idle' | 'loading' | 'error' | 'ready'
