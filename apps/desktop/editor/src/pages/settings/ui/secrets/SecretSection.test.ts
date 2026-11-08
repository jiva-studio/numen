/**
 * The section of one group as a landmark: its accessible name is the heading with the group name,
 * and its rows, columns and focus follow the secrets and drafts it is given.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SecretSection from './SecretSection.vue'
import {
  chooseGroupAction,
  chooseRowAction,
  click,
  findButton,
  findButtons,
  findText,
  rowOf,
  textInputs,
} from '@/testing/secretsDom'

const secret = (id: string, name: string, group = 'harbour') => ({
  id,
  name,
  group,
  allowed: [],
  hasValue: true,
})

const mountSection = (props: Record<string, unknown> = {}) =>
  mount(SecretSection, {
    attachTo: document.body,
    props: {
      section: { group: 'harbour', isAsked: false, secrets: [] },
      groups: ['harbour'],
      drafts: [],
      declarations: [],
      create: vi.fn(async () => true),
      ...props,
    },
  })

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the section of a group', () => {
  it('creates the secret of a group a plugin asks for through create', async () => {
    const create = vi.fn(async () => true)
    const wrapper = mountSection({
      section: { group: 'harbour', isAsked: true, secrets: [] },
      create,
    })
    const input = wrapper.get<HTMLInputElement>('input[type="password"]')
    await input.setValue('a-made-up-key')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(create).toHaveBeenCalledWith('Default', 'a-made-up-key')
    wrapper.unmount()
  })

  it('shows one empty row named Default in an empty group the person made, with no way to discard it', () => {
    const wrapper = mountSection()
    expect(document.querySelectorAll('input[type="password"]')).toHaveLength(1)
    expect(findButtons(/^discard$/i)).toHaveLength(0)
    wrapper.unmount()
  })

  it('names the empty row after the service of a made group', async () => {
    const create = vi.fn(async () => true)
    const wrapper = mountSection({
      section: { group: 'openai', isAsked: false, secrets: [] },
      create,
    })
    const input = wrapper.get<HTMLInputElement>('input[type="password"]')
    await input.setValue('a-made-up-key')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(create).toHaveBeenCalledWith('OpenAI', 'a-made-up-key')
    wrapper.unmount()
  })

  it('is named by its heading, also while the group is renamed', async () => {
    const wrapper = mountSection()
    await chooseGroupAction(/rename group/i)
    const id = document.querySelector('section')?.getAttribute('aria-labelledby') ?? ''
    expect(document.getElementById(id)?.textContent).toBe('harbour')
    expect(document.getElementById(id)?.tagName).toBe('H2')
    wrapper.unmount()
  })

  it('emits add from the Add secret button', async () => {
    const wrapper = mountSection({
      section: { group: 'harbour', isAsked: false, secrets: [secret('s1', 'Work')] },
    })
    await click(findButton(/^add secret$/i))
    expect(wrapper.emitted('add')).toHaveLength(1)
    wrapper.unmount()
  })
})

describe('the menu of a group', () => {
  const work = secret('s1', 'Work')

  it('emits delete-group with the button that opened the menu', async () => {
    const wrapper = mountSection({ section: { group: 'harbour', isAsked: false, secrets: [work] } })
    await chooseGroupAction(/delete group/i)
    expect(wrapper.emitted('delete-group')?.[0]?.[0]).toBe(findButton(/group actions/i))
    wrapper.unmount()
  })

  it('is absent in a group a plugin asks for', () => {
    const wrapper = mountSection({ section: { group: 'harbour', isAsked: true, secrets: [work] } })
    expect(findButtons(/group actions/i)).toHaveLength(0)
    wrapper.unmount()
  })

  it('offers renaming and deleting, and nothing else', async () => {
    const wrapper = mountSection({ section: { group: 'harbour', isAsked: false, secrets: [work] } })
    await click(findButton(/group actions/i))
    expect(
      Array.from(document.querySelectorAll('[role="menuitem"]')).map((one) =>
        one.textContent?.trim(),
      ),
    ).toEqual(['Rename group', 'Delete group'])
    wrapper.unmount()
  })
})

describe('the card of a well-known service', () => {
  it('shows the title and the description of the service', () => {
    const wrapper = mountSection({ section: { group: 'openai', isAsked: false, secrets: [] } })
    expect(wrapper.get('h2').text()).toBe('OpenAI')
    expect(findText('The maker of ChatGPT and the GPT models.')).not.toBeNull()
    wrapper.unmount()
  })

  it('shows only the name of an unknown group', () => {
    const wrapper = mountSection()
    expect(wrapper.get('h2').text()).toBe('harbour')
    expect(wrapper.findAll('p')).toHaveLength(0)
    wrapper.unmount()
  })
})

describe('the columns of a card', () => {
  const mountKeys = (
    names: readonly string[],
    drafts: readonly { id: number; name: string }[] = [],
  ) =>
    mountSection({
      section: {
        group: 'harbour',
        isAsked: false,
        secrets: names.map((one, index) => secret(`s${index}`, one)),
      },
      drafts: drafts.map((one) => ({ ...one, group: 'harbour' })),
    })

  it('labels Value and Used by, without Name or a name input, over one secret', () => {
    const wrapper = mountKeys(['Work'])
    expect(findText('Value')).not.toBeNull()
    expect(findText('Used by')).not.toBeNull()
    expect(findText('Name')).toBeNull()
    expect(textInputs()).toHaveLength(0)
    wrapper.unmount()
  })

  it('labels Name and shows the names over several secrets, once', () => {
    const wrapper = mountKeys(['Work', 'Home'])
    expect(findText('Name')).not.toBeNull()
    expect(textInputs()).toHaveLength(2)
    wrapper.unmount()
  })

  it('shows the names once a draft joins one saved secret', () => {
    const wrapper = mountKeys(['Work'], [{ id: 1, name: 'harbour V1' }])
    expect(textInputs()).toHaveLength(2)
    wrapper.unmount()
  })
})

describe('moving a secret', () => {
  const mountThree = () =>
    mountSection({
      section: {
        group: 'harbour',
        isAsked: false,
        secrets: [secret('s1', 'Work'), secret('s2', 'Home'), secret('s3', 'Boat')],
      },
    })

  it('moves one before the secret above it', async () => {
    const wrapper = mountThree()
    await chooseRowAction(rowOf('Home'), /move up/i)
    expect(wrapper.emitted('move')).toEqual([['s2', 's1']])
    wrapper.unmount()
  })

  it('moves one before the secret two places below it', async () => {
    const wrapper = mountThree()
    await chooseRowAction(rowOf('Work'), /move down/i)
    expect(wrapper.emitted('move')).toEqual([['s1', 's3']])
    wrapper.unmount()
  })

  it('moves the last but one to the end', async () => {
    const wrapper = mountThree()
    await chooseRowAction(rowOf('Home'), /move down/i)
    expect(wrapper.emitted('move')).toEqual([['s2', '']])
    wrapper.unmount()
  })
})

describe('the focus in a card', () => {
  const draft = { id: 1, group: 'harbour', name: 'harbour V1' }
  const secrets = [secret('s1', 'Work')]

  it('moves to the name of a draft that is added', async () => {
    const wrapper = mountSection({ section: { group: 'harbour', isAsked: false, secrets } })
    await wrapper.setProps({ drafts: [draft] })
    await flushPromises()
    expect(document.activeElement).toBe(textInputs().find((one) => one.value === 'harbour V1'))
    wrapper.unmount()
  })

  it('does not move to the empty secret of a group on its first draw', () => {
    const wrapper = mountSection()
    expect(document.activeElement).toBe(document.body)
    wrapper.unmount()
  })

  it('returns to the Add secret button when a draft is discarded', async () => {
    const wrapper = mountSection({
      section: { group: 'harbour', isAsked: false, secrets },
      drafts: [draft],
    })
    await click(findButton(/^discard$/i))
    await wrapper.setProps({ drafts: [] })
    await flushPromises()
    expect(wrapper.emitted('discard')).toEqual([[1]])
    expect(document.activeElement).toBe(findButton(/^add secret$/i))
    wrapper.unmount()
  })
})
