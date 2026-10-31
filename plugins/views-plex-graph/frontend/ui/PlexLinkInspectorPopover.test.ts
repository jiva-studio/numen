import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { ArrowLeft, ArrowLeftRight, ArrowRight, Minus } from '@lucide/vue'
import type { Connection } from '@numen/plugin-sdk'
import PlexLinkInspectorPopover from './PlexLinkInspectorPopover.vue'
import type { LinkInspectorRequest } from '../types'
import { WORDS as words } from '../words'

describe('PlexLinkInspectorPopover component', () => {
  const connection: Connection = {
    first: 'notes/alpha.md',
    second: 'notes/beta.md',
    role: 'jump',
    type: '',
    firstSide: { label: 'Relates to', marker: null, reason: 'kept' },
    secondSide: null,
  }

  const defaultRequest: LinkInspectorRequest = {
    nodeA: { id: 'n1', title: 'Note Alpha', path: 'notes/alpha.md' },
    nodeB: { id: 'n2', title: 'Note Beta', path: 'notes/beta.md' },
    connection,
    roles: ['parent', 'child', 'jump'],
    at: { x: 300, y: 200 },
  }

  const mountPopover = (request: LinkInspectorRequest = defaultRequest) => {
    return mount(PlexLinkInspectorPopover, {
      props: { request },
      attachTo: document.body,
    })
  }

  const saved = (wrapper: ReturnType<typeof mountPopover>) =>
    wrapper.emitted('save')?.[0]?.[0] as Connection

  it('renders positioned at resolved coordinates and shows node titles', () => {
    const wrapper = mountPopover()
    const popover = wrapper.get('.plex-link-popover')

    expect(popover.attributes('style')).toContain('left: 140px')
    expect(popover.attributes('style')).toContain('top: 28px')
    expect(wrapper.text()).toContain('Note Alpha')
    expect(wrapper.text()).toContain('Note Beta')
    expect(wrapper.findComponent(ArrowLeftRight).exists()).toBe(true)
    wrapper.unmount()
  })

  it('clamps left position and flips below when placed near top-left boundary', () => {
    const wrapper = mountPopover({ ...defaultRequest, at: { x: 20, y: 40 } })
    const popover = wrapper.get('.plex-link-popover')

    expect(popover.attributes('style')).toContain('left: 12px')
    expect(popover.attributes('style')).toContain('top: 52px') // 40 + 12
    wrapper.unmount()
  })

  it('shows one row for each of the two notes, the one that writes nothing unchecked and unlabelled', () => {
    const wrapper = mountPopover()

    const presence = wrapper.findAll('.plex-link-row__present')
    const labels = wrapper.findAll('.plex-link-row__input')
    expect(presence.map((one) => (one.element as HTMLInputElement).checked)).toEqual([true, false])
    expect(labels.map((one) => (one.element as HTMLInputElement).value)).toEqual(['Relates to', ''])
    expect(labels[1]?.attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })

  it('offers each kind of link the host names, with the one the connection has chosen', () => {
    const wrapper = mountPopover()
    const select = wrapper.get('select').element as HTMLSelectElement

    expect([...select.options].map((option) => option.value)).toEqual(['parent', 'child', 'jump'])
    expect(select.value).toBe('jump')
    expect(wrapper.get('select').text()).toContain('Note Beta is the child of Note Alpha')
    wrapper.unmount()
  })

  it('saves the kind of link chosen and leaves the records as they were', async () => {
    const wrapper = mountPopover()
    await wrapper.get('select').setValue('parent')
    await wrapper.get('.plex-link-popover__btn--done').trigger('click')

    expect(saved(wrapper)).toEqual({ ...connection, role: 'parent' })
    wrapper.unmount()
  })

  it('switches the arrow of one record without touching its label', async () => {
    const wrapper = mountPopover()
    const arrow = wrapper.findAll('.plex-link-row__arrow-btn')[0]!
    expect(arrow.findComponent(Minus).exists()).toBe(true)

    await arrow.trigger('click')
    expect(arrow.attributes('aria-pressed')).toBe('true')
    expect(arrow.findComponent(ArrowRight).exists()).toBe(true)
    await wrapper.get('.plex-link-popover__btn--done').trigger('click')

    expect(saved(wrapper).firstSide).toEqual({
      label: 'Relates to',
      marker: 'arrow',
      reason: 'kept',
    })
    wrapper.unmount()
  })

  it('draws the arrow of the record the second note writes pointing back at the first', async () => {
    const wrapper = mountPopover()
    await wrapper.findAll('.plex-link-row__present')[1]!.setValue(true)
    const arrow = wrapper.findAll('.plex-link-row__arrow-btn')[1]!
    await arrow.trigger('click')

    expect(arrow.findComponent(ArrowLeft).exists()).toBe(true)
    wrapper.unmount()
  })

  it('saves a record with an arrow and an empty label', async () => {
    const wrapper = mountPopover()
    await wrapper.findAll('.plex-link-row__present')[1]!.setValue(true)
    await wrapper.findAll('.plex-link-row__arrow-btn')[1]!.trigger('click')
    await wrapper.get('.plex-link-popover__btn--done').trigger('click')

    expect(saved(wrapper).secondSide).toEqual({ label: '', marker: 'arrow', reason: '' })
    wrapper.unmount()
  })

  it('takes a record away when its box is cleared', async () => {
    const wrapper = mountPopover()
    await wrapper.findAll('.plex-link-row__present')[0]!.setValue(false)
    await wrapper.get('.plex-link-popover__btn--done').trigger('click')

    expect(saved(wrapper).firstSide).toBeNull()
    wrapper.unmount()
  })

  it('updates the label locally and saves on Enter', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('.plex-link-row__input')
    await input.setValue('Causes')
    await input.trigger('keydown', { key: 'Enter' })

    expect(saved(wrapper).firstSide?.label).toBe('Causes')
    wrapper.unmount()
  })

  it('names every label field by a label of its own', () => {
    const wrapper = mountPopover()

    const fields = wrapper
      .findAll('.plex-link-row__input')
      .map((input) => input.element as HTMLInputElement)
    expect(
      fields.map((field) => [...(field.labels ?? [])].map((label) => label.textContent)),
    ).toEqual([[words.linkDescription], [words.linkDescription]])
    expect(new Set(fields.map((field) => field.id)).size).toBe(2)
    wrapper.unmount()
  })

  it('keeps the rows under the same keys while a record is added and taken away', async () => {
    const wrapper = mountPopover()
    const rows = () => wrapper.findAll('.plex-link-row').map((row) => row.element)
    const before = rows()

    await wrapper.findAll('.plex-link-row__present')[1]!.setValue(true)
    await wrapper.findAll('.plex-link-row__present')[0]!.setValue(false)

    expect(rows()).toEqual(before)
    wrapper.unmount()
  })

  it('asks for the removal of the connection when the remove button is clicked', async () => {
    const wrapper = mountPopover()
    await wrapper.get('.plex-link-popover__btn--remove').trigger('click')

    expect(wrapper.emitted('remove')).toBeTruthy()
    expect(wrapper.emitted('save')).toBeFalsy()
    wrapper.unmount()
  })

  it('emits dismiss on escape key', async () => {
    const wrapper = mountPopover()
    await wrapper.get('.plex-link-popover').trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('dismiss')).toBeTruthy()
    wrapper.unmount()
  })

  it('emits dismiss and does not save when close/cancel button is clicked', async () => {
    const wrapper = mountPopover()
    await wrapper.get('.plex-link-row__input').setValue('Modified Text')

    await wrapper.get('.plex-link-popover__close-btn').trigger('click')

    expect(wrapper.emitted('dismiss')).toBeTruthy()
    expect(wrapper.emitted('save')).toBeFalsy()
    wrapper.unmount()
  })

  it('saves and closes when clicking outside the popover', () => {
    const wrapper = mountPopover()
    const outsideEl = document.createElement('div')
    document.body.appendChild(outsideEl)

    const event = new PointerEvent('pointerdown', { bubbles: true })
    Object.defineProperty(event, 'target', { value: outsideEl, writable: false })
    window.dispatchEvent(event)

    expect(wrapper.emitted('save')).toBeTruthy()
    wrapper.unmount()
    outsideEl.remove()
  })

  it('gives a connection with no records the new link role once a record is added', async () => {
    const empty = { ...connection, role: null, firstSide: null }
    const wrapper = mountPopover({ ...defaultRequest, connection: empty })
    expect(wrapper.get('select').element.querySelector('option[disabled]')).not.toBeNull()

    await wrapper.findAll('.plex-link-row__present')[0]!.setValue(true)
    await wrapper.get('.plex-link-popover__btn--done').trigger('click')

    expect(saved(wrapper)).toMatchObject({ role: 'jump', firstSide: { label: '', marker: null } })
    wrapper.unmount()
  })

  it('updates position when window is resized or request prop updates', async () => {
    const wrapper = mountPopover()
    const popover = wrapper.get('.plex-link-popover')

    window.dispatchEvent(new Event('resize'))
    expect(wrapper.find('.plex-link-popover').exists()).toBe(true)

    await wrapper.setProps({ request: { ...defaultRequest, at: { x: 500, y: 350 } } })
    await nextTick()

    expect(popover.attributes('style')).toContain('left: 340px')
    expect(popover.attributes('style')).toContain('top: 178px')
    wrapper.unmount()
  })
})
