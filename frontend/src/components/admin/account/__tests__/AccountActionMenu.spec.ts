import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountActionMenu from '../AccountActionMenu.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const account = {
  id: 7,
  name: 'console-account',
  platform: 'openai',
  type: 'apikey',
  status: 'active',
  schedulable: true,
  quota_limit: 10
} as Account

describe('AccountActionMenu dialog', () => {
  it('uses a page dialog and keeps destructive actions on the existing emit flow', async () => {
    const wrapper = mount(AccountActionMenu, {
      props: { show: true, account },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show'],
            emits: ['close'],
            template: '<section v-if="show" data-test="page-dialog"><slot /></section>'
          },
          Icon: true
        }
      }
    })

    expect(wrapper.get('[data-test="page-dialog"]').exists()).toBe(true)
    const buttons = wrapper.findAll('button')
    await buttons.find(button => button.text() === 'common.edit')!.trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual([account])

    await wrapper.findAll('button').find(button => button.text() === 'common.delete')!.trigger('click')
    expect(wrapper.emitted('delete')?.[0]).toEqual([account])
    expect(wrapper.emitted('close')).toHaveLength(2)
  })
})
