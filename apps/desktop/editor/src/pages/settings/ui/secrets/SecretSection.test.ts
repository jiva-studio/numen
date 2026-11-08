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

describe('the deletion of a group', () => {
  const secret = { id: 's1', name: 'Work', group: 'harbour', allowed: [], hasValue: true }
  const mountWith = (isAsked: boolean, secrets: readonly (typeof secret)[]) =>
    mount(SecretSection, {
      attachTo: document.body,
      props: {
        section: { group: 'harbour', isAsked, secrets },
        drafts: [],
        declarations: [],
        create: vi.fn(async () => true),
      },
    })

  it('emits delete-group from the button of a group the person made', async () => {
    const wrapper = mountWith(false, [secret])
    await wrapper.get('button[aria-label="Delete group"]').trigger('click')
    expect(wrapper.emitted('delete-group')).toHaveLength(1)
    wrapper.unmount()
  })

  it('has no button for a group a plugin asks for', () => {
    const wrapper = mountWith(true, [secret])
    expect(wrapper.find('button[aria-label="Delete group"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('has a button for a group the person made with no secret', () => {
    const wrapper = mountWith(false, [])
    expect(wrapper.find('button[aria-label="Delete group"]').exists()).toBe(true)
    wrapper.unmount()
  })
})

describe('the name of a group', () => {
  it('selects the name of a new group for typing', async () => {
    const wrapper = mount(SecretSection, {
      attachTo: document.body,
      props: { section, drafts: [], declarations: [], isNew: true, create: vi.fn() },
    })
    await flushPromises()
    const input = document.querySelector<HTMLInputElement>('input[aria-label="Name of the group"]')
    expect(input?.value).toBe('harbour')
    expect(document.activeElement).toBe(input)
    wrapper.unmount()
  })

  const renameTo = async (name: string) => {
    const wrapper = mountSection()
    await wrapper.get('button[aria-label="Rename group"]').trigger('click')
    const input = wrapper.get<HTMLInputElement>('input[type="text"]')
    await input.setValue(name)
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    return wrapper
  }

  it('refuses a group name of more than 64 characters and says so', async () => {
    const wrapper = await renameTo('g'.repeat(65))
    expect(wrapper.emitted('rename-group')).toBeUndefined()
    expect(wrapper.get('[role="alert"]').text()).toBe('Use at most 64 characters.')
    wrapper.unmount()
  })

  it('keeps the field open after a name that is too long', async () => {
    const wrapper = await renameTo('g'.repeat(65))
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('drops the alert when the rename is cancelled', async () => {
    const wrapper = await renameTo('g'.repeat(65))
    await wrapper.get('input[type="text"]').trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('accepts a group name of 64 characters', async () => {
    const wrapper = await renameTo('g'.repeat(64))
    expect(wrapper.emitted('rename-group')).toEqual([['g'.repeat(64)]])
    wrapper.unmount()
  })
})

describe('the card of a well-known service', () => {
  const mountGroup = (group: string) =>
    mount(SecretSection, {
      attachTo: document.body,
      props: {
        section: { group, isAsked: false, secrets: [] },
        drafts: [],
        declarations: [],
        create: vi.fn(),
      },
    })

  it('shows the title and the description of the service', () => {
    const wrapper = mountGroup('openai')
    expect(wrapper.get('h2').text()).toBe('OpenAI')
    expect(wrapper.get('.secret-section__about').text()).toBe(
      'The maker of ChatGPT and the GPT models.',
    )
    wrapper.unmount()
  })

  it('shows only the name of an unknown group', () => {
    const wrapper = mountGroup('harbour')
    expect(wrapper.get('h2').text()).toBe('harbour')
    expect(wrapper.find('.secret-section__about').exists()).toBe(false)
    wrapper.unmount()
  })

  it('renames the key of the group, not its title', async () => {
    const wrapper = mountGroup('openai')
    await wrapper.get('button[aria-label="Rename group"]').trigger('click')
    expect(wrapper.get<HTMLInputElement>('input[type="text"]').element.value).toBe('openai')
    wrapper.unmount()
  })
})
