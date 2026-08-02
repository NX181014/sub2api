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
})
