<script setup lang="ts">
/**
 * Dialog konfirmasi untuk aksi yang merusak (hapus, nonaktifkan).
 *
 * Sengaja memakai dialog, bukan confirm() bawaan browser: confirm() tidak bisa
 * diberi konteks, tidak bisa doesnt style, dan pesan yang bisa ditulis tidak
 * bisa menjelaskan akibatnya.
 */
import { AlertTriangle } from 'lucide-vue-next'

import BaseButton from './BaseButton.vue'
import BaseModal from './BaseModal.vue'

interface Props {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  busy?: boolean
  /** Nilai yang sedang dihapus, ditampilkan supaya user tidak salah klik. */
  itemName?: string
}

withDefaults(defineProps<Props>(), {
  confirmLabel: 'Hapus',
  busy: false,
})

const emit = defineEmits<{ confirm: []; cancel: [] }>()
</script>

<template>
  <BaseModal
    :open="open"
    :title="title"
    size="sm"
    :busy="busy"
    @close="emit('cancel')"
  >
    <div class="flex gap-3">
      <div class="shrink-0 rounded-lg bg-status-cancelled/10 p-2 text-status-cancelled">
        <AlertTriangle class="size-5" aria-hidden="true" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="text-sm text-ink">{{ message }}</p>
        <p v-if="itemName" class="mt-2 text-sm font-medium text-ink">
          <span class="font-mono">{{ itemName }}</span>
        </p>
        <slot name="detail" />
      </div>
    </div>

    <template #footer>
      <BaseButton variant="secondary" :disabled="busy" @click="emit('cancel')">
        Batal
      </BaseButton>
      <BaseButton variant="danger" :loading="busy" @click="emit('confirm')">
        {{ confirmLabel }}
      </BaseButton>
    </template>
  </BaseModal>
</template>