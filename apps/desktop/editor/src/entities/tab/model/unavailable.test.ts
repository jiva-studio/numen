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

  it('preserves the original tab title when provided and falls back to the formatted kind title', () => {
    const withTitle = createUnavailableKind('files', { title: 'Files' })
    expect(withTitle.getTitle?.(withTitle.open('', 'tab'))).toBe('Files')

    const withoutTitle = createUnavailableKind('files')
    expect(withoutTitle.getTitle?.(withoutTitle.open('', 'tab'))).toBe('Files')
  })

  it('says that the plugin is disabled with a friendly feature name and omits the raw file path', () => {
    const kind = createUnavailableKind('reader:markdown')

    const view = mount(kind.pane, { props: { state: kind.open('cards/Place.md', 'tab') } })

    expect(view.text()).toContain('Plugin disabled')
    expect(view.text()).toContain('The plugin for Markdown notes is disabled or missing.')
    expect(view.text()).not.toContain('cards/Place.md')
  })

  it('dispatches numen:open-settings event with the feature filter when Open Settings button is clicked', async () => {
    const kind = createUnavailableKind('reader:markdown')
    const view = mount(kind.pane, { props: { state: kind.open('cards/Place.md', 'tab') } })

    let receivedFilter: string | undefined
    const listener = (event: Event) => {
      receivedFilter = (event as CustomEvent<{ filter?: string }>).detail?.filter
    }
    globalThis.addEventListener('numen:open-settings', listener)

    const button = view.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toBe('Open Settings')

    await button.trigger('click')
    expect(receivedFilter).toBe('reader:markdown')

    globalThis.removeEventListener('numen:open-settings', listener)
  })
})
