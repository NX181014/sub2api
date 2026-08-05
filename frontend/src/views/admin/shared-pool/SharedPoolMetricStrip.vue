<script setup lang="ts">
interface MetricItem {
  label: string
  value: string
  hint?: string
  tone?: 'default' | 'positive' | 'warning' | 'danger'
}

withDefaults(defineProps<{ items: MetricItem[] }>(), {
  items: () => []
})

const toneClass: Record<NonNullable<MetricItem['tone']>, string> = {
  default: 'text-gray-900 dark:text-white',
  positive: 'text-green-600 dark:text-green-400',
  warning: 'text-amber-600 dark:text-amber-400',
  danger: 'text-red-600 dark:text-red-400'
}
</script>

<template>
  <dl class="grid min-w-0 grid-cols-2 gap-px border-y border-gray-200 bg-gray-100 dark:border-dark-700 dark:bg-dark-700 sm:grid-cols-3 lg:grid-cols-5">
    <div v-for="item in items" :key="item.label" class="min-w-0 bg-white px-4 py-3 dark:bg-dark-800 sm:px-5">
      <dt class="truncate text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</dt>
      <dd class="mt-1 truncate text-lg font-semibold tabular-nums" :class="toneClass[item.tone || 'default']">{{ item.value }}</dd>
      <p v-if="item.hint" class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{{ item.hint }}</p>
    </div>
  </dl>
</template>
