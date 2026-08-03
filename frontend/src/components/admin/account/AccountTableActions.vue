<template>
  <div class="flex w-full min-w-0 flex-wrap items-center justify-end gap-2 sm:w-auto">
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
