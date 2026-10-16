<script setup lang="ts">
/**
 * Placeholder tabel saat data sedang dimuat.
 *
 * Menggantikan spinner karena spinner membuat layout melompat: tabel muncul
 * dengan tinggi berbeda begitu data datang, dan isi halaman bergeser di bawah
 * mata. Kerangka dengan jumlah baris dan kolom yang sama menjaga tinggi tetap.
 *
 * Animasi shimmer dihormati prefers-reduced-motion lewat CSS global.
 */
withDefaults(
  defineProps<{
    rows?: number
    columns?: number
  }>(),
  { rows: 5, columns: 4 },
)
</script>

<template>
  <div class="p-4" aria-hidden="true">
    <div class="space-y-3">
      <div
        v-for="row in rows"
        :key="row"
        class="flex items-center gap-4"
      >
        <div
          v-for="col in columns"
          :key="col"
          class="h-4 flex-1 animate-pulse rounded bg-default"
          :style="{ opacity: 1 - row * 0.08 }"
        />
      </div>
    </div>
  </div>
</template>