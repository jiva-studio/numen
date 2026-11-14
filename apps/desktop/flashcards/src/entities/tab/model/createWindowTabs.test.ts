import { describe, expect, it } from 'vitest'
import { onScopeDispose } from 'vue'
import type { AnyTabKind } from '@numen/plugin-loader'
import { createWindowTabs } from './createWindowTabs'

const createKind = (kind: string, pluginId: string, order?: number) => {
  const released: string[] = []
  const destroyed: string[] = []
  const tabKind: AnyTabKind = {
    kind,
    pluginId,
    pane: {},
    open: (vault, id) => {
      onScopeDispose(() => released.push(id))
      return { vault, id }
    },
    ...(order === undefined ? {} : { display: { placement: { pane: 'main' as const, order } } }),
    events: { onDestroy: (_, id) => destroyed.push(id) },
  }
  return { tabKind, released, destroyed }
}

describe('createWindowTabs', () => {
  it('puts in front the main-pane kind of the lowest order and falls back when it is taken away', () => {
    const tabs = createWindowTabs()
    const dropLate = tabs.registerKind(createKind('late', 'md.numen.a', 2).tabKind)
    const dropEarly = tabs.registerKind(createKind('early', 'md.numen.b', 1).tabKind)
    tabs.registerKind(createKind('unplaced', 'md.numen.c').tabKind)

    expect(tabs.front.value?.kind).toBe('early')
    expect(tabs.frontPlugins.value).toStrictEqual([{ pluginId: 'md.numen.b' }])
    dropEarly()
    expect(tabs.front.value?.kind).toBe('late')
    dropLate()
    expect(tabs.front.value).toBeNull()
    expect(tabs.frontPlugins.value).toStrictEqual([])
  })

  it('opens the front kind on a vault and replaces its tab when opened again', () => {
    const tabs = createWindowTabs()
    const kind = createKind('review', 'md.numen.a', 0)
    tabs.registerKind(kind.tabKind)

    tabs.openTab('physics')
    const first = tabs.tab.value
    tabs.openTab('words')

    expect(first?.vault).toBe('physics')
    expect(first?.state).toStrictEqual({ vault: 'physics', id: 'review#1' })
    expect(tabs.tab.value?.vault).toBe('words')
    expect(kind.destroyed).toEqual([first?.id])
    expect(kind.released).toEqual([first?.id])
  })

  it('opens nothing while no kind is in front', () => {
    const tabs = createWindowTabs()

    tabs.openTab('physics')

    expect(tabs.tab.value).toBeNull()
  })

  it('closes the tab named, and leaves it when the name is another one', () => {
    const tabs = createWindowTabs()
    tabs.registerKind(createKind('review', 'md.numen.a', 0).tabKind)
    tabs.openTab('physics')

    tabs.closeTab('another#9')
    expect(tabs.tab.value).not.toBeNull()
    tabs.closeTab(tabs.tab.value?.id)
    expect(tabs.tab.value).toBeNull()
  })

  it('closes the tab of a plugin that stops, and the tab of another stays', () => {
    const tabs = createWindowTabs()
    tabs.registerKind(createKind('review', 'md.numen.a', 0).tabKind)
    tabs.openTab('physics')

    tabs.closePluginTab('md.numen.other')
    expect(tabs.tab.value).not.toBeNull()
    tabs.closePluginTab('md.numen.a')
    expect(tabs.tab.value).toBeNull()
  })

  it('opens nothing and keeps the scope clean when the kind fails to open', () => {
    const tabs = createWindowTabs()
    const released: string[] = []
    tabs.registerKind({
      kind: 'broken',
      pluginId: 'md.numen.a',
      pane: {},
      display: { placement: { pane: 'main', order: 0 } },
      open: () => {
        onScopeDispose(() => released.push('scope'))
        throw new Error('no deck')
      },
    })

    expect(() => tabs.openTab('physics')).toThrowError('no deck')
    expect(tabs.tab.value).toBeNull()
    expect(released).toStrictEqual(['scope'])
  })
})
