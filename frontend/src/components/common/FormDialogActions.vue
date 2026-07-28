<template>
  <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:justify-end sm:gap-3">
    <button type="button" class="btn btn-secondary" :disabled="submitting" @click="emit('cancel')">
      {{ cancelText || t('common.cancel') }}
    </button>
    <button
      type="submit"
      :form="form"
      class="btn btn-primary"
      :disabled="submitting || disabled"
      :data-tour="submitDataTour"
    >
      <LoadingSpinner v-if="submitting" size="sm" />
      {{ submitText || t('common.save') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from './LoadingSpinner.vue'

defineProps<{
  form: string
  submitting?: boolean
  disabled?: boolean
  submitText?: string
  cancelText?: string
  submitDataTour?: string
}>()

const emit = defineEmits<{ cancel: [] }>()
const { t } = useI18n()
</script>
