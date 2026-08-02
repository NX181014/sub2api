<template>
  <section v-if="status?.enabled" class="min-w-0 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="font-semibold text-gray-900 dark:text-white">Mihomo 节点池</h2>
          <span :class="['badge', status.configured ? 'badge-success' : 'badge-danger']">
            {{ status.configured ? '订阅已配置' : '订阅未配置' }}
          </span>
          <span class="badge badge-gray">{{ status.alive_count }}/{{ status.node_count }} 可用</span>
        </div>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          自动分流、定向节点和动态轮换共用同一订阅；修改需另一位管理员审核。
          <span v-if="status.updated_at">最近刷新 {{ formatTime(status.updated_at) }}</span>
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary min-h-11" @click="openAction('refresh')">
          <Icon name="refresh" size="sm" class="mr-1.5" />刷新订阅
        </button>
        <button type="button" class="btn btn-primary min-h-11" @click="openAction('subscription')">更新订阅</button>
      </div>
    </div>

    <div class="mt-4 grid min-w-0 gap-3 xl:grid-cols-3">
      <article class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">自动</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">容灾优先会先保证可用；最低延迟会选择当前响应最快的节点。</p>
        <p class="mt-3 text-xs text-gray-500">当前：{{ automaticLabel }}</p>
        <div class="mt-2 grid grid-cols-2 gap-2">
          <button type="button" class="btn min-h-11 px-2" :class="modeClass('automatic', 'FALLBACK')" @click="openMode('automatic', 'FALLBACK')">容灾优先</button>
          <button type="button" class="btn min-h-11 px-2" :class="modeClass('automatic', 'AUTO')" @click="openMode('automatic', 'AUTO')">最低延迟</button>
        </div>
      </article>

      <article class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">定向</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">将此出口固定到指定订阅节点；未选择时拒绝流量，避免误走其他线路。</p>
        <p class="mt-3 truncate text-xs text-gray-500" :title="directionalLabel">当前：{{ directionalLabel }}</p>
        <div class="mt-2 flex min-w-0 gap-2">
          <input v-model.trim="directionalSelection" list="mihomo-directional-nodes" class="input min-w-0 flex-1" placeholder="搜索或选择节点" />
          <datalist id="mihomo-directional-nodes">
            <option value="REJECT">未选择</option>
            <option v-for="node in status.nodes" :key="node.name" :value="node.name">{{ node.alive ? '可用' : '异常' }} · {{ node.delay || '-' }}ms</option>
          </datalist>
          <button type="button" class="btn btn-secondary min-h-11 shrink-0 px-3" :disabled="!directionalSelection" @click="openMode('directional', directionalSelection)">提交</button>
        </div>
      </article>

      <article class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">动态</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">动态轮换按 Mihomo 策略切换节点；关闭后此出口直接拒绝流量。</p>
        <p class="mt-3 text-xs text-gray-500">当前：{{ dynamicLabel }}</p>
        <div class="mt-2 grid grid-cols-2 gap-2">
          <button type="button" class="btn min-h-11 px-2" :class="modeClass('dynamic', 'DYNAMIC')" @click="openMode('dynamic', 'DYNAMIC')">动态轮换</button>
          <button type="button" class="btn min-h-11 px-2" :class="modeClass('dynamic', 'REJECT')" @click="openMode('dynamic', 'REJECT')">关闭</button>
        </div>
      </article>
    </div>
  </section>

  <BaseDialog :show="dialogOpen" :title="dialogTitle" width="normal" @close="closeDialog">
    <form id="mihomo-approval-form" class="space-y-4" @submit.prevent="submit">
      <div v-if="action === 'subscription'">
        <label class="input-label" for="mihomo-subscription-url">订阅地址</label>
        <input id="mihomo-subscription-url" v-model.trim="subscriptionUrl" type="url" required autocomplete="off" class="input" placeholder="https://…" />
        <p class="input-hint mt-1">地址只会加密进入审批，不在页面、审批详情或审计日志中显示。</p>
      </div>
      <div v-else-if="action === 'mode'" class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-900/40">
        {{ modeName }}：{{ selectionLabel }}
      </div>
      <div>
        <label class="input-label" for="mihomo-approval-reason">变更原因</label>
        <textarea id="mihomo-approval-reason" v-model.trim="reason" required rows="3" class="input" placeholder="说明用途和影响范围，供另一位管理员审核"></textarea>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary min-h-11" @click="closeDialog">取消</button>
        <button type="submit" form="mihomo-approval-form" class="btn btn-primary min-h-11" :disabled="submitting || !reason">{{ submitting ? '提交中…' : '提交审核' }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { MihomoStatus } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

const emit = defineEmits<{ 'approval-submitted': [] }>()
const appStore = useAppStore()
const status = ref<MihomoStatus | null>(null)
const action = ref<'subscription' | 'refresh' | 'mode' | null>(null)
const mode = ref('')
const selection = ref('')
const directionalSelection = ref('')
const subscriptionUrl = ref('')
const reason = ref('')
const submitting = ref(false)
const dialogOpen = computed(() => action.value !== null)
const currentSelection = (key: 'automatic' | 'directional' | 'dynamic') => status.value?.modes?.[key]?.selection || 'REJECT'
const automaticLabel = computed(() => currentSelection('automatic') === 'AUTO' ? '最低延迟' : '容灾优先')
const directionalLabel = computed(() => currentSelection('directional') === 'REJECT' ? '未选择' : currentSelection('directional'))
const dynamicLabel = computed(() => currentSelection('dynamic') === 'DYNAMIC' ? '动态轮换' : '关闭')
const dialogTitle = computed(() => action.value === 'subscription' ? '更新 Mihomo 订阅' : action.value === 'refresh' ? '刷新 Mihomo 订阅' : '修改 Mihomo 模式')
const modeName = computed(() => ({ automatic: '自动', directional: '定向', dynamic: '动态' })[mode.value] || mode.value)
const selectionLabel = computed(() => {
  if (selection.value === 'FALLBACK') return '容灾优先'
  if (selection.value === 'AUTO') return '最低延迟'
  if (selection.value === 'DYNAMIC') return '动态轮换'
  if (selection.value === 'REJECT') return '关闭 / 未选择'
  return selection.value
})

const load = async () => {
  try { status.value = await adminAPI.mihomo.getStatus() } catch { status.value = null }
}
const formatTime = (value: string) => new Date(value).toLocaleString()
const modeClass = (key: 'automatic' | 'dynamic', value: string) => currentSelection(key) === value ? 'btn-primary' : 'btn-secondary'
const openAction = (next: 'subscription' | 'refresh') => { action.value = next }
const openMode = (nextMode: string, nextSelection: string) => {
  mode.value = nextMode
  selection.value = nextSelection
  action.value = 'mode'
}
const closeDialog = () => {
  action.value = null
  reason.value = ''
  subscriptionUrl.value = ''
}
const submit = async () => {
  if (!action.value || !reason.value) return
  submitting.value = true
  try {
    if (action.value === 'subscription') await adminAPI.mihomo.updateSubscription(subscriptionUrl.value, reason.value)
    else if (action.value === 'refresh') await adminAPI.mihomo.refresh(reason.value)
    else await adminAPI.mihomo.updateMode(mode.value, selection.value, reason.value)
    appStore.showSuccess('已提交给其他管理员审核')
    closeDialog()
    emit('approval-submitted')
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || '提交审核失败')
  } finally { submitting.value = false }
}

onMounted(load)
</script>
