<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { requestWithMeta } from '@/lib/http'

import BaseButton from '@/components/ui/BaseButton.vue'
import ToastNotification from '@/components/ui/ToastNotification.vue'
import { Printer } from 'lucide-vue-next'

const route = useRoute()
const receipt = ref<any>(null)
const toast = ref<{
  kind: 'success' | 'error' | 'info'
  title: string
  description?: string
} | null>(null)

async function load(): Promise<void> {
  const id = route.params.id
  const res = await requestWithMeta<any>(`/api/v1/payment-receipts/${id}`)
  receipt.value = res.data
}

async function cetak(): Promise<void> {
  const id = route.params.id
  try {
    // Buat blob dari PDF endpoint (iframe/print). Bila gagal, fallback print.
    const res = await fetch(`/api/v1/payment-receipts/${id}/pdf`, { credentials: 'include' })
    if (!res.ok) throw new Error('Gagal memuat PDF')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    window.open(url, '_blank')
    URL.revokeObjectURL(url)
  } catch {
    toast.value = {
      kind: 'info',
      title: 'Membuka print window',
      description: 'Menampilkan halaman untuk print (Ctrl/Cmd + P).',
    }
    window.print()
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold tracking-tight text-ink">Payment Receipt Detail</h1>
      <div class="flex gap-2">
        <BaseButton @click="cetak()">
          <template #icon><Printer class="size-4" aria-hidden="true" /></template>
          Cetak
        </BaseButton>
      </div>
    </div>
    <pre class="card p-4 text-xs overflow-auto">{{ receipt }}</pre>
  </div>
  <div class="fixed right-4 bottom-4 z-40 w-full max-w-sm">
    <Transition name="toast">
      <ToastNotification
        v-if="toast"
        :kind="toast.kind"
        :title="toast.title"
        :description="toast.description"
        @close="toast = null"
      />
    </Transition>
  </div>
</template>
