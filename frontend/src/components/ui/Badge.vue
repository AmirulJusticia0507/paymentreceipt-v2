<script setup lang="ts">
/**
 * Badge status dengan warna semantik seragam di seluruh aplikasi.
 *
 * Varian status (pending, checked, processed, finished, cancelled) pakai
 * palet status dari @theme di styles/main.css, bukan warna hex sembarang.
 * Varian `neutral` untuk label biasa yang bukan status transaksi.
 */
import { computed } from 'vue'

interface Props {
  variant?: 'pending' | 'checked' | 'processed' | 'finished' | 'cancelled' | 'neutral'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'neutral',
})

const VARIANTS: Record<string, string> = {
  pending:   'bg-status-pending/10 text-status-pending border border-status-pending/20',
  checked:   'bg-status-checked/10 text-status-checked border border-status-checked/20',
  processed: 'bg-status-processed/10 text-status-processed border border-status-processed/20',
  finished:  'bg-status-finished/10 text-status-finished border border-status-finished/20',
  cancelled: 'bg-status-cancelled/10 text-status-cancelled border border-status-cancelled/20',
  neutral:   'bg-default text-ink-muted border border-default/60',
}

const cls = computed(() => VARIANTS[props.variant ?? 'neutral'])
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium transition-colors"
    :class="cls"
    aria-hidden="true"
  >
    <slot />
  </span>
</template>
