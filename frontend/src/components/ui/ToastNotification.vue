<script setup lang="ts">
/**
 * Toast notifikasi.
 *
 * Muncul otomatis lalu hilang. Munculnya WAJIB: aksi yang mengubah data tanpa
 * umpan balik membuat user tidak yakin apakah berhasil atau gagal, dan itu
 * mendorong mereka menekan ulang - yang pada create berarti data dobel.
 */
import { onBeforeUnmount } from 'vue'
import { AlertCircle, CheckCircle2, Info, X } from 'lucide-vue-next'

export type ToastKind = 'success' | 'error' | 'info'

interface Props {
  kind: ToastKind
  title: string
  description?: string
  /** Lama tampil dalam milidetik. Default 4 detik, error 6 detik. */
  duration?: number
}

const props = defineProps<Props>()

const emit = defineEmits<{ close: [] }>()

let timer: ReturnType<typeof setTimeout> | undefined

function dismiss(): void {
  if (timer) clearTimeout(timer)
  emit('close')
}

// Error butuh waktu baca lebih lama: pesannya sering berisi instruksi apa
// yang harus dilakukan berikutnya.
const ms = props.duration ?? (props.kind === 'error' ? 6000 : 4000)
timer = setTimeout(dismiss, ms)

onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
})

const ICONS = {
  success: CheckCircle2,
  error: AlertCircle,
  info: Info,
} as const

const STYLES: Record<ToastKind, string> = {
  success: 'border-status-finished/40 bg-status-finished/10 text-status-finished',
  error: 'border-status-cancelled/40 bg-status-cancelled/10 text-status-cancelled',
  info: 'border-brand-500/40 bg-brand-500/10 text-brand-600 dark:text-brand-300',
}
</script>

<template>
  <!-- role=status untuk sukses, role=alert untuk error: toast sukses tidak boleh
       memutus pembacaan screen reader, sedangkan toast error perlu diumumkan. -->
  <div
    :role="kind === 'error' ? 'alert' : 'status'"
    :aria-live="kind === 'error' ? 'assertive' : 'polite'"
    class="card flex items-start gap-3 p-4 shadow-lg"
    :class="STYLES[kind]"
  >
    <component :is="ICONS[kind]" class="mt-0.5 size-5 shrink-0" aria-hidden="true" />

    <div class="min-w-0 flex-1">
      <p class="text-sm font-medium">{{ title }}</p>
      <p v-if="description" class="mt-0.5 text-sm opacity-90">{{ description }}</p>
    </div>

    <button
      type="button"
      class="shrink-0 rounded p-0.5 opacity-60 transition-opacity hover:opacity-100"
      aria-label="Tutup notifikasi"
      @click="dismiss"
    >
      <X class="size-4" aria-hidden="true" />
    </button>
  </div>
</template>