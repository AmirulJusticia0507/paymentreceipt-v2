<script setup lang="ts">
/**
 * Header halaman: judul, deskripsi singkat, dan breadcrumb opsional.
 *
 * Semua halaman pakai header yang sama supaya hierarki visual konsisten:
 * page title > section title > body > caption. Breadcrumb memandu user di
 * dalam sekelompok halaman yang terkait, mis. master data.
 */
import { RouterLink } from 'vue-router'

interface BreadcrumbItem {
  label: string
  to?: string
}

interface Props {
  title: string
  description?: string
  breadcrumb?: BreadcrumbItem[]
}

const props = withDefaults(defineProps<Props>(), {
  breadcrumb: () => [],
})
</script>

<template>
  <div class="flex flex-wrap items-baseline justify-between gap-3">
    <div>
      <h1 class="text-lg font-semibold tracking-tight text-ink">{{ title }}</h1>
      <p v-if="description" class="mt-0.5 text-sm text-ink-muted">{{ description }}</p>
    </div>

    <div v-if="breadcrumb.length" class="flex flex-wrap items-center gap-1 text-xs text-ink-muted">
      <span v-for="(item, index) in breadcrumb" :key="index" class="flex items-center gap-1">
        <RouterLink
          v-if="item.to && index < breadcrumb.length - 1"
          :to="item.to"
          class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
          >{{ item.label }}</RouterLink
        >
        <span v-else>{{ item.label }}</span>
        <svg
          v-if="index < breadcrumb.length - 1"
          class="size-3.5 text-ink-muted/50"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </span>
    </div>
  </div>
</template>
