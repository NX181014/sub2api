import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ProxyApprovalCenter from '../ProxyApprovalCenter.vue'

const mocks = vi.hoisted(() => ({
  listApprovals: vi.fn(),
  approveApproval: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    sharedPool: {
      listApprovals: mocks.listApprovals,
      approveApproval: mocks.approveApproval,
    },
  },
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { is_primary_admin: true } }) }))

const approval = {
  id: 9,
  action_type: 'UPDATE_MIHOMO' as const,
  object_type: 'mihomo' as const,
  status: 'pending' as const,
  reason: 'remove route',
  resource_key: 'route:3',
  requested_by_user_id: 2,
  requested_by_email: 'reviewer@example.test',
  requested_at: '2026-08-04T00:00:00Z',
}

describe('ProxyApprovalCenter', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.listApprovals.mockImplementation(({ object_type }: { object_type: string }) => Promise.resolve({
      items: object_type === 'mihomo' ? [approval] : [],
      total: object_type === 'mihomo' ? 1 : 0,
      page: 1,
      page_size: 50,
      pages: 1,
    }))
    mocks.approveApproval.mockResolvedValue({ ...approval, status: 'approved' })
  })

  it('notifies the proxy page only after an approval is applied', async () => {
    const wrapper = mount(ProxyApprovalCenter, { props: { show: true }, global: { stubs: { teleport: true } } })
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === '通过')!.trigger('click')
    await flushPromises()

    expect(mocks.approveApproval).toHaveBeenCalledWith(approval.id)
    expect(wrapper.emitted('applied')?.[0]).toEqual([{ ...approval, status: 'approved' }])
  })
})
