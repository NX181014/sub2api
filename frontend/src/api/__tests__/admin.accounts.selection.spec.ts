import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get } }))

import { exportData, getSelectionSummary, listImportBatch } from '@/api/admin/accounts'

describe('admin account selection API', () => {
  beforeEach(() => get.mockReset())

  it('requests an exact summary with the active filters', async () => {
    const summary = { total: 2, platforms: ['openai'], types: ['oauth'] }
    get.mockResolvedValueOnce({ data: summary })

    await expect(getSelectionSummary({ uploader_user_id: 77, status: 'active' })).resolves.toEqual(summary)
    expect(get).toHaveBeenCalledWith('/admin/accounts/selection-summary', {
      params: { uploader_user_id: 77, status: 'active' }
    })
  })

  it('paginates a complete import batch through the normal account list', async () => {
    const page = { items: [], total: 101, page: 2, page_size: 100, total_pages: 2 }
    get.mockResolvedValueOnce({ data: page })

    await expect(listImportBatch('668f52b3-14af-4a5a-bde0-e923ed69299a', 2, 100)).resolves.toEqual(page)
    expect(get).toHaveBeenCalledWith('/admin/accounts', {
      params: { page: 2, page_size: 100, import_batch_id: '668f52b3-14af-4a5a-bde0-e923ed69299a' },
      signal: undefined
    })
  })

  it('serializes uploader and import batch filters for export', async () => {
    get.mockResolvedValueOnce({ data: { accounts: [] } })

    await exportData({
      filters: { uploader_user_id: 77, import_batch_id: '668f52b3-14af-4a5a-bde0-e923ed69299a' }
    })
    expect(get).toHaveBeenCalledWith('/admin/accounts/data', {
      params: { uploader_user_id: '77', import_batch_id: '668f52b3-14af-4a5a-bde0-e923ed69299a' }
    })
  })
})
