import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, DeckEditorTab, StencilEditorTab, PresetEditorTab } from './index'

describe('flashcards-studio plugin', () => {
  it('registers deck, stencil, and preset tab kinds upon activation', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.flashcards-studio' })
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(3)

    const deckKind = tabKinds.find((k) => k.kind === 'deck')
    expect(deckKind).toBeDefined()
    expect(deckKind?.component).toBe(DeckEditorTab)
    expect(deckKind?.title).toBe('Deck Editor')

    const stencilKind = tabKinds.find((k) => k.kind === 'stencil')
    expect(stencilKind).toBeDefined()
    expect(stencilKind?.component).toBe(StencilEditorTab)
    expect(stencilKind?.title).toBe('Card Stencil Editor')

    const presetKind = tabKinds.find((k) => k.kind === 'preset')
    expect(presetKind).toBeDefined()
    expect(presetKind?.component).toBe(PresetEditorTab)
    expect(presetKind?.title).toBe('Review Presets')
  })

  it('activates cleanly using default export lifecycle', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.flashcards-studio' })
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(3)
  })

  it('unregisters subscriptions cleanly when disposed', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.flashcards-studio' })
    activate(ctx)

    expect(ctx.subscriptions.length).toBe(3)
    ctx.dispose()
    expect(ctx.subscriptions).toHaveLength(0)
  })
})
