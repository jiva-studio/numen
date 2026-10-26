/** The action panel as plain values: the words it is drawn with, and its rows. */
import type { PaletteKeys } from '@numen/ui'
import type { PaletteAction } from './item'
import { PALETTE_KEYS } from './keys'
import { getParts, type PalettePart } from './parts'

/** The words the action panel is drawn with. */
export interface ActionWords {
  /** What the panel is announced as, and what the key to it is called. */
  readonly name: string
  /** The words standing in for what has not been typed in its field. */
  readonly placeholder: string
  /** What it says when the words in its field leave no action. */
  readonly silence: string
}

export const ACTION_WORDS: ActionWords = {
  name: 'Actions',
  placeholder: 'Search actions',
  silence: 'No actions found',
}

/** One action as it is drawn in the action panel. */
export interface PlacedAction {
  readonly action: PaletteAction
  /** Its number in the panel, which is what the keyboard counts in. */
  readonly at: number
  /** Its name, with the run the words in the panel's field picked out. */
  readonly name: readonly PalettePart[]
  /** The key that reaches it without the panel, and nothing where none does. */
  readonly key: PaletteKeys | null
}

/**
 * Every action the words in the panel's field leave, numbered as it is drawn.
 * The order is the order they were offered in, which is the order the keys were
 * handed out in, so an action keeps its key wherever the words put it.
 *
 * A word is looked for anywhere in the name, and case is not part of the
 * question. Nothing typed leaves every action.
 */
export const placeActions = (
  actions: readonly PaletteAction[] = [],
  text = '',
): readonly PlacedAction[] => {
  const word = text.trim().toLowerCase()
  const out: PlacedAction[] = []
  for (const [proposed, action] of actions.entries()) {
    const found = action.text.toLowerCase().indexOf(word)
    if (found < 0) continue
    out.push({
      action,
      at: out.length,
      name: getParts(action.text, [{ from: found, to: found + word.length }]),
      key: PALETTE_KEYS[proposed] ?? null,
    })
  }
  return out
}

/**
 * Where the panel appears once its list has changed under it: on the action it
 * was on, wherever that action has moved to. An action that is gone hands it to
 * the first there is; a list holding none takes it nowhere.
 */
export const findKeptAction = (actions: readonly PlacedAction[], was: string): number => {
  const pos = actions.findIndex((one) => one.action.id === was)
  return Math.max(pos, actions.length > 0 ? 0 : -1)
}
