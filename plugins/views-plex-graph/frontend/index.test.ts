import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import type { JsonValue, TabKindRegistration } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, PlexGraphView, PLEX } from './index'
import { createHost } from './fixtures/host'
import { settle } from './fixtures'
import type { NoteGraph, PlexTabState } from './types'
import { Plex } from './plex'

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

    // The tab kind, the vault, the command, and one for each of the seventeen settings.
    expect(ctx.plugin.subscriptions).toHaveLength(20)

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

describe('the look of an open plex', () => {
  const openPlex = async (initialSettings: Record<string, JsonValue> = {}) => {
    const { host } = createHost({ opening: 'Root.md' })
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.plex.graph',
      workspace: host.workspace,
      notes: host.notes,
      relations: host.relations,
      initialSettings,
    })
    activate(ctx)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    const tab = mount(PlexGraphView, { props: { state }, attachTo: document.body })
    await settle()
    await nextTick()
    return { ctx, state, tab, getPlex: () => tab.findComponent(Plex) }
  }

  afterEach(() => {
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

  it('keeps the heading settings next to the look settings', async () => {
    const { ctx, state } = await openPlex({ showPills: false })

    ctx.setSetting('headingLimit', 3)
    await nextTick()

    expect(state.headings.maxHeadings.value).toBe(3)
  })
})

describe('the names of the notes related to the related notes of an open plex', () => {
  const answerFor = (path: string, depth: number | undefined): NoteGraph => {
    const items =
      depth === 2 && path === 'One.md'
        ? [
            { path: 'Aunt.md', title: 'Aunt', relation: 'parent' as const, relatedTo: 'Kid.md' },
            { path: 'Leaf1.md', title: 'Leaf1', relation: 'child' as const, relatedTo: 'Kid.md' },
            { path: 'Leaf2.md', title: 'Leaf2', relation: 'child' as const, relatedTo: 'Kid.md' },
            { path: 'Leaf3.md', title: 'Leaf3', relation: 'child' as const, relatedTo: 'Kid.md' },
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
    const asked: [string, number | undefined][] = []
    const notes = {
      ...host.notes,
      listRelatedNotes: async (path: string, depth?: 1 | 2) => {
        asked.push([path, depth])
        return answerFor(path, depth)
      },
    }
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.plex.graph',
      workspace: host.workspace,
      notes,
      relations: host.relations,
      initialSettings,
    })
    activate(ctx)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    const tab = mount(PlexGraphView, { props: { state }, attachTo: document.body })
    await settle()
    await nextTick()
    const getPlex = () => tab.findComponent(Plex)
    const getNames = () => tab.findAll('.plex__name-text').map((name) => name.text())
    return { ctx, state, tab, asked, getPlex, getNames }
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

  it('asks for the notes related to related notes where nothing is set, and draws no node for them', async () => {
    const { asked, state, getPlex, getNames } = await openPlex()

    expect(asked.at(-1)).toEqual(['One.md', 2])
    expect(getPlex().props('showNames')).toBe(true)
    expect(getNames()).toEqual(['Aunt', 'Leaf1', 'Leaf2', 'Leaf3'])
    expect(state.canvas.picture.value!.nodes.map((node) => node.title)).toEqual(['One', 'Kid'])
  })

  it('asks again for depth 1 and loses the names when the setting is turned off, and gets them back when it is turned on', async () => {
    const { ctx, asked, getPlex, getNames } = await openPlex()

    ctx.setSetting('showNames', false)
    await settle()
    expect(asked.at(-1)).toEqual(['One.md', 1])
    expect(getPlex().props('showNames')).toBe(false)
    expect(getNames()).toEqual([])

    ctx.setSetting('showNames', true)
    await settle()
    expect(asked.at(-1)).toEqual(['One.md', 2])
    expect(getNames()).toEqual(['Aunt', 'Leaf1', 'Leaf2', 'Leaf3'])
  })

  it('asks for depth 1 from the start when the setting is off', async () => {
    const { asked, getNames } = await openPlex({ showNames: false })

    expect(asked.every(([, depth]) => depth === 1)).toBe(true)
    expect(getNames()).toEqual([])
  })

  it('cuts the names to Names per note without asking again', async () => {
    const { ctx, asked, getNames } = await openPlex()
    const before = asked.length

    ctx.setSetting('namesPerNote', 2)
    await settle()

    expect(asked).toHaveLength(before)
    expect(getNames()).toEqual(['Aunt', 'Leaf1'])
  })

  it('moves the focus to the note of a name that is clicked', async () => {
    const { tab, state } = await openPlex()
    const name = tab.get('[role="button"][aria-label^="Leaf1, name under "]')

    await name.trigger('pointerenter')
    await name.trigger('click')
    await settle()

    expect(state.canvas.view.here.value).toBe('Leaf1.md')
  })

  it('moves nothing when a name that sits back is clicked', async () => {
    const { tab, state } = await openPlex()

    tab
      .get('[role="button"][aria-label^="Leaf1, name under "]')
      .element.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 1 }))
    await settle()

    expect(state.canvas.view.here.value).toBe('One.md')
  })
})
