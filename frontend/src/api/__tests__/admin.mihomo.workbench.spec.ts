import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { get, post } }))

import { getImportPreview, getWorkbench, refreshSubscription } from '@/api/admin/mihomo'

describe('admin Mihomo workbench API', () => {
  beforeEach(() => { get.mockReset(); post.mockReset() })

  it('normalizes persisted resource fields for the workbench', async () => {
    get.mockResolvedValue({
      data: {
        status: { enabled: true, configured: true },
        subscriptions: [{
          id: 7,
          name: '主订阅',
          status: 'active',
          masked_host: 'provider.example',
          refresh_interval_seconds: 1800,
          quota_used_bytes: 1024,
          quota_total_bytes: 4096,
        }],
        nodes: [{ id: 11, subscription_id: 7, node_key: 'node-11', original_name: 'HK-01', display_name: '香港 01', alive: true, delay_ms: 42 }],
        routes: [{ id: 3, name: '香港自动', kind: 'automatic', listener_port: 26784, proxy_id: 92, status: 'active', subscription_ids: [7], node_ids: [11], current_node_id: 11, exit_healthy: true, exit_delay_ms: 42 }],
      },
    })

    const result = await getWorkbench()

    expect(result.subscriptions[0]).toMatchObject({ enabled: true, source_host: 'provider.example', refresh_interval_minutes: 30, used_bytes: 1024, total_bytes: 4096 })
    expect(result.nodes[0]).toMatchObject({ key: 'node-11', name: '香港 01', delay: 42, subscription_name: '主订阅' })
    expect(result.routes[0]).toMatchObject({ enabled: true, current_node: '香港 01', health: 'healthy', latency_ms: 42, account_count: 0 })
    expect(get).toHaveBeenCalledWith('/admin/mihomo/workbench')
  })

  it('loads a read-only legacy import preview', async () => {
    get.mockResolvedValue({ data: { available: true, node_count: 4, route_count: 3, affected_account_count: 2, routes: [] } })

    await expect(getImportPreview()).resolves.toMatchObject({ available: true, node_count: 4 })
    expect(get).toHaveBeenCalledWith('/admin/mihomo/import-preview')
  })

  it('submits managed subscription refresh through the approval endpoint', async () => {
    post.mockResolvedValue({ data: { approval_required: true } })

    await expect(refreshSubscription(7, '同步节点库存')).resolves.toEqual({ approval_required: true })
    expect(post).toHaveBeenCalledWith('/admin/mihomo/subscriptions/7/refresh', { reason: '同步节点库存' })
  })
})
