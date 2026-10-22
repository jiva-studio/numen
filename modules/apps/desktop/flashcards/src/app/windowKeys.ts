/**
 * What a keystroke asks of the window, screen by screen.
 *
 * Each screen reads its own keys: the vaults screen opens the vault a letter
 * turns cards over and answers them. What a key means is the keyboard's answer;
 * what is done about it is the window's.
 */
import { getVaultForKey } from '@numen/ui'

import { getPickerKeyIntent, getSessionKeyIntent, shouldConsumeKey } from '@/features/keyboard'
import { canStart } from '@/pages/decks'
import type { Grade } from '@/entities/card'
import type { VaultCardsDue } from '@/entities/vault'
import type { SessionKeyIntent } from '@/features/keyboard'
import type { Preset } from '@/pages/decks'

/** The screens the window relies on, in the order a person goes through them. */
export type WindowScreen = 'vaults' | 'decks' | 'session'

/** What the keys ask of the window: what it is showing, and what it can do. */
export interface WindowKeysDeps {
  /** Which screen is on. */
  readonly getScreen: () => WindowScreen
  /** The vaults, in the order they are drawn. */
  readonly getVaults: () => readonly VaultCardsDue[]
  /** The vault whose decks are open, and nothing before one is. */
  readonly getChosenVault: () => VaultCardsDue | null
  /** The preset each deck is scheduled by, by the path the deck is filed under. */
  readonly getPresetsByDeck: () => ReadonlyMap<string, Preset>
  /** Whether the answer is already showing. */
  readonly isShown: () => boolean
  /** Which of the card and the two panels beside it the window is showing. */
  readonly getShowing: () => 'reading' | 'here' | 'asking'
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

const applyVaultKey = (press: KeyboardEvent, deps: WindowKeysDeps) => {
  const vaults = deps.getVaults()
  const at = getVaultForKey(press, vaults.length)
  const one = vaults[at ?? -1]
  if (!one?.isCounted) return
  press.preventDefault()
  deps.choose(one.vault)
}

const applyDeckKey = (press: KeyboardEvent, vault: VaultCardsDue, deps: WindowKeysDeps) => {
  const request = getPickerKeyIntent(press, vault.decks.length)
  if (!request) return
  press.preventDefault()

  switch (request.does) {
    case 'all':
      if (vault.due + vault.new > 0) deps.start('')
      break
    case 'deck': {
      const deck = vault.decks[request.at]
      if (deck && canStart(deck, deps.getPresetsByDeck())) deps.start(deck.deck)
      break
    }
    case 'back':
      deps.goToVaults()
      break
  }
}

const applySessionIntent = (
  intent: SessionKeyIntent,
  deps: WindowKeysDeps,
  closeShownPanel: () => void,
) => {
  switch (intent.does) {
    case 'show':
      deps.show()
      break
    case 'answer':
      deps.answer(intent.how)
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
      deps.scrollPage(intent.back)
      break
    case 'shut':
      closeShownPanel()
      break
  }
}

export const createWindowKeys = (deps: WindowKeysDeps) => {
  const closeShownPanel = () => {
    if (deps.getShowing() === 'asking') deps.closeAgent()
    else deps.closeNotes()
  }

  const onSessionKey = (press: KeyboardEvent) => {
    const request = getSessionKeyIntent(press, {
      isShown: deps.isShown(),
      isAsking: deps.getShowing() === 'asking',
      isNotesOpen: deps.getShowing() === 'reading',
    })
    if (!request) return
    if (shouldConsumeKey(request)) press.preventDefault()
    applySessionIntent(request, deps, closeShownPanel)
  }

  const onKeyDown = (press: KeyboardEvent) => {
    const on = deps.getScreen()
    if (on === 'vaults') return applyVaultKey(press, deps)
    if (on === 'decks') {
      const vault = deps.getChosenVault()
      return vault ? applyDeckKey(press, vault, deps) : undefined
    }
    onSessionKey(press)
  }

  return { onKeyDown }
}
