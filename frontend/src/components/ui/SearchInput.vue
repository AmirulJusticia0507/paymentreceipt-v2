<script setup lang="ts">
/**
 * Input pencarian dengan debounce.
 *
 * Debounce itu wajib, bukan hiasan: tanpa itu setiap ketikan memicu satu
 * request. Mengetik "general affairs" dengan cepat = 15 request, dan sebagian
 * besar akan datang tidak berurutan sehingga daftar berkedip.
 */
import { onBeforeUnmount, ref, watch } from 'vue'
import { Search, X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    modelValue: string
    placeholder?: string
    debounce?: number
  }>(),
  { placeholder: 'Cari...', debounce: 300 },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const local = ref(props.modelValue)
let timer: ReturnType<typeof setTimeout> | undefined

// Nilai coming dari luar (mis. saat filter direset) langsung disalin ke input.
watch(
  () => props.modelValue,
  (v) => {
    local.value = v
  },
)

watch(local, (value) => {
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => emit('update:modelValue', value), props.debounce)
})

onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
})

function clear(): void {
  local.value = ''
}
</script>

<template>
  <div class="relative w-full sm:max-w-xs">
    <Search
      class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-ink-muted"
      aria-hidden="true"
    />
    <input
      :value="local"
      type="search"
      :placeholder="placeholder"
      aria-label="Cari data"
      class="block w-full rounded-lg border border-default bg-raised py-2 pr-9 pl-9 text-sm text-ink transition-[var(--transition-fast)] placeholder:text-ink-muted/60 focus:border-brand-500 focus:ring-2 focus:ring-brand-500/20 focus:outline-none"
      @input="local = ($event.target as HTMLInputElement).value"
    />
    <button
      v-if="local"
      type="button"
      class="absolute top-1/2 right-2.5 -translate-y-1/2 rounded p-0.5 text-ink-muted transition-colors hover:text-ink"
      aria-label="Bersihkan pencarian"
      @click="clear"
    >
      <X class="size-4" aria-hidden="true" />
    </button>
  </div>
</template>
