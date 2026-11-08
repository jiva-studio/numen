/**
 * The section of one group as a landmark: its accessible name is the heading with the group name.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SecretSection from './SecretSection.vue'
import {
  chooseGroupAction,
  click,
  findButton,
  findButtons,
  findMenuItem,
} from '@/testing/secretsDom'

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

  it('shows one empty draft row named Default in an empty group the person made', () => {
    const wrapper = mountSection()
    expect(document.querySelectorAll('input[type="password"]')).toHaveLength(1)
    expect(document.querySelector('button[title="Discard"]')).toBeNull()
    wrapper.unmount()
  })

  it('names the empty draft row after the service of a made group', async () => {
    const create = vi.fn(async () => true)
    const wrapper = mount(SecretSection, {
      attachTo: document.body,
      props: {
        section: { group: 'openai', isAsked: false, secrets: [] },
        drafts: [],
        declarations: [],
        create,
      },
    })
    const input = wrapper.get<HTMLInputElement>('input[type="password"]')
    await input.setValue('a-made-up-key')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(create).toHaveBeenCalledWith('OpenAI', 'a-made-up-key')
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
    await chooseGroupAction(/rename group/i)
    const id = document.querySelector('section')?.getAttribute('aria-labelledby') ?? ''
    expect(document.getElementById(id)?.textContent).toBe('harbour')
    wrapper.unmount()
  })

  it('is renamed to the name without U+0085 at its ends, as the core trims it', async () => {
    const wrapper = mountSection()
    await chooseGroupAction(/rename group/i)
    const input = wrapper.get<HTMLInputElement>('input[type="text"]')
    await input.setValue('\u0085quay')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(wrapper.emitted('rename-group')).toEqual([['quay']])
    wrapper.unmount()
  })

  it('is renamed with U+FEFF kept at its ends, as the core keeps it', async () => {
    const wrapper = mountSection()
    await chooseGroupAction(/rename group/i)
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
    await chooseGroupAction(/delete group/i)
    expect(wrapper.emitted('delete-group')).toHaveLength(1)
    expect(wrapper.emitted('delete-group')?.[0]?.[0]).toBe(findButton(/group actions/i))
    wrapper.unmount()
  })

  it('offers only adding a key in a group a plugin asks for', () => {
    const wrapper = mountWith(true, [secret])
    expect(findButton(/^add key$/i)).toBeTruthy()
    expect(findButtons(/group actions/i)).toHaveLength(0)
    wrapper.unmount()
  })

  it('offers renaming and deleting in the group menu, and nothing else', async () => {
    const wrapper = mountWith(false, [secret])
    await click(findButton(/group actions/i))
    expect(
      Array.from(document.querySelectorAll('[role="menuitem"]')).map((one) =>
        one.textContent?.trim(),
      ),
    ).toEqual(['Rename group', 'Delete group'])
    wrapper.unmount()
  })

  it('offers deleting a group the person made with no secret', async () => {
    const wrapper = mountWith(false, [])
    await click(findButton(/group actions/i))
    expect(findMenuItem(/delete group/i)).toBeTruthy()
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
    await chooseGroupAction(/rename group/i)
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
    await chooseGroupAction(/rename group/i)
    expect(wrapper.get<HTMLInputElement>('input[type="text"]').element.value).toBe('openai')
    wrapper.unmount()
  })
})

describe('the columns of a card', () => {
  const key = (id: string, name: string) => ({
    id,
    name,
    group: 'harbour',
    allowed: [],
    hasValue: true,
  })
  const mountKeys = (names: readonly string[], drafts: readonly string[] = []) =>
    mount(SecretSection, {
      attachTo: document.body,
      props: {
        section: {
          group: 'harbour',
          isAsked: false,
          secrets: names.map((one, index) => key(`s${index}`, one)),
        },
        drafts,
        declarations: [],
        create: vi.fn(),
      },
    })

  it('labels Value and Used by, without Key or a name input, over one key', () => {
    const wrapper = mountKeys(['Work'])
    const labels = Array.from(document.querySelectorAll('.secret-labels__text')).map(
      (one) => one.textContent,
    )
    expect(labels).toEqual(['Value', 'Used by'])
    expect(document.querySelector('input[type="text"]')).toBeNull()
    wrapper.unmount()
  })

  it('labels Key and shows the names over several keys, once', () => {
    const wrapper = mountKeys(['Work', 'Home'])
    const labels = Array.from(document.querySelectorAll('.secret-labels__text')).map(
      (one) => one.textContent,
    )
    expect(labels).toEqual(['Key', 'Value', 'Used by'])
    expect(document.querySelectorAll('input[type="text"]')).toHaveLength(2)
    wrapper.unmount()
  })

  it('shows the names once a draft joins one saved key', () => {
    const wrapper = mountKeys(['Work'], ['harbour V1'])
    expect(document.querySelectorAll('input[type="text"]')).toHaveLength(2)
    wrapper.unmount()
  })

  it('emits add from the Add key button', async () => {
    const wrapper = mountKeys(['Work'])
    await click(findButton(/^add key$/i))
    expect(wrapper.emitted('add')).toHaveLength(1)
    wrapper.unmount()
  })
})
