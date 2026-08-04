import { defineComponent } from 'vue'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'

const showError = vi.fn()
const showSuccess = vi.fn()
const showWarning = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      importData: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      key === 'admin.accounts.dataImportNoProxyConfirmMessage'
        ? `${key}:${params?.count}`
        : key
  })
}))

const ConfirmDialogStub = defineComponent({
  name: 'ConfirmDialog',
  props: { show: Boolean, message: String },
  emits: ['confirm', 'cancel'],
  template: `
    <div v-if="show" data-test="proxyless-confirm" :data-message="message">
      <button type="button" data-test="proxyless-cancel" @click="$emit('cancel')">cancel</button>
      <button type="button" data-test="proxyless-continue" @click="$emit('confirm')">continue</button>
    </div>
  `
})

const mountModal = (props: Record<string, unknown> = {}) =>
  mount(ImportDataModal, {
    props: { show: true, ...props },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        ConfirmDialog: ConfirmDialogStub,
        ProxySelector: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<select data-test="import-default-proxy" :value="modelValue ?? \'\'" @change="$emit(\'update:modelValue\', $event.target.value ? Number($event.target.value) : null)"><option value="">none</option><option value="7">Proxy 7</option></select>'
        },
        GroupSelector: {
          props: ['modelValue', 'groups'],
          emits: ['update:modelValue'],
          template: '<button type="button" data-test="import-groups" :data-group-ids="groups.map(group => group.id).join(\',\')" @click="$emit(\'update:modelValue\', [3, 4])">groups</button>'
        }
      }
    }
  })

const continueWithoutProxy = async (wrapper: ReturnType<typeof mountModal>) => {
  await wrapper.get('[data-test="proxyless-continue"]').trigger('click')
  await flushPromises()
}

const makeJsonFile = (name: string, content: string, type = 'application/json') => {
  const file = new File([content], name, { type })
  Object.defineProperty(file, 'text', {
    value: () => Promise.resolve(content)
  })
  return file
}

const setInputFiles = (element: Element, files: File[]) => {
  Object.defineProperty(element, 'files', {
    value: files,
    configurable: true
  })
}

describe('ImportDataModal', () => {
  beforeEach(async () => {
    showError.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockReset()
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mountModal()

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时按文件名提示解析失败', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('data.json', 'invalid json')])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailedFile')
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })

  it('不是导出数据的 JSON 按文件名拒绝', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('random.json', JSON.stringify({ name: 'test' }))])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportInvalidFile')
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })

  it('rejects an unsupported account platform before importing', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile(
      'unknown-platform.json',
      JSON.stringify({ proxies: [], accounts: [{ platform: 'unknown' }] })
    )])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportInvalidFile')
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })

  it('无有效 JSON 的选择不清空已有选择', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0
    })

    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')

    const valid = makeJsonFile(
      'valid.json',
      JSON.stringify({ exported_at: '2026-07-05T00:00:00Z', proxies: [], accounts: [{ name: 'a' }] })
    )
    setInputFiles(input.element, [valid])
    await input.trigger('change')

    setInputFiles(input.element, [new File(['hello'], 'notes.txt', { type: 'text/plain' })])
    await input.trigger('change')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')

    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await continueWithoutProxy(wrapper)

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.objectContaining({
        accounts: [{ name: 'a' }]
      }),
      skip_default_group_bind: true
    })
  })

  it('merges multiple selected JSON files before importing', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 2,
      account_failed: 0
    })

    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    const first = makeJsonFile(
      'first.json',
      JSON.stringify({ exported_at: '2026-07-05T00:00:00Z', proxies: [], accounts: [{ name: 'a' }] })
    )
    const second = makeJsonFile(
      'second.json',
      JSON.stringify({
        exported_at: '2026-07-05T00:00:01Z',
        proxies: [{ proxy_key: 'p' }],
        accounts: [{ name: 'b' }]
      })
    )
    setInputFiles(input.element, [first, second])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await continueWithoutProxy(wrapper)

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.objectContaining({
        proxies: [{ proxy_key: 'p' }],
        accounts: [{ name: 'a' }, { name: 'b' }]
      }),
      skip_default_group_bind: true
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.dataImportSuccess')
  })

  it('sends the selected default proxy and groups with the import request', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0
    })

    const wrapper = mountModal({ proxies: [{ id: 7, name: 'Proxy 7' }], groups: [{ id: 3 }, { id: 4 }] })
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile(
      'bindings.json',
      JSON.stringify({ exported_at: '2026-07-05T00:00:00Z', proxies: [], accounts: [{ name: 'a' }] })
    )])

    await input.trigger('change')
    await wrapper.get('[data-test="import-default-proxy"]').setValue('7')
    await wrapper.get('[data-test="import-groups"]').trigger('click')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith(expect.objectContaining({
      default_proxy_id: 7,
      group_ids: [3, 4]
    }))
  })

  it('shows only groups compatible with every imported account platform', async () => {
    const wrapper = mountModal({
      groups: [
        { id: 3, platform: 'openai' },
        { id: 4, platform: 'gemini' },
        { id: 5, platform: 'composite' }
      ]
    })
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile(
      'mixed-platforms.json',
      JSON.stringify({
        proxies: [],
        accounts: [{ platform: 'openai' }, { platform: 'gemini' }]
      })
    )])

    await input.trigger('change')
    await flushPromises()

    expect(wrapper.get('[data-test="import-groups"]').attributes('data-group-ids')).toBe('5')
  })

  it('waits for platform detection before importing selected groups', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0
    })
    let resolveText!: (value: string) => void
    const content = JSON.stringify({
      proxies: [],
      accounts: [{ name: 'a', platform: 'openai' }]
    })
    const file = new File([content], 'delayed.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: vi.fn()
        .mockImplementationOnce(() => new Promise<string>((resolve) => { resolveText = resolve }))
        .mockResolvedValue(content)
    })
    const wrapper = mountModal({ groups: [{ id: 3, platform: 'gemini' }] })
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [file])

    await input.trigger('change')
    expect(wrapper.get('[data-test="import-groups"]').attributes('data-group-ids')).toBe('')
    await wrapper.get('[data-test="import-groups"]').trigger('click')
    const submit = wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()

    resolveText(content)
    await submit
    await flushPromises()
    await continueWithoutProxy(wrapper)

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.objectContaining({ accounts: [{ name: 'a', platform: 'openai' }] }),
      skip_default_group_bind: true
    })
  })

  it('部分成功时关闭弹窗仍通知父组件刷新', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 1
    })

    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [
      makeJsonFile(
        'mixed.json',
        JSON.stringify({
          exported_at: '2026-07-05T00:00:00Z',
          proxies: [],
          accounts: [{ name: 'a' }, { name: 'b' }]
        })
      )
    ])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await continueWithoutProxy(wrapper)

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportCompletedWithErrors')
    expect(wrapper.emitted('imported')).toBeUndefined()

    // 第二个 btn-secondary 是 footer 的取消按钮(第一个是选择文件)
    await wrapper.findAll('button.btn-secondary')[1]!.trigger('click')

    expect(wrapper.emitted('imported')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('parses first and confirms only accounts without a file or default proxy', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 1,
      proxy_failed: 0,
      account_created: 2,
      account_failed: 0
    })
    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile(
      'proxy-priority.json',
      JSON.stringify({
        proxies: [{ proxy_key: 'file-proxy' }],
        accounts: [
          { name: 'mapped', proxy_key: 'file-proxy' },
          { name: 'direct' }
        ]
      })
    )])

    await input.trigger('change')
    await flushPromises()
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
    expect(wrapper.get('[data-test="proxyless-confirm"]').attributes('data-message'))
      .toBe('admin.accounts.dataImportNoProxyConfirmMessage:1')

    await continueWithoutProxy(wrapper)
    expect(adminAPI.accounts.importData).toHaveBeenCalledTimes(1)
  })

  it('cancels without losing the form and invalidates confirmation when the proxy changes', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0
    })
    const wrapper = mountModal({ proxies: [{ id: 7, name: 'Proxy 7' }] })
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile(
      'keep-form.json',
      JSON.stringify({ proxies: [], accounts: [{ name: 'direct' }] })
    )])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    setInputFiles(input.element, [makeJsonFile(
      'replacement.json',
      JSON.stringify({ proxies: [], accounts: [{ name: 'replacement' }] })
    )])
    await input.trigger('change')
    expect(wrapper.find('[data-test="proxyless-confirm"]').exists()).toBe(false)

    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await wrapper.get('[data-test="proxyless-cancel"]').trigger('click')

    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
    expect(wrapper.get('input[type="file"]').exists()).toBe(true)

    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await wrapper.get('[data-test="import-default-proxy"]').setValue('7')
    expect(wrapper.find('[data-test="proxyless-confirm"]').exists()).toBe(false)

    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(adminAPI.accounts.importData).toHaveBeenCalledWith(expect.objectContaining({
      default_proxy_id: 7
    }))
  })
})
