/**
 * What the deck tabs of a window do when one opens, comes on screen, is asked
 * about, and closes.
 */
import { describe, expect, it, vi } from 'vitest'
import { createDeckKind, type DeckTabsInside } from './kind'
import type { DeckTabState } from './types'

const createInside = () => {
  const cardTabPathMap = new Map<string, string>()
  const pendingTabIds = new Map<string, string>()
  const state = { id: 'tab-1' } as unknown as DeckTabState
  const inside = {
    cardTabPathMap,
    pendingTabIds,
    store: { open: vi.fn(), getPath: (id: string) => `decks/${id}.md` },
    vaultAnswers: { getTitle: (path: string) => `title of ${path}` },
    listStencils: vi.fn(async () => {}),
    listPresets: vi.fn(async () => {}),
    listStencilsAgain: vi.fn(),
    listPresetsAgain: vi.fn(),
    refreshDeckPreset: vi.fn(async () => {}),
    createDeckTabState: vi.fn(() => state),
  }
  return { inside, state, kind: createDeckKind(inside as unknown as DeckTabsInside) }
}

describe('a deck tab opened', () => {
  it('opens the note at the path its card tab was filed under and forgets both records', () => {
    const { inside, state, kind } = createInside()
    inside.cardTabPathMap.set('tab-1', 'decks/Animals.md')
    inside.pendingTabIds.set('decks/Animals.md', 'tab-1')

    expect(kind.open('tab-1', 'tab')).toBe(state)

    expect(inside.store.open).toHaveBeenCalledWith('tab-1', 'decks/Animals.md')
    expect(inside.cardTabPathMap.has('tab-1')).toBe(false)
    expect(inside.pendingTabIds.has('decks/Animals.md')).toBe(false)
    expect(inside.listStencils).toHaveBeenCalledTimes(1)
    expect(inside.listPresets).toHaveBeenCalledTimes(1)
    expect(inside.refreshDeckPreset).toHaveBeenCalledWith('decks/Animals.md')
    expect(inside.createDeckTabState).toHaveBeenCalledWith('tab-1')
  })

  it('opens the note at the path it was asked for where no card tab filed one', () => {
    const { inside, kind } = createInside()

    kind.open('decks/Animals.md', 'tab')

    expect(inside.store.open).toHaveBeenCalledWith('decks/Animals.md', 'decks/Animals.md')
  })
})

describe('what a deck tab is known by', () => {
  it('is the path it opens on', () => {
    expect(createInside().kind.identity?.('decks/Animals.md')).toBe('decks/Animals.md')
  })
})

describe('a deck tab coming on screen', () => {
  it('asks again for the stencils and the presets', () => {
    const { inside, state, kind } = createInside()

    kind.onShow?.(state, 'tab-1')

    expect(inside.listStencilsAgain).toHaveBeenCalledTimes(1)
    expect(inside.listPresetsAgain).toHaveBeenCalledTimes(1)
  })
})

describe('what a deck tab tells whoever answers for the person', () => {
  it('is the path and title of the deck as the command target', () => {
    const { state, kind } = createInside()

    expect(kind.getTarget?.(state)).toStrictEqual({
      path: 'decks/tab-1.md',
      title: 'title of decks/tab-1.md',
    })
  })

  it('is the path of the deck as the open tab', () => {
    const { state, kind } = createInside()

    expect(kind.getOpenTab?.(state)).toStrictEqual({ path: 'decks/tab-1.md' })
  })
})

describe('a deck tab closed', () => {
  it('closes the tab and leaves the pane to the caller', () => {
    const { kind } = createInside()
    const close = vi.fn()
    const state = { close } as unknown as DeckTabState

    expect(kind.onClose?.(state, 'tab-1')).toBe(false)
    expect(close).toHaveBeenCalledWith('tab-1')
  })
})
