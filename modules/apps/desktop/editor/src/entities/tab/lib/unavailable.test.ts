import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createUnavailableKind } from './unavailable'

describe('createUnavailableKind', () => {
  it('keeps the name of the missing kind and what the tab stood on', () => {
    const kind = createUnavailableKind('reader:reader-pdf')

    const state = kind.open('/a.pdf')

    expect(kind.kind).toBe('reader:reader-pdf')
    expect(kind.getOpenTab?.(state)).toEqual({ path: '/a.pdf' })
    expect(kind.identity?.('/a.pdf')).toBe('/a.pdf')
  })

  it('says that the plugin is unavailable and names the kind and the path', () => {
    const kind = createUnavailableKind('reader:reader-pdf')

    const view = mount(kind.pane, { props: { state: kind.open('/a.pdf') } })

    expect(view.text()).toContain('Plugin unavailable')
    expect(view.text()).toContain('reader:reader-pdf')
    expect(view.text()).toContain('/a.pdf')
  })
})
