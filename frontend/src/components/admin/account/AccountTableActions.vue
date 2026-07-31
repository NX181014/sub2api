<template>
  <div class="flex w-full min-w-0 flex-wrap items-center justify-end gap-2 rounded-xl border border-gray-200 bg-white p-2 shadow-sm dark:border-dark-700 dark:bg-dark-800/70 sm:w-auto sm:gap-3 sm:border-0 sm:bg-transparent sm:p-0 sm:shadow-none">
    <slot name="before"></slot>
    <button
      type="button"
      class="btn btn-secondary min-h-11 min-w-11"
      :disabled="loading"
      :title="t('common.refresh')"
      :aria-label="t('common.refresh')"
      @click="$emit('refresh')"
    >
      <Icon name="refresh" size="md" :class="[loading ? 'animate-spin' : '']" />
    </button>
    <slot name="after"></slot>
    <slot name="beforeCreate"></slot>
    <button type="button" @click="$emit('create')" class="btn btn-primary min-h-11 px-3 sm:px-4">{{ t('admin.accounts.createAccount') }}</button>
    <slot name="afterCreate"></slot>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps(['loading'])
defineEmits(['refresh', 'create'])

const { t } = useI18n()
</script>
