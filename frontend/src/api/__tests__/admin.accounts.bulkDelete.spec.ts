import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { post } }))

import { bulkDelete } from '@/api/admin/accounts'

describe('admin account bulk deletion API', () => {
  beforeEach(() => post.mockReset())

  it('sends account IDs and preserves deleted, approval, and failed results', async () => {
    const result = {
      deleted: 1,
      approval_required: 1,
      failed: 1,
      results: [
        { account_id: 1, status: 'deleted' as const },
        { account_id: 2, status: 'approval_required' as const, approval: { id: 9 } },
        { account_id: 3, status: 'failed' as const, error: 'not found' }
      ]
    }
    post.mockResolvedValueOnce({ data: result })

    await expect(bulkDelete([1, 2, 3])).resolves.toEqual(result)
    expect(post).toHaveBeenCalledWith('/admin/accounts/bulk-delete', { account_ids: [1, 2, 3] })
  })
})
