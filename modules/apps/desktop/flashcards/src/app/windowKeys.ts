/**
 * What a keystroke asks of the window, screen by screen.
 *
 * Each screen reads its own keys: the vaults screen opens the vault a letter
 * stands at, the decks screen picks what to sit down to, and the session screen
 * turns cards over and answers them. What a key means is the keyboard's answer;
 * what is done about it is the window's.
 */
import { getVaultForKey } from '@numen/ui'

import { getPickerKeyIntent, getSessionKeyIntent, isSwallowed } from '@/features/keyboard'
import { canStart } from '@/pages/decks'
import type { Grade } from '@/entities/card'
import type { VaultCardsDue } from '@/entities/vault'
import type { Preset } from '@/pages/decks'

/** The screens the window stands on, in the order a person goes through them. */
export type WindowScreen = 'vaults' | 'decks' | 'session'

/** What the keys ask of the window: what it is showing, and what it can do. */
export interface WindowKeysDeps {
  /** Which screen is on. */
  readonly screen: () => WindowScreen
  /** The vaults, in the order they are drawn. */
  readonly vaults: () => readonly VaultCardsDue[]
  /** The vault whose decks are open, and nothing before one is. */
  readonly chosen: () => VaultCardsDue | null
  /** The preset each deck is scheduled by, by the path the deck is filed under. */
  readonly byDeck: () => ReadonlyMap<string, Preset>
  /** Whether the answer is already showing. */
  readonly shown: () => boolean
  /** Which of the card and the two panels beside it the window is showing. */
  readonly showing: () => 'reading' | 'here' | 'asking'
  /** Into a vault. */
  readonly choose: (vault: string) => void
  /** Sit down to one deck, or to the whole vault under an empty path. */
  readonly start: (deck: string) => void
  /** Turn the card over. */
  readonly show: () => void
  /** Answer the card in front of the person. */
  readonly answer: (how: Grade) => void
  /** Take the last answer back. */
  readonly takeBack: () => void
  /** Out of the session and back to the decks. */
  readonly leave: () => void
  /** Back to the vaults. */
  readonly goToVaults: () => void
  /** The panel a card is asked about in, brought in or taken away. */
  readonly toggleAgent: () => void
  /** The panel the deck's notes are read in, brought in or taken away. */
  readonly toggleNotes: () => void
  /** The reading moved a page, forward or back. */
  readonly scrollPage: (back: boolean) => void
  /** The panel a card is asked about in, put away. */
  readonly closeAgent: () => void
  /** The panel the deck's notes are read in, put away. */
  readonly closeNotes: () => void
}

export const createWindowKeys = (deps: WindowKeysDeps) => {
  /**
   * The keys a person picks a vault with: a letter opens the vault standing at
   * it, which is the letter drawn on that row. A vault whose count has not
   * arrived carries no letter, and the letter standing at it opens nothing.
   */
  const handleVaultKey = (press: KeyboardEvent) => {
    const vaults = deps.vaults()
    const at = getVaultForKey(press, vaults.length)
    const one = at === null ? undefined : vaults[at]
    if (!one || !one.counted) return
    press.preventDefault()
    deps.choose(one.vault)
  }

  /** The keys a person picks what to sit down to with. */
  const handleDeckKey = (press: KeyboardEvent, vault: VaultCardsDue) => {
    const asked = getPickerKeyIntent(press, vault.decks.length)
    if (!asked) return
    press.preventDefault()

    switch (asked.does) {
      case 'all':
        if (vault.due + vault.new > 0) deps.start('')
        break
      case 'deck': {
        const deck = vault.decks[asked.at]
        if (deck && canStart(deck, deps.byDeck())) deps.start(deck.deck)
        break
      }
      case 'back':
        deps.goToVaults()
        break
    }
  }

  const handleSessionKey = (press: KeyboardEvent) => {
    const asked = getSessionKeyIntent(press, {
      shown: deps.shown(),
      asking: deps.showing() === 'asking',
      reading: deps.showing() === 'reading',
    })
    if (!asked) return
    if (isSwallowed(asked)) press.preventDefault()

    switch (asked.does) {
      case 'show':
        deps.show()
        break
      case 'answer':
        deps.answer(asked.how)
        break
      case 'takeBack':
        deps.takeBack()
        break
      case 'leave':
        deps.leave()
        break
      case 'ask':
        deps.toggleAgent()
        break
      case 'read':
        deps.toggleNotes()
        break
      case 'scroll':
        deps.scrollPage(asked.back)
        break
      case 'shut':
        if (deps.showing() === 'asking') deps.closeAgent()
        else deps.closeNotes()
        break
    }
  }

  const handleKey = (press: KeyboardEvent) => {
    const on = deps.screen()
    if (on === 'vaults') return handleVaultKey(press)
    if (on === 'decks') {
      const vault = deps.chosen()
      return vault ? handleDeckKey(press, vault) : undefined
    }
    if (on === 'session') handleSessionKey(press)
  }

  return { handleKey }
}
