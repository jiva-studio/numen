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
    first: 'notes/harbour.md',
    second: 'notes/lighthouse.md',
    role: 'child',
    type: '',
    firstSide: { label: 'guides ships', marker: null, reason: 'kept' },
    secondSide: null,
  }

  const defaultRequest: LinkInspectorRequest = {
    nodeA: { id: 'n1', title: 'Harbour', path: 'notes/harbour.md' },
    nodeB: { id: 'n2', title: 'Lighthouse', path: 'notes/lighthouse.md' },
    connection,
    at: { x: 300, y: 200 },
  }

  const mountPopover = (request: LinkInspectorRequest = defaultRequest) =>
    mount(PlexLinkInspectorPopover, {
      props: { request },
      attachTo: document.body,
    })

  type Wrapper = ReturnType<typeof mountPopover>

  const saved = (wrapper: Wrapper) => wrapper.emitted('save')?.[0]?.[0] as Connection

  const directionButtons = (wrapper: Wrapper) => wrapper.findAll('.plex-link-row__dir-btn')

  const getIcons = (wrapper: Wrapper) =>
    directionButtons(wrapper).map((button) => {
      if (button.findComponent(ArrowRight).exists()) return '→'
      if (button.findComponent(ArrowLeft).exists()) return '←'
      return button.findComponent(Minus).exists() ? '—' : '?'
    })

  const clickDone = (wrapper: Wrapper) =>
    wrapper.get('.plex-link-popover__btn--done').trigger('click')

  const clickAdd = (wrapper: Wrapper) =>
    wrapper.get('.plex-link-popover__btn--add').trigger('click')

  it('renders positioned at resolved coordinates and shows node titles', () => {
    const wrapper = mountPopover()
    const popover = wrapper.get('.plex-link-popover')

    expect(popover.attributes('style')).toContain('left: 140px')
    expect(popover.attributes('style')).toContain('top: 28px')
    expect(wrapper.text()).toContain('Harbour')
    expect(wrapper.text()).toContain('Lighthouse')
    expect(wrapper.findComponent(ArrowLeftRight).exists()).toBe(true)
    wrapper.unmount()
  })

  it('clamps left position and flips below when placed near top-left boundary', () => {
    const wrapper = mountPopover({ ...defaultRequest, at: { x: 20, y: 40 } })
    const popover = wrapper.get('.plex-link-popover')

    expect(popover.attributes('style')).toContain('left: 12px')
    expect(popover.attributes('style')).toContain('top: 52px')
    wrapper.unmount()
  })

  it('shows one row with a direction button, the description and a trash button', () => {
    const wrapper = mountPopover()

    expect(wrapper.findAll('.plex-link-row')).toHaveLength(1)
    expect(getIcons(wrapper)).toEqual(['—'])
    expect((wrapper.get('.plex-link-row__input').element as HTMLInputElement).value).toBe(
      'guides ships',
    )
    expect(wrapper.findAll('.plex-link-row__delete-btn')).toHaveLength(1)
    expect(wrapper.get('.plex-link-popover__btn--add').text()).toContain(words.addReverseDirection)
    wrapper.unmount()
  })

  it('shows no kind of link and no checkbox', () => {
    const wrapper = mountPopover()

    expect(wrapper.find('select').exists()).toBe(false)
    expect(wrapper.find('input[type="checkbox"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('cycles the direction through none, forward, backward and none again', async () => {
    const wrapper = mountPopover()
    const directions = [getIcons(wrapper)[0]]

    for (let step = 0; step < 3; step += 1) {
      await directionButtons(wrapper)[0]!.trigger('click')
      directions.push(getIcons(wrapper)[0])
    }

    expect(directions).toEqual(['—', '→', '←', '—'])
    expect(wrapper.emitted('save')).toBeFalsy()
    wrapper.unmount()
  })

  it('keeps the same direction button while the record moves to the other note', async () => {
    const wrapper = mountPopover()
    const button = directionButtons(wrapper)[0]!.element

    await directionButtons(wrapper)[0]!.trigger('click')
    await directionButtons(wrapper)[0]!.trigger('click')

    expect(getIcons(wrapper)).toEqual(['←'])
    expect(directionButtons(wrapper)[0]!.element).toBe(button)
    wrapper.unmount()
  })

  it('saves a forward direction as an arrow on the record of the first note', async () => {
    const wrapper = mountPopover()
    await directionButtons(wrapper)[0]!.trigger('click')
    await clickDone(wrapper)

    expect(saved(wrapper)).toStrictEqual({
      ...connection,
      firstSide: { label: 'guides ships', marker: 'arrow', reason: 'kept' },
    })
    wrapper.unmount()
  })

  it('saves a backward direction as the record of the second note, with the label and the role', async () => {
    const wrapper = mountPopover()
    await directionButtons(wrapper)[0]!.trigger('click')
    await directionButtons(wrapper)[0]!.trigger('click')
    await clickDone(wrapper)

    expect(saved(wrapper)).toStrictEqual({
      ...connection,
      role: 'child',
      firstSide: null,
      secondSide: { label: 'guides ships', marker: 'arrow', reason: 'kept' },
    })
    wrapper.unmount()
  })

  it('updates the description locally and saves on Enter', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('.plex-link-row__input')

    await input.setValue('keeps watch')
    await input.trigger('keydown', { key: 'Enter' })

    expect(saved(wrapper).firstSide?.label).toBe('keeps watch')
    wrapper.unmount()
  })

  it('names every description field by a label of its own', async () => {
    const wrapper = mountPopover()
    await clickAdd(wrapper)

    const inputs = wrapper.findAll('.plex-link-row__input')
    expect(inputs).toHaveLength(2)
    for (const input of inputs) {
      const id = input.attributes('id')
      expect(id).toBeTruthy()
      expect(wrapper.find(`label[for="${id}"]`).text()).toBe(words.linkDescription)
    }
    wrapper.unmount()
  })

  it('adds the reverse direction as a second row with both arrows', async () => {
    const wrapper = mountPopover()
    await clickAdd(wrapper)

    expect(getIcons(wrapper)).toEqual(['→', '←'])
    expect(wrapper.find('.plex-link-popover__btn--add').exists()).toBe(false)
    wrapper.unmount()
  })

  it('names the arrow of a record and the one a click makes', async () => {
    const wrapper = mountPopover()
    const button = directionButtons(wrapper)[0]!

    expect(button.attributes('aria-label')).toBe('No arrow. Click for: Arrow to Lighthouse')
    expect(button.attributes('title')).toBe(button.attributes('aria-label'))
    await button.trigger('click')
    expect(button.attributes('aria-label')).toBe('Arrow to Lighthouse. Click for: Arrow to Harbour')
    await button.trigger('click')
    expect(button.attributes('aria-label')).toBe('Arrow to Harbour. Click for: No arrow')
    wrapper.unmount()
  })

  it('turns the arrow of each of two undirected records on and off on its own', async () => {
    const wrapper = mountPopover({
      ...defaultRequest,
      connection: {
        ...connection,
        secondSide: { label: 'shelters it', marker: null, reason: '' },
      },
    })
    expect(getIcons(wrapper)).toEqual(['—', '—'])
    expect(directionButtons(wrapper).every((button) => !button.attributes('disabled'))).toBe(true)

    await directionButtons(wrapper)[1]!.trigger('click')
    expect(getIcons(wrapper)).toEqual(['—', '←'])
    await directionButtons(wrapper)[0]!.trigger('click')
    expect(getIcons(wrapper)).toEqual(['→', '←'])
    await directionButtons(wrapper)[1]!.trigger('click')
    expect(getIcons(wrapper)).toEqual(['→', '—'])

    await clickDone(wrapper)
    expect(saved(wrapper)).toStrictEqual({
      ...connection,
      firstSide: { label: 'guides ships', marker: 'arrow', reason: 'kept' },
      secondSide: { label: 'shelters it', marker: null, reason: '' },
    })
    wrapper.unmount()
  })

  it('saves both records with arrows and their own descriptions', async () => {
    const wrapper = mountPopover()
    await clickAdd(wrapper)
    await wrapper.findAll('.plex-link-row__input')[1]!.setValue('shelters it')
    await clickDone(wrapper)

    expect(saved(wrapper)).toStrictEqual({
      ...connection,
      firstSide: { label: 'guides ships', marker: 'arrow', reason: 'kept' },
      secondSide: { label: 'shelters it', marker: 'arrow', reason: '' },
    })
    wrapper.unmount()
  })

  it('adds the reverse direction on Ctrl+Enter and on Meta+Enter', async () => {
    for (const modifier of ['ctrlKey', 'metaKey'] as const) {
      const wrapper = mountPopover()
      await wrapper
        .get('.plex-link-row__input')
        .trigger('keydown', { key: 'Enter', [modifier]: true })

      expect(wrapper.findAll('.plex-link-row')).toHaveLength(2)
      expect(wrapper.emitted('save')).toBeFalsy()
      wrapper.unmount()
    }
  })

  it('adds the reverse direction on Ctrl+Enter pressed anywhere in the popover', async () => {
    const wrapper = mountPopover()
    await wrapper.get('.plex-link-popover').trigger('keydown', { key: 'Enter', ctrlKey: true })

    expect(wrapper.findAll('.plex-link-row')).toHaveLength(2)
    wrapper.unmount()
  })

  it('removes one row with its trash button and lets the other cycle again', async () => {
    const wrapper = mountPopover()
    await clickAdd(wrapper)
    await wrapper.findAll('.plex-link-row__delete-btn')[0]!.trigger('click')

    expect(getIcons(wrapper)).toEqual(['←'])
    expect(directionButtons(wrapper)[0]!.attributes('disabled')).toBeUndefined()

    await clickDone(wrapper)
    expect(saved(wrapper)).toStrictEqual({
      ...connection,
      firstSide: null,
      secondSide: { label: '', marker: 'arrow', reason: '' },
    })
    wrapper.unmount()
  })

  it('shows the removal notice once the last row is removed and asks to remove the link on Done', async () => {
    const wrapper = mountPopover()
    await wrapper.get('.plex-link-row__delete-btn').trigger('click')

    expect(wrapper.find('.plex-link-row').exists()).toBe(false)
    expect(wrapper.get('.plex-link-popover__empty-state').text()).toBe(words.linkWillBeRemoved)
    expect(wrapper.get('.plex-link-popover__btn--add').text()).toContain(words.addLink)

    await clickDone(wrapper)
    expect(wrapper.emitted('remove')).toHaveLength(1)
    expect(wrapper.emitted('save')).toBeFalsy()
    wrapper.unmount()
  })

  it('adds an undirected row again with Add link', async () => {
    const wrapper = mountPopover()
    await wrapper.get('.plex-link-row__delete-btn').trigger('click')
    await clickAdd(wrapper)

    expect(getIcons(wrapper)).toEqual(['—'])
    await clickDone(wrapper)
    expect(saved(wrapper)).toStrictEqual({
      ...connection,
      firstSide: { label: '', marker: null, reason: '' },
    })
    wrapper.unmount()
  })

  it('gives a connection with no records the new link role once a row is added', async () => {
    const empty: Connection = { ...connection, role: null, firstSide: null }
    const wrapper = mountPopover({ ...defaultRequest, connection: empty })
    expect(wrapper.find('.plex-link-popover__empty-state').exists()).toBe(true)

    await clickAdd(wrapper)
    await clickDone(wrapper)

    expect(saved(wrapper)).toMatchObject({ role: 'jump', firstSide: { label: '', marker: null } })
    wrapper.unmount()
  })

  it('shows a connection both notes write as two rows', () => {
    const wrapper = mountPopover({
      ...defaultRequest,
      connection: {
        ...connection,
        firstSide: { label: 'guides ships', marker: 'arrow', reason: '' },
        secondSide: { label: 'shelters it', marker: 'arrow', reason: '' },
      },
    })

    expect(getIcons(wrapper)).toEqual(['→', '←'])
    expect(
      wrapper
        .findAll('.plex-link-row__input')
        .map((one) => (one.element as HTMLInputElement).value),
    ).toEqual(['guides ships', 'shelters it'])
    wrapper.unmount()
  })

  it('names the note that writes each row', () => {
    const wrapper = mountPopover({
      ...defaultRequest,
      connection: {
        ...connection,
        secondSide: { label: 'shelters it', marker: null, reason: '' },
      },
    })

    expect(wrapper.findAll('.plex-link-row').map((row) => row.attributes('data-from'))).toEqual([
      'Harbour',
      'Lighthouse',
    ])
    wrapper.unmount()
  })

  it('emits dismiss on escape key', async () => {
    const wrapper = mountPopover()
    await wrapper.get('.plex-link-popover').trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('dismiss')).toBeTruthy()
    expect(wrapper.emitted('save')).toBeFalsy()
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

    expect(saved(wrapper)).toStrictEqual(connection)
    wrapper.unmount()
    outsideEl.remove()
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
