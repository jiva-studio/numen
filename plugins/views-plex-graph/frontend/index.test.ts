import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, PlexGraphView, PLEX } from './index'
import { createHost } from './fixtures/host'
import { settle } from './fixtures'
import type { PlexTabState } from './types'

const createContext = () => {
  const { host, requested } = createHost({ opening: 'Root.md' })
  const ctx = createMockPluginContext({
    pluginId: 'md.numen.views.plex.graph',
    workspace: host.workspace,
    notes: host.notes,
    relations: host.relations,
  })
  return { ctx, requested }
}

describe('view-plex-graph plugin', () => {
  it('registers the plex tab kind upon activation', () => {
    const { ctx } = createContext()
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe(PLEX)
    expect(tabKinds[0]?.component).toBe(PlexGraphView)
    expect(tabKinds[0]?.open).toBeTypeOf('function')
  })

  it('holds subscriptions for tab kind, vault, commands and context menus, and releases all', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.subscriptions).toHaveLength(10)
    const menus = ctx.getRegisteredContextMenus()
    expect(menus).toHaveLength(4)
    expect(menus.map((m) => m.id)).toEqual(['travel', 'child', 'parent', 'jump'])

    ctx.dispose()
    expect(ctx.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredContextMenus()).toHaveLength(0)
    expect(ctx.getRegisteredCommands()).toHaveLength(0)
  })

  it('declares a new plex over the window and the plex as where a note is put in front', () => {
    const { ctx } = createContext()
    activate(ctx)

    const commands = ctx.getRegisteredCommands()
    expect(commands.map((one) => [one.id, one.scope, one.shortcut])).toStrictEqual([
      ['plex', undefined, 'mod+shift+x'],
      ['travel', 'note', 'mod+shift+p'],
    ])
    expect(commands.map((one) => one.isNoteDestination === true)).toStrictEqual([false, true])
  })

  it('opens a plex, and moves the plex last in front to a note', async () => {
    const openCalls: (readonly [string, string | undefined])[] = []
    const navigateCalls: (readonly [string, string])[] = []
    const { host } = createHost({ opening: 'Root.md' })
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.plex.graph',
      workspace: host.workspace,
      notes: host.notes,
      relations: host.relations,
      openTab: (kind, path) => {
        openCalls.push([kind, path])
        return Promise.resolve('')
      },
      navigateTab: (kind, path) => {
        navigateCalls.push([kind, path])
        return Promise.resolve('')
      },
    })
    activate(ctx)

    await ctx.triggerCommand('plex')
    await ctx.triggerCommand('travel', { path: 'One.md', title: 'One', type: 'note' })
    await ctx.getRegisteredContextMenus()[0]!.handler({
      path: 'Two.md',
      name: 'Two',
      isFolder: false,
      isNote: true,
    })

    expect(openCalls).toStrictEqual([[PLEX, undefined]])
    expect(navigateCalls).toStrictEqual([
      [PLEX, 'One.md'],
      [PLEX, 'Two.md'],
    ])
  })

  it('asks every plex for its picture again when the vault changes', async () => {
    const { ctx, requested } = createContext()
    activate(ctx)
    const kind = ctx.getRegisteredTabKinds()[0]!
    kind.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    requested.headings.length = 0

    ctx.emitVaultChange({ paths: ['One.md'] })
    await settle()
    await settle()

    expect(requested.headings.length).toBeGreaterThan(0)
  })

  it('reads no headings while its nest_headings setting is off, and reads them once it is on', async () => {
    const { host, requested } = createHost({ opening: 'Root.md' })
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.plex.graph',
      workspace: host.workspace,
      notes: host.notes,
      relations: host.relations,
      initialSettings: { nest_headings: false },
    })
    activate(ctx)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    expect(requested.headings).toStrictEqual([])

    ctx.setSetting('nest_headings', true)
    ctx.setSetting('heading_limit', 3)
    await settle()

    expect(requested.headings.length).toBeGreaterThan(0)
    expect(state.maxHeadings.value).toBe(3)
  })

  it('refuses to activate in a window that offers no workspace, notes or relationships', () => {
    expect(() =>
      activate(createMockPluginContext({ pluginId: 'md.numen.views.plex.graph' })),
    ).toThrow(
      'a plex needs the window it is drawn in, the notes of the vault and their relationships',
    )
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx } = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe(PLEX)
  })
})
