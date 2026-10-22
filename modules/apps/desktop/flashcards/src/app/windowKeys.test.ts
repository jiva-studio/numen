/**
 * A key is read by the screen that is on, and a key the screen acts on is kept
 * from the page. A key the screen does not read is left to the page.
 */
import { describe, expect, it, vi } from 'vitest'

import { createWindowKeys } from './windowKeys'
import type { WindowKeysDeps, WindowScreen } from './windowKeys'
import type { VaultCardsDue } from '@/entities/vault'

/** One keystroke, as the window meets it. */
const createPress = (key: string, more: Partial<KeyboardEvent> = {}) =>
  ({
    key,
    repeat: false,
    altKey: false,
    ctrlKey: false,
    metaKey: false,
    shiftKey: false,
    preventDefault: vi.fn(),
    ...more,
  }) as unknown as KeyboardEvent & { preventDefault: ReturnType<typeof vi.fn> }

/** A vault as the counting hands one over. */
const createVault = (more: Partial<VaultCardsDue> = {}): VaultCardsDue => ({
  vault: 'garden',
  name: 'Garden',
  path: '/vaults/Garden',
  isCounted: true,
  faces: 4,
  due: 2,
  new: 2,
  decks: [{ deck: 'decks/Leaves.md', faces: 4, due: 2, new: 2, learned: 0, unbegun: 0 }],
  presets: [],
  unread: '',
  isIndexing: false,
  ...more,
})

const createKeys = (
  screen: WindowScreen,
  vault: VaultCardsDue,
  more: Partial<WindowKeysDeps> = {},
) => {
  const deps: WindowKeysDeps = {
    getScreen: () => screen,
    getVaults: () => [vault],
    getChosenVault: () => vault,
    getPresetsByDeck: () => new Map(),
    isShown: () => false,
    getShowing: () => 'here',
    choose: vi.fn(),
    start: vi.fn(),
    show: vi.fn(),
    answer: vi.fn(),
    takeBack: vi.fn(),
    leave: vi.fn(),
    goToVaults: vi.fn(),
    toggleAgent: vi.fn(),
    toggleNotes: vi.fn(),
    scrollPage: vi.fn(),
    closeAgent: vi.fn(),
    closeNotes: vi.fn(),
    ...more,
  }
  return { ...createWindowKeys(deps), deps }
}

describe('the keys of the vaults screen', () => {
  it('opens the vault a letter remains at, and keeps the letter from the page', () => {
    const { onKeyDown, deps } = createKeys('vaults', createVault())
    const press = createPress('a')

    onKeyDown(press)

    expect(deps.choose).toHaveBeenCalledWith('garden')
    expect(press.preventDefault).toHaveBeenCalled()
  })
})

describe('the keys of the decks screen', () => {
  it('opens vault with only new cards and consumes key event', () => {
    const { onKeyDown, deps } = createKeys('decks', createVault({ due: 0, new: 3 }))
    const press = createPress('Enter')

    onKeyDown(press)

    expect(deps.start).toHaveBeenCalledWith('')
    expect(press.preventDefault).toHaveBeenCalled()
  })

  it('does not sit down to a vault that owes nothing', () => {
    const { onKeyDown, deps } = createKeys('decks', createVault({ due: 0, new: 0 }))

    onKeyDown(createPress('Enter'))

    expect(deps.start).not.toHaveBeenCalled()
  })

  it('leaves a key it does not read to the page', () => {
    const { onKeyDown, deps } = createKeys('decks', createVault())
    const press = createPress('Tab')

    onKeyDown(press)

    expect(press.preventDefault).not.toHaveBeenCalled()
    expect(deps.start).not.toHaveBeenCalled()
    expect(deps.goToVaults).not.toHaveBeenCalled()
  })
})

describe('the keys of the session screen', () => {
  it('reads escape as leaving the session, with a vault chosen behind it', () => {
    const { onKeyDown, deps } = createKeys('session', createVault())

    onKeyDown(createPress('Escape'))

    expect(deps.leave).toHaveBeenCalled()
    expect(deps.goToVaults).not.toHaveBeenCalled()
  })

  it('turns the card over, keeping the space bar from the page', () => {
    const { onKeyDown, deps } = createKeys('session', createVault())
    const press = createPress(' ')

    onKeyDown(press)

    expect(deps.show).toHaveBeenCalled()
    expect(press.preventDefault).toHaveBeenCalled()
  })

  it('answers the card, leaving the number to the page', () => {
    const { onKeyDown, deps } = createKeys('session', createVault(), { isShown: () => true })
    const press = createPress('3')

    onKeyDown(press)

    expect(deps.answer).toHaveBeenCalledWith('good')
    expect(press.preventDefault).not.toHaveBeenCalled()
  })

  it('brings each panel in with the overlay key', () => {
    const { onKeyDown, deps } = createKeys('session', createVault())

    onKeyDown(createPress('a', { ctrlKey: true }))
    expect(deps.toggleAgent).toHaveBeenCalledOnce()
    expect(deps.toggleNotes).not.toHaveBeenCalled()

    onKeyDown(createPress('r', { ctrlKey: true }))
    expect(deps.toggleNotes).toHaveBeenCalledOnce()
    expect(deps.toggleAgent).toHaveBeenCalledOnce()
  })

  it('puts the conversation away on escape, and stays in the session', () => {
    const { onKeyDown, deps } = createKeys('session', createVault(), {
      getShowing: () => 'asking',
    })

    onKeyDown(createPress('Escape'))

    expect(deps.closeAgent).toHaveBeenCalled()
    expect(deps.closeNotes).not.toHaveBeenCalled()
    expect(deps.leave).not.toHaveBeenCalled()
  })

  it('puts the reading away on escape, and stays in the session', () => {
    const { onKeyDown, deps } = createKeys('session', createVault(), {
      getShowing: () => 'reading',
    })

    onKeyDown(createPress('Escape'))

    expect(deps.closeNotes).toHaveBeenCalled()
    expect(deps.closeAgent).not.toHaveBeenCalled()
    expect(deps.leave).not.toHaveBeenCalled()
  })
})
