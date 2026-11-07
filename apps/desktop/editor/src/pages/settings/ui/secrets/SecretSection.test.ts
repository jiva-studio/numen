/**
 * The section of one group as a landmark: its accessible name is the heading with the group name.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SecretSection from './SecretSection.vue'

const section = { group: 'harbour', isAsked: false, secrets: [] }

const mountSection = () =>
  mount(SecretSection, {
    attachTo: document.body,
    props: { section, drafts: [], declarations: [] },
  })

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the section of a group', () => {
  it('is named by its heading', () => {
    const wrapper = mountSection()
    const region = document.querySelector('section')
    const id = region?.getAttribute('aria-labelledby') ?? ''
    expect(id).not.toBe('')
    expect(document.getElementById(id)?.textContent).toBe('harbour')
    expect(document.getElementById(id)?.tagName).toBe('H2')
    wrapper.unmount()
  })

  it('keeps its name while the group is being renamed', async () => {
    const wrapper = mountSection()
    await wrapper.get('button[aria-label="Rename group"]').trigger('click')
    const id = document.querySelector('section')?.getAttribute('aria-labelledby') ?? ''
    expect(document.getElementById(id)?.textContent).toBe('harbour')
    wrapper.unmount()
  })

  it('is renamed to the name without U+0085 at its ends, as the core trims it', async () => {
    const wrapper = mountSection()
    await wrapper.get('button[aria-label="Rename group"]').trigger('click')
    const input = wrapper.get<HTMLInputElement>('input[type="text"]')
    await input.setValue('\u0085quay')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(wrapper.emitted('rename-group')).toEqual([['quay']])
    wrapper.unmount()
  })

  it('is renamed with U+FEFF kept at its ends, as the core keeps it', async () => {
    const wrapper = mountSection()
    await wrapper.get('button[aria-label="Rename group"]').trigger('click')
    const input = wrapper.get<HTMLInputElement>('input[type="text"]')
    await input.setValue('\uFEFFquay')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(wrapper.emitted('rename-group')).toEqual([['\uFEFFquay']])
    wrapper.unmount()
  })
})
