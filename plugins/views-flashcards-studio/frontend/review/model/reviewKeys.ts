/**
 * The action of each keystroke in the review, screen by screen.
 *
 * The decks screen selects a deck by its letter, and the session turns cards
 * over and answers them. This module maps a key to an action; the review
 * performs the action.
 */
import { getPickerKeyIntent, getSessionKeyIntent, shouldConsumeKey } from '../lib/keyboard/keys'
import { canStart } from '../lib/decks/progress'
import type { Grade } from '../lib/card/grades'
import type { VaultCardsDue } from '../lib/vault/types'
import type { SessionKeyIntent } from '../lib/keyboard/keys'
import type { Preset } from '../lib/decks/types'

/** The screens of the review, in the order the user visits them. */
export type ReviewScreen = 'decks' | 'session'

/** What the review is showing. */
export interface ReviewKeysView {
  readonly getScreen: () => ReviewScreen
  readonly getChosenVault: () => VaultCardsDue | null
  readonly getPresetsByDeck: () => ReadonlyMap<string, Preset>
  readonly isShown: () => boolean
  readonly getShowing: () => 'reading' | 'here' | 'asking'
}

/** What a key does to the session and to the screens. */
export interface ReviewKeysActions {
  readonly start: (deck: string) => void
  readonly show: () => void
  readonly answer: (how: Grade) => void
  readonly takeBack: () => void
  readonly leave: () => void
  readonly goToVaults: () => void
  readonly scrollPage: (back: boolean) => void
}

/** What a key does to the agent panel and the notes panel. */
export interface ReviewKeysPanels {
  readonly toggleAgent: () => void
  readonly toggleNotes: () => void
  readonly closeAgent: () => void
  readonly closeNotes: () => void
}

/** What the keys ask of the review: what it is showing, and what it can do. */
export interface ReviewKeysDeps {
  readonly view: ReviewKeysView
  readonly actions: ReviewKeysActions
  readonly panels: ReviewKeysPanels
}

const applyDeckKey = (press: KeyboardEvent, vault: VaultCardsDue, deps: ReviewKeysDeps) => {
  const { view, actions } = deps
  const request = getPickerKeyIntent(press, vault.decks.length)
  if (!request) return
  press.preventDefault()

  switch (request.does) {
    case 'all':
      if (vault.totals.due + vault.totals.new > 0) actions.start('')
      break
    case 'deck': {
      const deck = vault.decks[request.at]
      if (deck && canStart(deck, view.getPresetsByDeck())) actions.start(deck.deck)
      break
    }
    case 'back':
      actions.goToVaults()
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
      deps.actions.show()
      break
    case 'answer':
      deps.actions.answer(intent.how)
      break
    case 'takeBack':
      deps.actions.takeBack()
      break
    case 'leave':
      deps.actions.leave()
      break
    case 'ask':
      deps.panels.toggleAgent()
      break
    case 'read':
      deps.panels.toggleNotes()
      break
    case 'scroll':
      deps.actions.scrollPage(intent.back)
      break
    case 'shut':
      closeShownPanel()
      break
  }
}

export const createReviewKeys = (deps: ReviewKeysDeps) => {
  const closeShownPanel = () => {
    if (deps.view.getShowing() === 'asking') deps.panels.closeAgent()
    else deps.panels.closeNotes()
  }

  const onSessionKey = (press: KeyboardEvent) => {
    const request = getSessionKeyIntent(press, {
      isShown: deps.view.isShown(),
      isAsking: deps.view.getShowing() === 'asking',
      isNotesOpen: deps.view.getShowing() === 'reading',
    })
    if (!request) return
    if (shouldConsumeKey(request)) press.preventDefault()
    applySessionIntent(request, deps, closeShownPanel)
  }

  /** A key pressed while the review is in front, answered with whether the page is kept from it. */
  const onKeyDown = (press: KeyboardEvent): boolean => {
    if (deps.view.getScreen() === 'session') onSessionKey(press)
    else {
      const vault = deps.view.getChosenVault()
      if (vault) applyDeckKey(press, vault, deps)
    }
    return press.defaultPrevented
  }

  return { onKeyDown }
}
