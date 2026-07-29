<template>
  <AppLayout>
    <CostLedgerPanel
      :initial-purchase-source-id="initialPurchaseSourceID"
      :initial-uploader-user-id="initialUploaderUserID"
      @open-account="openAccountRecord"
      @edit-entry="editLedgerEntry"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import type { SharedPoolLedgerEntry } from '@/api/admin/sharedPool'
import CostLedgerPanel from '@/views/admin/shared-pool/CostLedgerPanel.vue'

const route = useRoute()
const router = useRouter()
const queryID = (value: unknown) => {
  const id = Number(Array.isArray(value) ? value[0] : value)
  return Number.isSafeInteger(id) && id > 0 ? id : 0
}
const initialPurchaseSourceID = computed(() => queryID(route.query.purchase_source_id))
const initialUploaderUserID = computed(() => queryID(route.query.uploader_user_id))

const openAccountRecord = (accountID: number) => router.push({
  name: 'AdminSharedPool',
  query: { tab: 'accounts', account_id: String(accountID) }
})
const editLedgerEntry = (entry: SharedPoolLedgerEntry) => router.push({
  name: 'AdminSharedPool',
  query: { tab: 'accounts', account_id: String(entry.account_id), ledger_entry_id: String(entry.id) }
})
</script>
