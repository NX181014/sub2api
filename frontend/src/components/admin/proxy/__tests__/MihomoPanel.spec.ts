import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import MihomoPanel from '../MihomoPanel.vue'

const mocks = vi.hoisted(() => ({
  getWorkbench: vi.fn(),
  getImportPreview: vi.fn(),
  importLegacy: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    mihomo: {
      getWorkbench: mocks.getWorkbench,
      getImportPreview: mocks.getImportPreview,
      importLegacy: mocks.importLegacy,
    },
  },
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { is_primary_admin: true } }) }))

const workbench = {
  status: {
    enabled: true,
    configured: true,
    controller_connected: true,
    config_valid: true,
    version: '1.19.0',
  },
  subscriptions: [{
    id: 7,
    name: '香港主订阅',
    enabled: true,
    status: 'healthy',
    refresh_interval_minutes: 60,
    node_count: 2,
    alive_count: 2,
  }],
  nodes: [{ id: 11, name: 'HK-01', subscription_id: 7, subscription_name: '香港主订阅', alive: true, delay: 38 }],
  routes: [{
    id: 3,
    name: '香港定向 01',
    kind: 'directional',
    subscription_ids: [7],
    subscription_names: ['香港主订阅'],
    node_ids: [11],
    listener_port: 26784,
    proxy_id: 92,
    enabled: true,
    current_node: 'HK-01',
    exit_ip: '198.51.100.8',
    health: 'healthy',
    latency_ms: 38,
    account_count: 4,
  }],
}

describe('MihomoPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getWorkbench.mockResolvedValue(workbench)
    mocks.getImportPreview.mockResolvedValue({ available: false, routes: [] })
  })

  it('shows the managed route business details and exposes them to the proxy list', async () => {
    const wrapper = mount(MihomoPanel)
    await flushPromises()

    expect(wrapper.text()).toContain('香港定向 01')
    expect(wrapper.text()).toContain('HK-01')
    expect(wrapper.text()).toContain('198.51.100.8')
    expect(wrapper.text()).toContain('绑定账号')
    expect(wrapper.emitted('routes-loaded')?.[0]).toEqual([workbench.routes])
  })

  it('filters routes without removing the workbench controls', async () => {
    const wrapper = mount(MihomoPanel)
    await flushPromises()

    await wrapper.get('input[aria-label="搜索 Mihomo 线路"]').setValue('不存在')

    expect(wrapper.text()).toContain('没有符合条件的线路')
    expect(wrapper.text()).toContain('新建线路')
  })

  it('shows the exact legacy resources before import', async () => {
    mocks.getWorkbench.mockResolvedValue({ ...workbench, subscriptions: [], nodes: [], routes: [] })
    mocks.getImportPreview.mockResolvedValue({
      available: true,
      subscription_host: 'provider.example',
      node_count: 8,
      route_count: 3,
      affected_account_count: 5,
      routes: [{ name: '自动线路', listener_port: 26781, proxy_id: 91, account_count: 5 }],
    })

    const wrapper = mount(MihomoPanel)
    await flushPromises()

    expect(wrapper.text()).toContain('provider.example')
    expect(wrapper.text()).toContain('端口 26781')
    expect(wrapper.text()).toContain('代理 #91')
    expect(wrapper.text()).toContain('5 个账号')
  })

  it('shows the service error when importing the legacy configuration', async () => {
    mocks.getWorkbench.mockResolvedValue({ ...workbench, subscriptions: [], nodes: [], routes: [] })
    mocks.getImportPreview.mockResolvedValue({ available: true, node_count: 8, route_count: 3, affected_account_count: 5, routes: [] })
    mocks.importLegacy.mockRejectedValue({ message: 'Mihomo 配置校验失败：监听端口已占用' })

    const wrapper = mount(MihomoPanel, { global: { stubs: { teleport: true } } })
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('确认导入'))!.trigger('click')
    await wrapper.get('#mihomo-confirm-form').trigger('submit')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('Mihomo 配置校验失败：监听端口已占用')
  })
})
