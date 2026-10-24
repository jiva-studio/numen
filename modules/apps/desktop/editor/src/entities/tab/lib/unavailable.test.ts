import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createUnavailableKind } from './unavailable'

describe('createUnavailableKind', () => {
  it('keeps the name of the missing kind and what the tab stood on', () => {
    const kind = createUnavailableKind('reader:reader-pdf')

    const state = kind.open('/a.pdf', 'tab')

    expect(kind.kind).toBe('reader:reader-pdf')
    expect(kind.getOpenTab?.(state)).toEqual({ path: '/a.pdf' })
    expect(kind.identity?.('/a.pdf')).toBe('/a.pdf')
  })

  it('says that the plugin is unavailable and names the kind and the path', () => {
    const kind = createUnavailableKind('reader:reader-pdf')

    const view = mount(kind.pane, { props: { state: kind.open('/a.pdf', 'tab') } })

    expect(view.text()).toContain('Plugin unavailable')
    expect(view.text()).toContain('reader:reader-pdf')
    expect(view.text()).toContain('/a.pdf')
  })

  it('dispatches numen:open-settings event when Open Settings button is clicked', async () => {
    const kind = createUnavailableKind('reader:reader-pdf')
    const view = mount(kind.pane, { props: { state: kind.open('/a.pdf', 'tab') } })

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
