<script setup lang="ts">
/**
 * Tombol dengan variant, loading state, dan ikon.
 *
 * `loading` bukan sekadar gaya: tombol yang sedang mengirim data harus
 * dinonaktifkan. Tanpa itu, user menekan berulang dan membuat tiga request
 * create untuk satu data yang sama.
 */
import { Loader2 } from 'lucide-vue-next'

type Variant = 'primary' | 'secondary' | 'danger' | 'ghost'
type Size = 'sm' | 'md'

interface Props {
  variant?: Variant
  size?: Size
  type?: 'button' | 'submit' | 'reset'
  loading?: boolean
  disabled?: boolean
  full?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
  loading: false,
  disabled: false,
  full: false,
})

const emit = defineEmits<{ click: [MouseEvent] }>()

// Satu sumber kebenaran untuk semua variant, bukan kelas utility yang
// disusun ulang di setiap pemakai. Konsistensi visual jadi otomatis.
const VARIANTS: Record<Variant, string> = {
  primary:
    'bg-brand-600 text-white shadow-sm hover:bg-brand-700 active:bg-brand-800 disabled:hover:bg-brand-600',
  secondary:
    'border border-default bg-raised text-ink shadow-sm hover:bg-default active:bg-default/80',
  danger:
    'bg-status-cancelled text-white shadow-sm hover:bg-status-cancelled/90 active:bg-status-cancelled/80',
  ghost: 'text-ink-muted hover:bg-default hover:text-ink active:bg-default/80',
}

const SIZES: Record<Size, string> = {
  sm: 'gap-1.5 px-2.5 py-1.5 text-xs',
  md: 'gap-2 px-3.5 py-2 text-sm',
}

function onClick(event: MouseEvent): void {
  // Loading disables the button anyway, tapi dicek juga di sini supaya
  // pemicu via keyboard atau programmatic tidak bisa menembusnya.
  if (props.loading || props.disabled) {
    event.preventDefault()
    return
  }
  emit('click', event)
}
</script>

<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :class="[
      'inline-flex items-center justify-center rounded-lg font-medium transition-[var(--transition-fast)]',
      'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500',
      'disabled:cursor-not-allowed disabled:opacity-60',
      VARIANTS[variant],
      SIZES[size],
      full ? 'w-full' : '',
    ]"
    :aria-busy="loading || undefined"
    @click="onClick"
  >
    <Loader2 v-if="loading" class="size-4 animate-spin" aria-hidden="true" />
    <slot name="icon" />
    <slot />
  </button>
</template>