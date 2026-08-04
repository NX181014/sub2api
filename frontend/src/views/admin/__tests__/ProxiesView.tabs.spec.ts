import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../ProxiesView.vue'), 'utf8')

describe('ProxiesView tabs', () => {
  it('keeps the proxy list and Mihomo workbench in separate mounted panels', () => {
    expect(source).toContain("v-show=\"activeTab === 'proxies'\"")
    expect(source).toContain("v-show=\"activeTab === 'mihomo'\"")
    expect(source).toContain("const activeTab = ref<'proxies' | 'mihomo'>('proxies')")
    expect(source.indexOf('<MihomoPanel')).toBeGreaterThan(source.indexOf('</TablePageLayout>'))
  })

  it('executes primary administrator proxy actions without opening approval UI', () => {
    expect(source).toContain('if (isPrimaryAdmin.value)')
    expect(source).toContain('void submitApprovalRequest(true)')
    expect(source).toContain('if (result.approval_required)')
  })

  it('keeps ordinary proxy geography free of exit IP data', () => {
    const proxyList = source.slice(source.indexOf('<TablePageLayout'), source.indexOf('</TablePageLayout>'))
    const locationFormatter = source.slice(source.indexOf('const formatLocation'), source.indexOf('const flagUrl'))

    expect(proxyList).not.toContain('exit_ip')
    expect(proxyList).toContain('位置未检测')
    expect(locationFormatter).not.toContain('exit_ip')
    expect(locationFormatter).toContain('proxy.city || proxy.region')
  })

  it('runs batch checks only for an explicit selection', () => {
    const batchTest = source.slice(source.indexOf('const handleBatchTest'), source.indexOf('const handleBatchQualityCheck'))
    const batchQuality = source.slice(source.indexOf('const handleBatchQualityCheck'), source.indexOf('const approvalRequestTitle'))

    expect(batchTest).toContain('Array.from(selectedProxyIds.value)')
    expect(batchTest).not.toContain('fetchAllProxiesForBatch')
    expect(batchQuality).toContain('Array.from(selectedProxyIds.value)')
    expect(batchQuality).not.toContain('fetchAllProxiesForBatch')
    expect(source).toContain('v-if="selectedCount > 0"')
    expect(source).toContain('选择当前结果')
  })

  it('uses fixed responsive columns without changing the shared table', () => {
    expect(source).toContain("class: 'w-36 min-w-36 max-w-36'")
    expect(source).toContain('min-[1440px]:table-cell')
    expect(source).toContain('min-[1180px]:flex')
    expect(source).toContain('table-layout: fixed')
  })

  it('refreshes proxies after Mihomo changes and keeps removed routes operable', () => {
    const routesLoaded = source.slice(source.indexOf('const handleMihomoRoutesLoaded'), source.indexOf('const openManagedProxy'))
    const approvalApplied = source.slice(source.indexOf('const handleApprovalApplied'), source.indexOf('const handleProxyRevealed'))

    expect(routesLoaded).toContain('const shouldRefreshProxies = mihomoRoutesLoaded.value')
    expect(routesLoaded).toContain('if (shouldRefreshProxies) void loadProxies()')
    expect(source).toContain("proxy.managed_source?.startsWith('mihomo:route:')")
    expect(source).toContain('@applied="handleApprovalApplied"')
    expect(approvalApplied).toContain('const routesWereLoaded = mihomoRoutesLoaded.value')
    expect(approvalApplied).toContain('if (!routesWereLoaded) await loadProxies()')
    expect(source).toContain('线路已移除')
    expect(source).toContain('(!row.managed_source || isRemovedManagedRoute(row))')
    expect(source).toContain('@view-proxy-accounts="openManagedProxyAccounts"')
    expect(source).toContain("router.push({ path: '/admin/accounts', query: { tab: 'accounts' } })")
  })
})
