import { describe, expect, it, vi } from 'vitest'
import {
  createMockBufferHost,
  createMockPluginContext,
  createMockTabWindow,
} from '@numen/plugin-sdk/testing'
import type { WorkspaceHost } from '@numen/plugin-sdk'
import plugin, { activate, DECK, PRESET, REVIEW, STENCIL } from './index'
import { cards, presets } from './cards'

const createContext = () => {
  const writers: string[] = []
  const workspace = {
    getMessageWriter: (name: string) => {
      writers.push(name)
      return () => {}
    },
    getToday: () => '2026-09-04',
  } as unknown as WorkspaceHost
  const buffers = createMockBufferHost()
  const window = createMockTabWindow()
  const closeTab = vi.fn(window.finishClose)
  const ctx = createMockPluginContext({
    pluginId: 'md.numen.views.flashcards.studio',
    workspace,
    buffers,
    closeTab,
  })
  return { ctx, writers, buffers, window, closeTab }
}

describe('flashcards-studio plugin', () => {
  it('draws only the review in the window of the flashcards target', () => {
    const ctx = createMockPluginContext({ target: 'desktop.flashcards' })
    activate(ctx)

    const kinds = ctx.getRegisteredTabKinds()
    expect(kinds.map((one) => one.kind)).toStrictEqual([REVIEW])
    expect(kinds[0]?.placement).toStrictEqual({ pane: 'main', order: 0 })
  })

  it('registers the deck, stencil and preset tab kinds upon activation', () => {
    const { ctx } = createContext()
    activate(ctx)

    const kinds = ctx.getRegisteredTabKinds()
    expect(kinds.map((one) => one.kind)).toStrictEqual([DECK, STENCIL, PRESET])
    expect(kinds.every((one) => typeof one.open === 'function')).toBe(true)
    expect(kinds.every((one) => one.component !== undefined && one.icon !== undefined)).toBe(true)
  })

  it('names the kinds the window files its tabs under', () => {
    expect([DECK, STENCIL, PRESET]).toStrictEqual(['deck', 'stencil', 'preset'])
  })

  it('removes the three kinds when the plugin is disabled', () => {
    const { ctx } = createContext()
    activate(ctx)

    ctx.dispose()

    expect(ctx.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('writes the messages of the stencils and the presets under a name of their own', () => {
    const { ctx, writers } = createContext()
    activate(ctx)

    expect(writers).toStrictEqual(['stencil', 'preset'])
  })

  it('makes each kind able to make the file it edits', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.getRegisteredTabKinds().every((one) => typeof one.createFile === 'function')).toBe(
      true,
    )
  })

  it('queries kind handlers for pending changes before flush', () => {
    const { ctx } = createContext()
    activate(ctx)

    const [deck, stencil, preset] = ctx.getRegisteredTabKinds()
    expect(typeof deck?.isAsking).toBe('function')
    expect(typeof stencil?.isAsking).toBe('function')
    expect(preset?.isAsking).toBeUndefined()
    expect(ctx.getRegisteredTabKinds().every((one) => typeof one.flushPending === 'function')).toBe(
      true,
    )
  })

  it('answers where a command over a file leads from the preset kind', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(typeof ctx.getRegisteredTabKinds()[2]?.resolveCommand).toBe('function')
  })

  it('passes the closing of a tab that took its own close to the window', async () => {
    const { ctx, window, closeTab } = createContext()
    activate(ctx)
    const kinds = ctx.getRegisteredTabKinds()
    for (const one of kinds) window.register(one as never)
    const id = await window.openTab(PRESET, 'Slow.md')

    window.requestClose(id)
    await vi.waitFor(() => expect(closeTab).toHaveBeenCalledWith(id))
    expect(window.each(PRESET)).toHaveLength(0)
  })

  it('refuses to activate in a window that offers no workspace or open files', () => {
    expect(() =>
      activate(createMockPluginContext({ pluginId: 'md.numen.views.flashcards.studio' })),
    ).toThrow('the flashcards need the window their tabs are drawn in and its open files')
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx } = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(3)
  })

  it('registers context menu items when files service is available', async () => {
    const createDeck = vi
      .spyOn(cards, 'createDeck')
      .mockResolvedValue({ ok: true, value: { path: 'Decks/New deck.md' } } as never)
    const createStencil = vi
      .spyOn(cards, 'createStencil')
      .mockResolvedValue({ ok: true, value: { path: 'Stencils/New stencil.md' } } as never)
    const createPreset = vi
      .spyOn(presets, 'createPreset')
      .mockResolvedValue({ ok: true, value: { path: 'Presets/New preset.md' } } as never)
    const { ctx } = createContext()
    ;(ctx as unknown as { files: unknown }).files = {}

    activate(ctx)

    const menus = ctx.getRegisteredContextMenus()
    expect(menus.map((m) => m.id)).toEqual(['newDeck', 'newStencil', 'newPreset'])
    expect(menus.every((m) => m.parentId === 'new')).toBe(true)

    await menus[0]?.handler({ path: 'Decks', isFolder: true, kind: 'other', isSeveral: false })
    expect(createDeck).toHaveBeenCalledWith('New deck', 'Decks')

    await menus[1]?.handler({
      path: 'Decks/One.md',
      isFolder: false,
      kind: 'note',
      isSeveral: false,
    })
    expect(createStencil).toHaveBeenCalledWith('New stencil', 'Decks', [])

    await menus[2]?.handler({ path: '', isFolder: false, kind: 'other', isSeveral: false })
    expect(createPreset).toHaveBeenCalledWith('New preset', '')
  })
})
