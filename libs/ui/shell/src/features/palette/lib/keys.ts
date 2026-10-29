/** The keystrokes a palette answers to, and the caps they are drawn as. */
import { keyChord, type PaletteKeys } from '@numen/ui'
import { canChoose, type PaletteAction, type PaletteItem } from './item'

/** The keys that reach an item's actions, in the order the actions are offered. */
export const PALETTE_KEYS: readonly PaletteKeys[] = [
  { icons: ['return'], letter: '' },
  { icons: ['shift', 'return'], letter: '' },
]

/** One action and the key that triggers it directly from the list. */
export interface PaletteShortcut {
  readonly action: PaletteAction
  readonly key: PaletteKeys | null
}

/**
 * The actions of this item that have a key, in display order. The palette
 * footer lists them, and the key assignment is decided here.
 */
export const getShortcuts = (item: PaletteItem | undefined): readonly PaletteShortcut[] => {
  const actions = item && canChoose(item) ? (item.actions ?? []) : []
  return actions
    .slice(0, PALETTE_KEYS.length)
    .map((action, index) => ({ action, key: PALETTE_KEYS[index] ?? null }))
}

/** The id of the action triggered by Enter, or by Shift+Enter when `second` is set. */
export const getActionId = (item: PaletteItem | undefined, second: boolean): string => {
  const actions = item && canChoose(item) ? item.actions : undefined
  return actions?.[second ? 1 : 0]?.id ?? ''
}

/** Whether this keystroke opens the action panel. */
export const isActionsChord = (event: {
  readonly key: string
  readonly ctrlKey: boolean
  readonly metaKey: boolean
  readonly shiftKey: boolean
}): boolean =>
  (event.ctrlKey || event.metaKey) && !event.shiftKey && event.key.toLowerCase() === 'k'

/** The keystroke that opens the action panel. */
export const commandKeyChord = (agent: string): PaletteKeys => keyChord('k', agent)
