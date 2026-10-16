<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { Compass } from 'lucide-vue-next'
</script>

<template>
  <main class="flex min-h-dvh items-center justify-center bg-surface px-4">
    <div class="max-w-sm text-center">
      <div
        class="mx-auto mb-4 flex size-12 items-center justify-center rounded-xl bg-default text-ink-muted"
      >
        <Compass class="size-6" aria-hidden="true" />
      </div>
      <h1 class="text-xl font-semibold tracking-tight text-ink">Halaman tidak ditemukan</h1>
      <p class="mt-2 text-sm text-ink-muted">
        Alamat yang Anda buka tidak tersedia, atau sudah dipindahkan.
      </p>
      <RouterLink
        to="/"
        class="mt-6 inline-flex items-center rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-[var(--transition-fast)] hover:bg-brand-700"
      >
        Kembali ke dashboard
      </RouterLink>
    </div>
  </main>
</template>