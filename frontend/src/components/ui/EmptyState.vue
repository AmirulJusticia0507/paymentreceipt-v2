<script setup lang="ts">
/**
 * Placeholder untuk kondisi tanpa data.
 *
 * "Kosong" tanpa penjelasan adalah tanda buruk: user tidak tahu apakah
 * halaman gagal memuat, atau memang belum ada data, atau filter terlalu
 * ketat. Karena itu setiap kondisi kosong menjelaskan apa yang terjadi dan
 * offer jalan keluar lewat slot aksi.
 */
</script>

<template>
  <div class="flex flex-col items-center gap-3 text-center">
    <div class="rounded-xl bg-default p-3 text-ink-muted">
      <slot name="icon" />
    </div>
    <slot />
    <slot name="action" />
  </div>
</template>