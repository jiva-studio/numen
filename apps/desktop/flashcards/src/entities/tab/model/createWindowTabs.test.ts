import { describe, expect, it } from 'vitest'
import { onScopeDispose } from 'vue'
import { createWindowTabs } from './createWindowTabs'

const createKind = (kind: string, order: number) => {
  const released: string[] = []
  const destroyed: string[] = []
  const registration = {
    kind,
    component: {},
    placement: { pane: 'main' as const, order },
    open: (vault: string, id: string) => {
      onScopeDispose(() => released.push(id))
      return { vault, id }
    },
    onDestroy: (_: unknown, id: string) => destroyed.push(id),
  }
  return { registration, released, destroyed }
}

describe('createWindowTabs', () => {
  it('puts in front the main-pane kind of the lowest order and falls back when its plugin is dropped', () => {
    const tabs = createWindowTabs()
    const late = tabs.addView({
      pluginId: 'md.numen.a',
      registration: createKind('late', 2).registration,
    })
    tabs.addView({ pluginId: 'md.numen.b', registration: createKind('early', 1).registration })
    tabs.addView({ pluginId: 'md.numen.c', registration: { kind: 'unplaced', component: {} } })

    expect(tabs.front.value?.registration.kind).toBe('early')
    tabs.dropPlugin('md.numen.b')
    expect(tabs.front.value?.registration.kind).toBe('late')
    late.dispose()
    expect(tabs.front.value).toBeNull()
  })

  it('opens the front kind on a vault and replaces its tab when opened again', () => {
    const tabs = createWindowTabs()
    const kind = createKind('review', 0)
    tabs.addView({ pluginId: 'md.numen.a', registration: kind.registration })

    tabs.openTab('physics')
    const first = tabs.tab.value
    tabs.openTab('words')

    expect(first?.vault).toBe('physics')
    expect(tabs.tab.value?.vault).toBe('words')
    expect(kind.destroyed).toEqual([first?.id])
    expect(kind.released).toEqual([first?.id])
  })

  it('closes the tab named, and leaves it when the name is another one', () => {
    const tabs = createWindowTabs()
    tabs.addView({ pluginId: 'md.numen.a', registration: createKind('review', 0).registration })
    tabs.openTab('physics')

    tabs.closeTab('another#9')
    expect(tabs.tab.value).not.toBeNull()
    tabs.closeTab(tabs.tab.value?.id)
    expect(tabs.tab.value).toBeNull()
  })

  it('closes the tab of a plugin that is dropped, and the tab of another stays', () => {
    const tabs = createWindowTabs()
    tabs.addView({ pluginId: 'md.numen.a', registration: createKind('review', 0).registration })
    tabs.openTab('physics')

    tabs.dropPlugin('md.numen.other')
    expect(tabs.tab.value).not.toBeNull()
    tabs.dropPlugin('md.numen.a')
    expect(tabs.tab.value).toBeNull()
  })

  it('opens nothing and keeps the scope clean when the kind fails to open', () => {
    const tabs = createWindowTabs()
    tabs.addView({
      pluginId: 'md.numen.a',
      registration: {
        kind: 'broken',
        component: {},
        placement: { pane: 'main', order: 0 },
        open: () => {
          throw new Error('no deck')
        },
      },
    })

    expect(() => tabs.openTab('physics')).toThrowError('no deck')
    expect(tabs.tab.value).toBeNull()
  })
})
