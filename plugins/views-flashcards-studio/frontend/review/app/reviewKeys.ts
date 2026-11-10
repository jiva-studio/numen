/**
 * The action of each keystroke in the review, screen by screen.
 *
 * The decks screen selects a deck by its letter, and the session turns cards
 * over and answers them. This module maps a key to an action; the review
 * performs the action.
 */
import { getPickerKeyIntent, getSessionKeyIntent, shouldConsumeKey } from '../features/keyboard'
import { canStart } from '../pages/decks'
import type { Grade } from '../entities/card'
import type { VaultCardsDue } from '../entities/vault'
import type { SessionKeyIntent } from '../features/keyboard'
import type { Preset } from '../pages/decks'

/** The screens of the review, in the order the user visits them. */
export type ReviewScreen = 'decks' | 'session'

/** What the keys ask of the review: what it is showing, and what it can do. */
export interface ReviewKeysDeps {
  readonly getScreen: () => ReviewScreen
  readonly getChosenVault: () => VaultCardsDue | null
  readonly getPresetsByDeck: () => ReadonlyMap<string, Preset>
  readonly isShown: () => boolean
  readonly getShowing: () => 'reading' | 'here' | 'asking'
  readonly start: (deck: string) => void
  readonly show: () => void
  readonly answer: (how: Grade) => void
  readonly takeBack: () => void
  readonly leave: () => void
  readonly goToVaults: () => void
  readonly toggleAgent: () => void
  readonly toggleNotes: () => void
  readonly scrollPage: (back: boolean) => void
  readonly closeAgent: () => void
  readonly closeNotes: () => void
}

const applyDeckKey = (press: KeyboardEvent, vault: VaultCardsDue, deps: ReviewKeysDeps) => {
  const request = getPickerKeyIntent(press, vault.decks.length)
  if (!request) return
  press.preventDefault()

  switch (request.does) {
    case 'all':
      if (vault.totals.due + vault.totals.new > 0) deps.start('')
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
  deps: ReviewKeysDeps,
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

export const createReviewKeys = (deps: ReviewKeysDeps) => {
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

  /** A key pressed while the review is in front, answered with whether the page is kept from it. */
  const onKeyDown = (press: KeyboardEvent): boolean => {
    if (deps.getScreen() === 'session') onSessionKey(press)
    else {
      const vault = deps.getChosenVault()
      if (vault) applyDeckKey(press, vault, deps)
    }
    return press.defaultPrevented
  }

  return { onKeyDown }
}
