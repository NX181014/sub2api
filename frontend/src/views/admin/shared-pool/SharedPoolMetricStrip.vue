<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

type MetricIcon = 'server' | 'users' | 'dollar' | 'chart' | 'exclamationTriangle' | 'checkCircle' | 'calculator' | 'link' | 'book' | 'bolt'

interface MetricItem {
  label: string
  value: string
  hint?: string
  tone?: 'default' | 'positive' | 'warning' | 'danger'
  icon?: MetricIcon
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

const iconClass: Record<NonNullable<MetricItem['tone']>, string> = {
  default: 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-300',
  positive: 'bg-green-50 text-green-600 dark:bg-green-900/20 dark:text-green-400',
  warning: 'bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400',
  danger: 'bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400'
}
</script>

<template>
  <dl class="grid min-w-0 grid-cols-2 gap-px overflow-hidden rounded-lg bg-gray-200 dark:bg-dark-700 sm:grid-cols-3 xl:grid-cols-5">
    <div v-for="item in items" :key="item.label" class="flex min-w-0 items-start gap-3 bg-gray-50 px-3 py-3 dark:bg-dark-900/50">
      <span v-if="item.icon" class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg" :class="iconClass[item.tone || 'default']">
        <Icon :name="item.icon" size="sm" />
      </span>
      <div class="min-w-0">
        <dt class="truncate text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</dt>
        <dd class="mt-1 truncate text-base font-semibold tabular-nums" :class="toneClass[item.tone || 'default']">{{ item.value }}</dd>
        <p v-if="item.hint" class="mt-0.5 line-clamp-2 text-xs text-gray-500 dark:text-gray-400" :title="item.hint">{{ item.hint }}</p>
      </div>
    </div>
  </dl>
</template>
