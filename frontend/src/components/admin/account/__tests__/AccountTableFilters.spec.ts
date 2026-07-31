import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountTableFilters from '../AccountTableFilters.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  template: '<div data-test="select"></div>'
}

describe('AccountTableFilters', () => {
  it('keeps secondary filters collapsed and emits chip removal', async () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: { platform: 'openai', type: '', status: '', privacy_mode: '', group: '', uploader_user_id: '' },
        resultAccountCount: 4,
        resultBatchCount: 1
      },
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: { name: 'SearchInput', template: '<input />' }
        }
      }
    })

    expect(wrapper.findAll('[data-test="select"]')).toHaveLength(3)
    const moreFilters = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.moreFilters'))
    await moreFilters?.trigger('click')
    expect(wrapper.findAll('[data-test="select"]')).toHaveLength(6)

    const platformChip = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.columns.platform'))
    await platformChip?.trigger('click')
    expect(wrapper.emitted('update:filters')?.at(-1)?.[0]).toMatchObject({ platform: '' })
  })
})
