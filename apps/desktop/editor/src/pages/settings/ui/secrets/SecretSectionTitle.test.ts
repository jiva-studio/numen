/**
 * The header of a section: renaming a group in place, and the names the person may not give it.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SecretSectionTitle from './SecretSectionTitle.vue'
import { chooseGroupAction } from '@/testing/secretsDom'

const mountTitle = (props: Record<string, unknown> = {}) =>
  mount(SecretSectionTitle, {
    attachTo: document.body,
    props: {
      group: 'harbour',
      groups: ['harbour', 'quay'],
      headingId: 'heading',
      isAsked: false,
      isNew: false,
      ...props,
    },
  })

const renameTo = async (name: string, props: Record<string, unknown> = {}) => {
  const wrapper = mountTitle(props)
  await chooseGroupAction(/rename group/i)
  const input = wrapper.get<HTMLInputElement>('input[type="text"]')
  await input.setValue(name)
  await input.trigger('keydown', { key: 'Enter' })
  await flushPromises()
  return wrapper
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the name of a new group', () => {
  it('is selected for typing', async () => {
    const wrapper = mountTitle({ isNew: true })
    await flushPromises()
    const input = document.querySelector<HTMLInputElement>('input[aria-label="Name of the group"]')
    expect(input?.value).toBe('harbour')
    expect(document.activeElement).toBe(input)
    wrapper.unmount()
  })
})

describe('renaming a group', () => {
  it('starts from the name of the group, not the title of its service', async () => {
    const wrapper = mountTitle({ group: 'openai', groups: ['openai'] })
    await chooseGroupAction(/rename group/i)
    expect(wrapper.get<HTMLInputElement>('input[type="text"]').element.value).toBe('openai')
    wrapper.unmount()
  })

  it('emits the name without U+0085 at its ends, as the core trims it', async () => {
    const wrapper = await renameTo('\u0085pier')
    expect(wrapper.emitted('rename')).toEqual([['pier']])
    wrapper.unmount()
  })

  it('emits the name with U+FEFF kept at its ends, as the core keeps it', async () => {
    const wrapper = await renameTo('﻿pier')
    expect(wrapper.emitted('rename')).toEqual([['﻿pier']])
    wrapper.unmount()
  })

  it('accepts a name of 64 characters', async () => {
    const wrapper = await renameTo('g'.repeat(64))
    expect(wrapper.emitted('rename')).toEqual([['g'.repeat(64)]])
    wrapper.unmount()
  })
})

describe('a name the group may not have', () => {
  it('is refused above 64 characters, with an alert, and the field stays open', async () => {
    const wrapper = await renameTo('g'.repeat(65))
    expect(wrapper.emitted('rename')).toBeUndefined()
    expect(wrapper.get('[role="alert"]').text()).toBe('Use at most 64 characters.')
    expect(wrapper.get('input[type="text"]').attributes('aria-invalid')).toBe('true')
    wrapper.unmount()
  })

  it('is refused where another group has it, with an alert, and the field stays open', async () => {
    const wrapper = await renameTo('quay')
    expect(wrapper.emitted('rename')).toBeUndefined()
    expect(wrapper.get('[role="alert"]').text()).toBe('A group named quay already exists.')
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('is accepted where it differs from another group in case, as the core compares names exactly', async () => {
    const wrapper = await renameTo('Quay')
    expect(wrapper.emitted('rename')).toEqual([['Quay']])
    wrapper.unmount()
  })

  it('is accepted where it is the name the group has', async () => {
    const wrapper = await renameTo('harbour')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.emitted('rename')).toBeUndefined()
    wrapper.unmount()
  })

  it('loses its alert when the rename is cancelled', async () => {
    const wrapper = await renameTo('quay')
    await wrapper.get('input[type="text"]').trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
