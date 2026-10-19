<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { requestWithMeta } from '@/lib/http'

const route = useRoute()
const archive = ref<any>(null)

async function load(): Promise<void> {
  const id = route.params.id
  const res = await requestWithMeta<any>(`/api/v1/pr-archives/${id}`)
  archive.value = res.data
}

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold tracking-tight text-ink">Detail Arsip PR</h1>
    </div>

    <div v-if="archive" class="card p-4 space-y-3">
      <div class="grid grid-cols-2 gap-4 text-sm">
        <div>
          <p class="text-ink-muted">Kode</p>
          <p class="font-mono text-ink font-medium">{{ archive.kode }}</p>
        </div>
        <div>
          <p class="text-ink-muted">Receipt Asal</p>
          <p class="font-mono text-ink font-medium">{{ archive.receipt_kode }}</p>
        </div>
        <div>
          <p class="text-ink-muted">Nama Request</p>
          <p class="text-ink">{{ archive.nama_request }}</p>
        </div>
        <div>
          <p class="text-ink-muted">Manager Approved</p>
          <p class="text-ink">{{ archive.manager_approved || '-' }}</p>
        </div>
        <div>
          <p class="text-ink-muted">Approver</p>
          <p class="text-ink">{{ archive.approved_by_nama || '-' }}</p>
        </div>
        <div>
          <p class="text-ink-muted">Quantity</p>
          <p class="text-ink">{{ archive.quantity }}</p>
        </div>
        <div>
          <p class="text-ink-muted">Harga Rencana</p>
          <p class="text-ink tabular-nums">{{ archive.harga_rencana }}</p>
        </div>
        <div>
          <p class="text-ink-muted">Harga Aktual</p>
          <p class="text-ink font-bold tabular-nums">{{ archive.harga_aktual }}</p>
        </div>
        <div>
          <p class="text-ink-muted">Tanggal Selesai</p>
          <p class="text-ink">
            {{ archive.tanggal_selesai ? new Date(archive.tanggal_selesai).toLocaleString('id-ID') : '-' }}
          </p>
        </div>
        <div>
          <p class="text-ink-muted">Keterangan</p>
          <p class="text-ink">{{ archive.description || '-' }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
