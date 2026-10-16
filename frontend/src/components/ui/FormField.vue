<script setup lang="ts">
/**
 * Input form dengan label, pesan error, dan helper text.
 *
 * Ketiganya digabung dalam satu komponen karena selalu muncul bersama dan
 * harus saling terhubung lewat atribut aria. Kalau dipisah, pasti ada bagian
 * yang lupa memasang aria-describedby sehingga pembacaan layar tidak tahu
 * ada pesan error di bawah input.
 *
 * `modelValue` dipakai lewat v-model supaya form bisa mengikat nilai dari
 * vee-validate tanpa perlu menembus DOM.
 */
import { useId } from 'vue'
import { AlertCircle } from 'lucide-vue-next'

interface Props {
  label: string
  modelValue?: string
  error?: string
  hint?: string
  required?: boolean
  disabled?: boolean
  placeholder?: string
  maxlength?: number | string
  type?: string
  autocomplete?: string
  inputmode?: 'text' | 'numeric' | 'decimal' | 'tel' | 'email' | 'search'
  /** Teks tombol kecil di kanan input, mis. tombol lihat password. */
  suffix?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  type: 'text',
  suffix: false,
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

// useId menghasilkan id unik dan stabil antar render. Id manual berisiko
// bentrok kalau halaman punya dua form dengan label yang sama - dan duplikat
// id merusak hubungan label dengan input untuk pengguna keyboard.
const id = useId()
const errorId = `${id}-error`
const hintId = `${id}-hint`

function onInput(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="space-y-1.5">
    <label :for="id" class="block text-sm font-medium text-ink">
      {{ label }}
      <span v-if="required" class="text-status-cancelled" aria-hidden="true">*</span>
    </label>

    <div class="relative">
      <input
        :id="id"
        :type="type"
        :value="props.modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :required="required"
        :maxlength="maxlength"
        :autocomplete="autocomplete"
        :inputmode="inputmode"
        :aria-invalid="error ? true : undefined"
        :aria-describedby="error ? errorId : hint ? hintId : undefined"
        class="block w-full rounded-lg border bg-raised px-3 py-2 text-sm text-ink transition-[var(--transition-fast)] placeholder:text-ink-muted/60 focus:ring-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
        :class="[
          suffix ? 'pr-10' : '',
          error
            ? 'border-status-cancelled focus:border-status-cancelled focus:ring-status-cancelled/20'
            : 'border-default focus:border-brand-500 focus:ring-brand-500/20',
        ]"
        @input="onInput"
      />

      <span class="absolute top-1/2 right-3 -translate-y-1/2">
        <AlertCircle
          v-if="error"
          class="pointer-events-none size-4 text-status-cancelled"
          aria-hidden="true"
        />
        <slot v-else name="suffix" />
      </span>
    </div>

    <!-- Error diprioritaskan di atas helper text: kalau ada yang salah, itu
         yang perlu dibaca lebih dulu. -->
    <p v-if="error" :id="errorId" class="text-xs text-status-cancelled">{{ error }}</p>
    <p v-else-if="hint" :id="hintId" class="text-xs text-ink-muted">{{ hint }}</p>
  </div>
</template>