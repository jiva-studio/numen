/** The action panel as plain values: the words it is drawn with, and its rows. */
import type { Keystroke } from '@numen/ui'
import type { PaletteAction } from './item'
import { PALETTE_KEYS } from './keys'
import { getParts, type PalettePart } from './parts'

/** The words the action panel is drawn with. */
export interface ActionWords {
  readonly name: string
  readonly placeholder: string
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
  readonly at: number
  readonly name: readonly PalettePart[]
  readonly key: Keystroke | null
}

/**
 * The actions the words in the field leave, numbered as drawn.
 * The order is the order they were offered in, which is the order the keys were handed out in, so an action keeps its key wherever the words put it.
 * A word matches anywhere in the name, ignoring case. Nothing typed leaves every action.
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
      name: getParts(action.text, [{ start: found, end: found + word.length }]),
      key: PALETTE_KEYS[proposed] ?? null,
    })
  }
  return out
}

/**
 * Where the panel focus lands once its list has changed: on the action it was
 * on, wherever that action moved to. If that action is gone, focus goes to the
 * first action; if the list is empty, nowhere.
 */
export const findKeptAction = (actions: readonly PlacedAction[], was: string): number => {
  const pos = actions.findIndex((one) => one.action.id === was)
  return Math.max(pos, actions.length > 0 ? 0 : -1)
}
