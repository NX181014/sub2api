import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get } }))

import { exportData, getSelectionSummary, listImportBatch, listRows } from '@/api/admin/accounts'

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

  it('requests logical rows with the active sort and filters', async () => {
    const page = { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
    get.mockResolvedValueOnce({ data: page })

    await expect(listRows(1, 20, { status: 'active', sort_by: 'created_at', sort_order: 'desc' })).resolves.toEqual(page)
    expect(get).toHaveBeenCalledWith('/admin/accounts/rows', {
      params: { page: 1, page_size: 20, status: 'active', sort_by: 'created_at', sort_order: 'desc' },
      signal: undefined
    })
  })

  it('passes the standalone and batched workbench scopes to server-side queries', async () => {
    const page = { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
    const summary = { total: 0, platforms: [], types: [] }
    get.mockResolvedValueOnce({ data: page }).mockResolvedValueOnce({ data: summary })

    await listRows(1, 20, { import_batch_scope: 'batched' })
    await getSelectionSummary({ import_batch_scope: 'standalone' })

    expect(get).toHaveBeenNthCalledWith(1, '/admin/accounts/rows', {
      params: { page: 1, page_size: 20, import_batch_scope: 'batched' },
      signal: undefined
    })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/accounts/selection-summary', {
      params: { import_batch_scope: 'standalone' }
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

  it('serializes workbench scope for export', async () => {
    get.mockResolvedValueOnce({ data: { accounts: [] } })

    await exportData({ filters: { import_batch_scope: 'standalone' } })

    expect(get).toHaveBeenCalledWith('/admin/accounts/data', {
      params: { import_batch_scope: 'standalone' }
    })
  })
})
