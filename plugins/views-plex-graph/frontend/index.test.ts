import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import type { JsonValue, TabKindRegistration } from '@numen/plugin-sdk'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, PlexGraphView, PLEX } from './index'
import { createHost } from './fixtures/host'
import { settle } from './fixtures'
import type { NoteGraph, PlexTabState } from './types'
import { Plex } from './plex'

const createContext = () => {
  const { host, requested } = createHost({ opening: 'Root.md' })
  const ctx = createMockPluginContext({ pluginId: 'md.numen.views.plex.graph' })
  const services = createMockHostServices({
    workspace: host.workspace,
    notes: host.notes,
    relations: host.relations,
  })
  return { ctx, services, requested }
}

const createWindow = () => {
  let getKinds = (): readonly TabKindRegistration[] => []
  const tabOpens: (readonly [string, string])[] = []
  const tabs: { kind: string; state: object }[] = []
  const openTab = async (kind: string, path: string): Promise<string> => {
    const registration = getKinds().find((one) => one.kind === kind)!
    tabOpens.push([kind, path])
    tabs.push({ kind, state: registration.open!(path, `${kind}-${tabs.length}`) })
    return kind
  }
  const bind = (ctx: { getRegisteredTabKinds(): TabKindRegistration[] }): void => {
    getKinds = () => ctx.getRegisteredTabKinds()
  }
  return { tabOpens, tabs, openTab, bind }
}

describe('view-plex-graph plugin', () => {
  it('registers the plex tab kind upon activation', () => {
    const { ctx, services } = createContext()
    activate(ctx, services)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe(PLEX)
    expect(tabKinds[0]?.component).toBe(PlexGraphView)
    expect(tabKinds[0]?.open).toBeTypeOf('function')
  })

  it('holds subscriptions for tab kind and commands, and releases all', () => {
    const { ctx, services } = createContext()
    activate(ctx, services)

    // The tab kind, the command, and one for each of the fourteen settings.
    expect(ctx.identity.subscriptions).toHaveLength(16)

    ctx.dispose()
    expect(ctx.identity.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredCommands()).toHaveLength(0)
  })

  it('registers the handler that opens the plex', () => {
    const { ctx, services } = createContext()
    activate(ctx, services)

    expect(ctx.getRegisteredCommands().map((one) => one.id)).toStrictEqual(['graph.open'])
  })

  const createWindowContext = () => {
    const { host } = createHost({ opening: 'Root.md' })
    const strip = createWindow()
    const ctx = createMockPluginContext({ pluginId: 'md.numen.views.plex.graph' })
    const services = createMockHostServices({
      workspace: host.workspace,
      notes: host.notes,
      relations: host.relations,
      openTab: strip.openTab,
    })
    strip.bind(ctx)
    activate(ctx, services)
    return { ctx, strip }
  }

  it('opens another plex each time the command runs', async () => {
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

    expect(strip.tabOpens).toStrictEqual([
      [PLEX, ''],
      [PLEX, ''],
      [PLEX, 'Other.md'],
    ])
    expect(strip.tabs).toHaveLength(3)
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

  it('registers no command that makes a note', () => {
    const { ctx, services } = createContext()
    activate(ctx, services)

    expect(ctx.getRegisteredCommands().map((one) => one.id)).not.toContain('graph.createNote')
  })

  it('asks every plex for its picture again when the vault changes', async () => {
    const { ctx, services, requested } = createContext()
    activate(ctx, services)
    const kind = ctx.getRegisteredTabKinds()[0]!
    kind.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    requested.related.length = 0

    services.events.emitVaultChange({ paths: ['One.md'] })
    await settle()
    await settle()

    expect(requested.related.length).toBeGreaterThan(0)
  })

  it('refuses to activate in a window that offers no workspace, notes or relationships', () => {
    expect(() =>
      activate(
        createMockPluginContext({ pluginId: 'md.numen.views.plex.graph' }),
        createMockHostServices(),
      ),
    ).toThrow(
      'a plex needs the window it is drawn in, the notes of the vault and their relationships',
    )
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx, services } = createContext()
    await plugin.activate(ctx, services)

    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe(PLEX)
  })
})

describe('the look of an open plex', () => {
  const openPlex = async (initialSettings: Record<string, JsonValue> = {}) => {
    const { host } = createHost({ opening: 'Root.md' })
    const ctx = createMockPluginContext({ pluginId: 'md.numen.views.plex.graph', initialSettings })
    const services = createMockHostServices({
      workspace: host.workspace,
      notes: host.notes,
      relations: host.relations,
    })
    activate(ctx, services)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    const tab = mount(PlexGraphView, { props: { state }, attachTo: document.body })
    await settle()
    await nextTick()
    return { ctx, state, tab, getPlex: () => tab.findComponent(Plex) }
  }

  beforeEach(() => {
    // A window that reports its size, which the plex waits for before it draws.
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(private readonly took: ResizeObserverCallback) {}
        observe(target: Element) {
          if (!target.classList.contains('plex-frame')) return
          const contentRect = { width: 1200, height: 800 } as DOMRectReadOnly
          this.took([{ contentRect } as ResizeObserverEntry], this as unknown as ResizeObserver)
        }
        disconnect = () => undefined
      },
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.replaceChildren()
  })

  it('draws the plex with the values the settings hold when the tab opens', async () => {
    const { getPlex } = await openPlex({
      showPills: false,
      highlightLinks: false,
      animationLength: 900,
      lineThickness: 3,
      outlineThickness: 1.5,
      lineCurvature: 0.2,
      focusSize: 1.6,
      pillPaddingSides: 4,
      pillPaddingEnds: 3.5,
      noteRoom: 12,
    })

    expect(getPlex().props('showPills')).toBe(false)
    expect(getPlex().props('highlightLinks')).toBe(false)
    expect(getPlex().props('duration')).toBe(900)
    expect(getPlex().props('lineWidth')).toBe(3)
    expect(getPlex().props('outlineWidth')).toBe(1.5)
    expect(getPlex().props('options')).toMatchObject({
      focusScale: 1.6,
      padding: { inline: 4, block: 3.5 },
      room: 12,
      routing: { curvature: 0.2 },
    })
  })

  it('gives the open plex the default look where nothing is set', async () => {
    const { getPlex } = await openPlex()

    expect(getPlex().props('showPills')).toBe(true)
    expect(getPlex().props('highlightLinks')).toBe(true)
    expect(getPlex().props('duration')).toBe(420)
    expect(getPlex().props('lineWidth')).toBe(2)
    expect(getPlex().props('outlineWidth')).toBe(1)
    expect(getPlex().props('options')).toMatchObject({
      focusScale: 1.22,
      padding: { inline: 10, block: 9.25 },
      room: 0,
      routing: { curvature: 0.55 },
    })
  })

  it('redraws the open plex without its pills when Pill background is turned off, and with them when it is turned on', async () => {
    const { ctx, tab } = await openPlex()
    expect(tab.findAll('.plex__box').length).toBeGreaterThan(0)

    ctx.setSetting('showPills', false)
    await nextTick()
    expect(tab.findAll('.plex__box')).toHaveLength(0)
    expect(tab.findAll('.plex__dot')).toHaveLength(0)

    ctx.setSetting('showPills', true)
    await nextTick()
    expect(tab.findAll('.plex__box').length).toBeGreaterThan(0)
  })

  it('passes every changed setting to the plex that is already open', async () => {
    const { ctx, getPlex } = await openPlex()

    ctx.setSetting('animationLength', 1000)
    ctx.setSetting('lineThickness', 3)
    ctx.setSetting('outlineThickness', 1)
    ctx.setSetting('lineCurvature', 0)
    ctx.setSetting('focusSize', 2)
    ctx.setSetting('pillPaddingSides', 20)
    ctx.setSetting('pillPaddingEnds', 2)
    ctx.setSetting('noteRoom', 20)
    await nextTick()

    expect(getPlex().props('duration')).toBe(1000)
    expect(getPlex().props('lineWidth')).toBe(3)
    expect(getPlex().props('outlineWidth')).toBe(1)
    expect(getPlex().props('options')).toMatchObject({
      focusScale: 2,
      padding: { inline: 20, block: 2 },
      room: 20,
      routing: { curvature: 0 },
    })
  })

  it('passes the line thickness to the lines and the outline thickness to the pills', async () => {
    const { ctx, getPlex } = await openPlex({ lineThickness: 3, outlineThickness: 1 })
    expect([getPlex().props('lineWidth'), getPlex().props('outlineWidth')]).toStrictEqual([3, 1])

    ctx.setSetting('lineThickness', 1)
    ctx.setSetting('outlineThickness', 3)
    await nextTick()

    expect([getPlex().props('lineWidth'), getPlex().props('outlineWidth')]).toStrictEqual([1, 3])
  })

  it('moves at once once Animation is set to 0', async () => {
    const { ctx, getPlex } = await openPlex({ animationLength: 1000 })

    ctx.setSetting('animationLength', 0)
    await nextTick()

    expect(getPlex().props('duration')).toBe(0)
  })
})

describe('an open plex with notes related to its related notes', () => {
  const answerFor = (path: string, depth: number | undefined): NoteGraph => {
    const items =
      depth === 2 && path === 'One.md'
        ? [
            { path: 'Aunt.md', title: 'Aunt', relation: 'parent' as const, relatedTo: 'Kid.md' },
            { path: 'Leaf1.md', title: 'Leaf1', relation: 'child' as const, relatedTo: 'Kid.md' },
          ]
        : []
    return {
      focus: { path, title: path.replace(/\.md$/, '') },
      focusType: 'note',
      related:
        path === 'One.md'
          ? [
              { path: 'Kid.md', title: 'Kid', relation: 'child', type: 'note' },
              ...items.map((one) => ({ ...one, type: 'note' })),
            ]
          : [],
      connections:
        path === 'One.md'
          ? [
              {
                first: 'One.md',
                second: 'Kid.md',
                role: 'child',
                type: '',
                firstSide: null,
                secondSide: null,
              },
            ]
          : [],
    }
  }

  const openPlex = async (initialSettings: Record<string, JsonValue> = {}) => {
    const { host } = createHost({ opening: 'One.md' })
    const requests: [string, number | undefined][] = []
    const notes = {
      ...host.notes,
      listRelatedNotes: async (paths: readonly string[], depth?: 1 | 2) => {
        const path = paths[0] ?? ''
        requests.push([path, depth])
        return [answerFor(path, depth)]
      },
    }
    const ctx = createMockPluginContext({ pluginId: 'md.numen.views.plex.graph', initialSettings })
    const services = createMockHostServices({
      workspace: host.workspace,
      notes,
      relations: host.relations,
    })
    activate(ctx, services)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    const tab = mount(PlexGraphView, { props: { state }, attachTo: document.body })
    await settle()
    await nextTick()
    return { ctx, state, tab, requests, getPlex: () => tab.findComponent(Plex) }
  }

  beforeEach(() => {
    vi.stubGlobal('CanvasRenderingContext2D', class {})
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
      font: '',
      measureText: (text: string) => ({ width: text.length * 7 }) as TextMetrics,
    } as unknown as CanvasRenderingContext2D)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    document.body.replaceChildren()
  })

  it('asks for the notes related to the focus only, whatever the old names settings hold', async () => {
    const { requests } = await openPlex({ showNames: true, namesPerNote: 6 })

    expect(requests.length).toBeGreaterThan(0)
    expect(requests.every(([, depth]) => depth === 1)).toBe(true)
  })

  it('draws no name of its own, whatever the old names settings hold', async () => {
    const { tab } = await openPlex({ showNames: true })

    expect(tab.findAll('.plex__name')).toHaveLength(0)
    expect(tab.findAll('[aria-label*=", name under "]')).toHaveLength(0)
  })

  it('draws a node for the notes related to the focus and none for the ones beyond them', async () => {
    const { state } = await openPlex()

    expect(state.canvas.picture.value!.nodes.map((node) => node.title)).toEqual(['One', 'Kid'])
  })

  it('gives the drawing no names, no room for names and no opacity of names', async () => {
    const { getPlex } = await openPlex()

    expect(Object.keys(getPlex().props()).filter((key) => /names|roomOf/i.test(key))).toStrictEqual(
      [],
    )
  })

  it('holds no state of names in the tab', async () => {
    const { state } = await openPlex()

    expect(state.canvas).not.toHaveProperty('names')
  })
})

describe('a plex of a vault with no notes', () => {
  const mountEmptyPlex = async () => {
    const { host } = createHost({ opening: '' })
    const ctx = createMockPluginContext({ pluginId: 'md.numen.views.plex.graph' })
    activate(
      ctx,
      createMockHostServices({
        workspace: host.workspace,
        notes: host.notes,
        relations: host.relations,
      }),
    )
    const state = ctx.getRegisteredTabKinds()[0]!.open!('', 'plex:1') as PlexTabState
    const tab = mount(PlexGraphView, { props: { state } })
    await nextTick()
    return tab
  }

  it('says there are no notes and offers to create one', async () => {
    const tab = await mountEmptyPlex()

    expect(tab.text()).toContain('No notes in this vault')
    expect(tab.get('button').text()).toBe('Create note')
  })

  it('draws icons on the empty state and its button', async () => {
    const tab = await mountEmptyPlex()

    expect(tab.find('.plex__empty-icon').exists()).toBe(true)
    expect(tab.find('button svg').exists()).toBe(true)
  })

  it('does not show a loading indicator', async () => {
    const tab = await mountEmptyPlex()

    expect(tab.text()).not.toContain('Loading')
  })
})
