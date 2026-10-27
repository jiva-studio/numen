import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createUnavailableKind } from './unavailable'

describe('createUnavailableKind', () => {
  it('keeps the name of the missing kind and what the tab stood on', () => {
    const kind = createUnavailableKind('reader:viewer')

    const state = kind.open('/a.bin', 'tab')

    expect(kind.kind).toBe('reader:viewer')
    expect(kind.getOpenTab?.(state)).toEqual({ path: '/a.bin' })
    expect(kind.identity?.('/a.bin')).toBe('/a.bin')
  })

  it('keeps the state the plugin saved for the tab', () => {
    const kind = createUnavailableKind('reader:viewer')

    const state = kind.open('a.bin', 'tab', { page: 4 })

    expect(kind.getState?.(state)).toEqual({ page: 4 })
    expect(kind.getTitle?.(state)).toBe('a.bin')
  })

  it('says that the plugin is disabled and names the kind and the path', () => {
    const kind = createUnavailableKind('reader:viewer')

    const view = mount(kind.pane, { props: { state: kind.open('/a.bin', 'tab') } })

    expect(view.text()).toContain('Plugin disabled')
    expect(view.text()).toContain('reader:viewer')
    expect(view.text()).toContain('/a.bin')
  })

  it('dispatches numen:open-settings event when Open Settings button is clicked', async () => {
    const kind = createUnavailableKind('reader:viewer')
    const view = mount(kind.pane, { props: { state: kind.open('/a.bin', 'tab') } })

    let eventFired = false
    const listener = () => {
      eventFired = true
    }
    globalThis.addEventListener('numen:open-settings', listener)

    const button = view.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toBe('Open Settings')

    await button.trigger('click')
    expect(eventFired).toBe(true)

    globalThis.removeEventListener('numen:open-settings', listener)
  })
})
