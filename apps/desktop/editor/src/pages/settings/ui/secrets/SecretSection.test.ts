/**
 * The section of one group as a landmark: its accessible name is the heading with the group name.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SecretSection from './SecretSection.vue'

const section = { group: 'harbour', isAsked: false, secrets: [] }

const mountSection = (create = vi.fn(async () => true)) =>
  mount(SecretSection, {
    attachTo: document.body,
    props: { section, drafts: [], declarations: [], create },
  })

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the section of a group', () => {
  it('creates the secret of a group a plugin asks for through create', async () => {
    const create = vi.fn(async () => true)
    const wrapper = mount(SecretSection, {
      attachTo: document.body,
      props: {
        section: { group: 'harbour', isAsked: true, secrets: [] },
        drafts: [],
        declarations: [],
        create,
      },
    })
    const input = wrapper.get<HTMLInputElement>('input[type="password"]')
    await input.setValue('a-made-up-key')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(create).toHaveBeenCalledWith('Default', 'a-made-up-key')
    wrapper.unmount()
  })

  it('requires create: a section without it is a type error', () => {
    // vue-tsc fails on the unused directive while the prop is optional, which would drop a typed value.
    // @ts-expect-error create is required
    const mountWithoutCreate = () =>
      mount(SecretSection, { props: { section, drafts: [], declarations: [] } })
    expect(mountWithoutCreate).toBeTypeOf('function')
  })

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
