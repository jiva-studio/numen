/**
 * What the deck tabs of a window do when one opens, comes on screen, is asked
 * about, and closes.
 */
import { describe, expect, it, vi } from 'vitest'
import { asValue } from '@numen/plugin-sdk'
import { createDeckKind, type DeckTabsInside } from './kind'
import type { DeckTabState } from './types'

const createInside = (open: readonly string[] = []) => {
  const state = {
    id: 'deck:Animals.md',
    note: { value: { state: 'unsaved' } },
  } as unknown as DeckTabState
  const inside = {
    store: {
      open: vi.fn(),
      getPath: (id: string) => `decks/${id}.md`,
      getOpenIds: () => open,
      stale: vi.fn(() => null as unknown),
      flushPending: vi.fn(async () => {}),
    },
    vaultAnswers: { getTitle: (path: string) => `title of ${path}` },
    createDeck: vi.fn(async (title: string) => asValue({ path: `${title}.md` })),
    listCardModels: vi.fn(async () => {}),
    listPresets: vi.fn(async () => {}),
    listCardModelsAgain: vi.fn(),
    listPresetsAgain: vi.fn(),
    refreshDeckPreset: vi.fn(async () => {}),
    createDeckTabState: vi.fn(() => state),
  }
  return { inside, state, kind: createDeckKind(inside as unknown as DeckTabsInside) }
}

describe('a deck tab opened', () => {
  it('opens the note at the path it was requested for, under the identity of the tab', () => {
    const { inside, state, kind } = createInside()

    expect(kind.open?.('decks/Animals.md', 'deck:decks/Animals.md')).toBe(state)

    expect(inside.store.open).toHaveBeenCalledWith('deck:decks/Animals.md', 'decks/Animals.md')
    expect(inside.listCardModels).toHaveBeenCalledTimes(1)
    expect(inside.listPresets).toHaveBeenCalledTimes(1)
    expect(inside.refreshDeckPreset).toHaveBeenCalledWith('decks/Animals.md')
    expect(inside.createDeckTabState).toHaveBeenCalledWith('deck:decks/Animals.md')
  })
})

describe('what a deck tab is known by', () => {
  it('is the path it opens on', () => {
    expect(createInside().kind.identity?.('decks/Animals.md')).toBe('decks/Animals.md')
  })

  it('is the one the open deck was filed under where its file now remains at the path', () => {
    const { inside, kind } = createInside(['deck:Old.md'])
    inside.store.getPath = () => 'New.md'

    expect(kind.identity?.('New.md')).toBe('Old.md')
  })
})

describe('a deck tab coming on screen', () => {
  it('asks again for the card models and the presets', () => {
    const { inside, state, kind } = createInside()

    kind.onShow?.(state, 'tab-1')

    expect(inside.listCardModelsAgain).toHaveBeenCalledTimes(1)
    expect(inside.listPresetsAgain).toHaveBeenCalledTimes(1)
  })
})

describe('what a deck tab notifies whoever answers for the person', () => {
  it('is the path and title of the deck as the command target', () => {
    const { state, kind } = createInside()

    expect(kind.getTarget?.(state)).toStrictEqual({
      path: 'decks/deck:Animals.md.md',
      title: 'title of decks/deck:Animals.md.md',
    })
  })

  it('is the path of the deck as the open tab', () => {
    const { state, kind } = createInside()

    expect(kind.getOpenTab?.(state)).toStrictEqual({ path: 'decks/deck:Animals.md.md' })
  })

  it('is the mark of its note', () => {
    const { state, kind } = createInside()

    expect(kind.getMark?.(state)).toBe('unsaved')
  })
})

describe('what a deck tab owes the file', () => {
  it('is an answer while the store holds a question for the person', () => {
    const { inside, state, kind } = createInside()
    expect(kind.isAsking?.(state)).toBe(false)

    inside.store.stale.mockReturnValue({ says: 'moved' })

    expect(kind.isAsking?.(state)).toBe(true)
  })

  it('is written out when requested to be', async () => {
    const { inside, state, kind } = createInside()

    await kind.flushPending?.(state)

    expect(inside.store.flushPending).toHaveBeenCalledWith('deck:Animals.md')
  })
})

describe('a deck made', () => {
  it('is requested of the vault under the name and the folder it was given', async () => {
    const { inside, kind } = createInside()

    expect(await kind.createFile?.('zoology', 'Animals')).toStrictEqual(
      asValue({ path: 'Animals.md' }),
    )
    expect(inside.createDeck).toHaveBeenCalledWith('Animals', 'zoology')
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
