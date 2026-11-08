/**
 * A secret not created yet: where the focus goes when its row is added, and the name it creates
 * the secret under.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SecretDraftRow from './SecretDraftRow.vue'

const mountRow = (props: Record<string, unknown> = {}) =>
  mount(SecretDraftRow, {
    attachTo: document.body,
    props: {
      name: 'Default V1',
      canDiscard: true,
      hasName: true,
      submit: vi.fn(async () => true),
      ...props,
    },
  })

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the focus of a new row', () => {
  it('goes to the name where the rows have a name column', () => {
    const wrapper = mountRow({ hasAutofocus: true })
    expect(document.activeElement).toBe(document.querySelector('input[type="text"]'))
    wrapper.unmount()
  })

  it('goes to the value where the rows have no name column', () => {
    const wrapper = mountRow({ hasAutofocus: true, hasName: false })
    expect(document.activeElement).toBe(document.querySelector('input[type="password"]'))
    wrapper.unmount()
  })

  it('takes no focus on a row that does not request it', () => {
    const wrapper = mountRow()
    expect(document.activeElement).toBe(document.body)
    wrapper.unmount()
  })
})

describe('the name a row creates a secret under', () => {
  it('is the name typed, trimmed', async () => {
    const submit = vi.fn(async () => true)
    const wrapper = mountRow({ submit })
    await wrapper.get('input[type="text"]').setValue('  Staging ')
    const value = wrapper.get('input[type="password"]')
    await value.setValue('made-up-value')
    await value.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(submit).toHaveBeenCalledWith('Staging', 'made-up-value')
    wrapper.unmount()
  })

  it('creates nothing where the name typed is empty', async () => {
    const submit = vi.fn(async () => true)
    const wrapper = mountRow({ submit })
    await wrapper.get('input[type="text"]').setValue('   ')
    const value = wrapper.get('input[type="password"]')
    await value.setValue('made-up-value')
    await value.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(submit).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
