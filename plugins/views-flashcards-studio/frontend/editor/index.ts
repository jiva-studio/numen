/** The editors of a deck and of a card model, the tabs a vault's flashcard files open in. */
import type { Presets } from '../lib/presets'
import DeckTab from './ui/deck/DeckTab.vue'
import CardModelTab from './ui/card-model/CardModelTab.vue'
import { useDeckTabs as useDeckTabsOf, type DeckTabsOptions } from './model/deck/useDeckTabs'
import {
  createCardModelTabs as createCardModelTabsOf,
  type CardModelTabsOptions,
} from './model/card-model/kind'
import type { Cards } from '../types'

export { DECK } from './model/deck/kind'
export { CARD_MODEL } from './model/card-model/kind'
export { DeckTab, CardModelTab }
export type { DeckTabState } from './model/deck/tabState'
export type { CardModelTabState } from './model/card-model/types'

/** The deck tabs of a window, drawn by `DeckTab`. */
export const useDeckTabs = (
  cards: Cards,
  presets: Presets,
  options: Omit<DeckTabsOptions, 'component'>,
) => useDeckTabsOf(cards, presets, { ...options, component: DeckTab })

/** The card model tabs of a window, drawn by `CardModelTab`. */
export const createCardModelTabs = (
  cards: Cards,
  options: Omit<CardModelTabsOptions, 'component'>,
) => createCardModelTabsOf(cards, { ...options, component: CardModelTab })
