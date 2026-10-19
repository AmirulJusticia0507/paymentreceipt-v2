<script setup lang="ts">
/**
 * Dialog modal dengan focus trap dan penutupan lewat Escape.
 *
 * Modal tanpa penanganan khusus punya beberapa masalah kecil yang mahal: Tab
 * keluar dari dialog ke halaman di belakang, Escape tidak menutup, atau klik
 * di luar tidak menutup. Yang terakhir bisa merusak data, karena user mengira
 * sedang membatalkan lalu klik di tempat lain.
 */
import { onBeforeUnmount, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'

type Size = 'sm' | 'md' | 'lg'

interface Props {
  open: boolean
  title: string
  description?: string
  /** Cegah penutupan - dipakai saat request sedang berjalan. */
  busy?: boolean
  size?: Size
}

const props = withDefaults(defineProps<Props>(), {
  busy: false,
  size: 'md',
})

const emit = defineEmits<{ close: [] }>()

const panel = ref<HTMLElement | null>(null)
/** Elemen yang aktif sebelum modal dibuka, untuk mengembalikan fokus. */
let previouslyFocused: HTMLElement | null = null

function requestClose(): void {
  // Saat request berjalan, menutup modal berarti user kehilangan konteks
  // tentang apa yang sedang disimpan.
  if (props.busy) return
  emit('close')
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    event.stopPropagation()
    requestClose()
    return
  }

  // Focus trap: Tab tidak boleh keluar dari panel dialog.
  if (event.key !== 'Tab' || !panel.value) return

  const focusable = panel.value.querySelectorAll<HTMLElement>(
    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
  )
  if (focusable.length === 0) return

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const active = document.activeElement

  if (event.shiftKey && active === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && active === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      previouslyFocused = document.activeElement as HTMLElement | null
      document.addEventListener('keydown', onKeydown)
      // Scroll latar dikunci supaya halaman di belakang tidak ikut bergeser.
      document.body.style.overflow = 'hidden'
      // Fokus masuk ke panel supaya screen reader langsung membaca isi dialog,
      // bukan elemen yang masih terfokus di belakangnya.
      requestAnimationFrame(() => panel.value?.focus())
    } else {
      document.removeEventListener('keydown', onKeydown)
      document.body.style.overflow = ''
      previouslyFocused?.focus()
    }
  },
)

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})

// Record butuh key yang bukan union dengan undefined. Props['size'] masih
// menyertakan undefined karena opsional, jadi Size dideklarasikan terpisah.
const SIZES: Record<Size, string> = {
  sm: 'max-w-sm',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-end justify-center p-4 sm:items-center">
      <!-- Backdrop -->
      <div class="absolute inset-0 bg-black/50" aria-hidden="true" @click="requestClose" />

      <div
        ref="panel"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        :aria-busy="busy || undefined"
        tabindex="-1"
        :class="['card relative z-10 w-full shadow-xl focus:outline-none', SIZES[size]]"
      >
        <!-- Header -->
        <div class="flex items-start justify-between gap-4 border-b border-default px-5 py-4">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink">{{ title }}</h2>
            <p v-if="description" class="mt-0.5 text-sm text-ink-muted">{{ description }}</p>
          </div>
          <button
            type="button"
            class="shrink-0 rounded-md p-1 text-ink-muted transition-colors hover:bg-default hover:text-ink disabled:opacity-50"
            :disabled="busy"
            aria-label="Tutup"
            @click="requestClose"
          >
            <X class="size-5" aria-hidden="true" />
          </button>
        </div>

        <!-- Body -->
        <div class="px-5 py-4">
          <slot />
        </div>

        <!-- Footer -->
        <div v-if="$slots.footer" class="flex justify-end gap-2 border-t border-default px-5 py-4">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Teleport>
</template>
