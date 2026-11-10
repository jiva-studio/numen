import { describe, expect, it } from 'vitest'
import type { TabKindRegistration } from '@numen/plugin-sdk'
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

const createWindow = () => {
  let getKinds = (): readonly TabKindRegistration[] => []
  const tabOpens: (readonly [string, string])[] = []
  const tabs: { kind: string; state: object }[] = []
  const navigateTab = async (kind: string, path: string): Promise<string> => {
    const registration = getKinds().find((one) => one.kind === kind)!
    const last = tabs.filter((one) => one.kind === kind).at(-1)
    if (last) {
      await registration.navigate?.(last.state, path, '')
      return kind
    }
    tabOpens.push([kind, path])
    tabs.push({ kind, state: registration.open!(path, `${kind}-${tabs.length}`) })
    return kind
  }
  const bind = (ctx: { getRegisteredTabKinds(): TabKindRegistration[] }): void => {
    getKinds = () => ctx.getRegisteredTabKinds()
  }
  return { tabOpens, tabs, navigateTab, bind }
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

  it('holds subscriptions for tab kind, vault and commands, and releases all', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.plugin.subscriptions).toHaveLength(5)

    ctx.dispose()
    expect(ctx.plugin.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredCommands()).toHaveLength(0)
  })

  it('registers the handler that opens the plex', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.getRegisteredCommands().map((one) => one.id)).toStrictEqual(['graph.open'])
  })

  const createWindowContext = () => {
    const { host } = createHost({ opening: 'Root.md' })
    const strip = createWindow()
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.plex.graph',
      workspace: host.workspace,
      notes: host.notes,
      relations: host.relations,
      navigateTab: strip.navigateTab,
    })
    strip.bind(ctx)
    activate(ctx)
    return { ctx, strip }
  }

  it('opens one plex however many times the command runs', async () => {
    const { ctx, strip } = createWindowContext()

    await ctx.triggerCommand('graph.open')
    await ctx.triggerCommand('graph.open')
    await ctx.triggerCommand('graph.open', {
      path: 'Other.md',
      title: 'Other.md',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(strip.tabOpens).toStrictEqual([[PLEX, '']])
    expect(strip.tabs).toHaveLength(1)
  })

  it('shows the active note in a plex that is already open', async () => {
    const { ctx, strip } = createWindowContext()
    await ctx.triggerCommand('graph.open')
    await settle()

    await ctx.triggerCommand('graph.open', {
      path: 'Other.md',

      title: 'Other.md',

      type: 'note',

      isFolder: false,

      others: [],

      tab: '',
    })
    await settle()

    const state = strip.tabs[0]!.state as PlexTabState
    expect(state.canvas.view.here.value).toBe('Other.md')
  })

  it('shows the active note in the plex it opens', async () => {
    const { ctx, strip } = createWindowContext()

    await ctx.triggerCommand('graph.open', {
      path: 'Other.md',

      title: 'Other.md',

      type: 'note',

      isFolder: false,

      others: [],

      tab: '',
    })

    expect(strip.tabOpens).toStrictEqual([[PLEX, 'Other.md']])
  })

  it('keeps its note without an active note', async () => {
    const { ctx, strip } = createWindowContext()
    await ctx.triggerCommand('graph.open', {
      path: 'Other.md',
      title: 'Other.md',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })
    await settle()

    await ctx.triggerCommand('graph.open')
    await settle()

    const state = strip.tabs[0]!.state as PlexTabState
    expect(state.canvas.view.here.value).toBe('Other.md')
  })

  it('registers no command that makes a note', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.getRegisteredCommands().map((one) => one.id)).not.toContain('graph.createNote')
  })

  it('asks every plex for its picture again when the vault changes', async () => {
    const { ctx, requested } = createContext()
    activate(ctx)
    const kind = ctx.getRegisteredTabKinds()[0]!
    kind.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    requested.headings.length = 0

    ctx.events.emitVaultChange({ paths: ['One.md'] })
    await settle()
    await settle()

    expect(requested.headings.length).toBeGreaterThan(0)
  })

  it('reads no headings while its nestHeadings setting is off, and reads them once it is on', async () => {
    const { host, requested } = createHost({ opening: 'Root.md' })
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.plex.graph',
      workspace: host.workspace,
      notes: host.notes,
      relations: host.relations,
      initialSettings: { nestHeadings: false },
    })
    activate(ctx)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    expect(requested.headings).toStrictEqual([])

    ctx.setSetting('nestHeadings', true)
    ctx.setSetting('headingLimit', 3)
    await settle()

    expect(requested.headings.length).toBeGreaterThan(0)
    expect(state.headings.maxHeadings.value).toBe(3)
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
