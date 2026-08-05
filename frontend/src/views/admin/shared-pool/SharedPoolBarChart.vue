<script setup lang="ts">
import { computed } from 'vue'
import EmptyState from '@/components/common/EmptyState.vue'

interface BarItem {
  key: string | number
  label: string
  value: number
  display?: string
}

const props = withDefaults(defineProps<{
  title: string
  items: BarItem[]
  color?: string
  emptyTitle?: string
}>(), {
  color: '#14b8a6',
  emptyTitle: 'No data'
})

const maxValue = computed(() => Math.max(...props.items.map(item => Math.max(Number(item.value) || 0, 0)), 1))
</script>

<template>
  <section class="min-w-0 border-y border-gray-200 bg-white px-4 py-4 dark:border-dark-700 dark:bg-dark-800 sm:px-5">
    <div class="mb-4 flex min-w-0 items-center justify-between gap-3">
      <h2 class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ title }}</h2>
      <slot name="actions" />
    </div>
    <div v-if="items.length" class="space-y-3">
      <div v-for="item in items" :key="item.key" class="min-w-0">
        <div class="mb-1 flex min-w-0 items-center justify-between gap-3 text-xs">
          <span class="min-w-0 truncate text-gray-600 dark:text-gray-300" :title="item.label">{{ item.label }}</span>
          <span class="shrink-0 font-medium tabular-nums text-gray-900 dark:text-white">{{ item.display ?? item.value }}</span>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700" role="presentation">
          <div class="h-full rounded-full transition-[width] duration-200" :style="{ width: `${Math.min(Math.max((Number(item.value) || 0) / maxValue * 100, 0), 100)}%`, backgroundColor: color }" />
        </div>
      </div>
    </div>
    <EmptyState v-else :title="emptyTitle" />
  </section>
</template>
