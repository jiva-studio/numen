/** The keystrokes a palette answers to, and the caps they are drawn as. */
import type { PaletteIcon, PaletteKeys } from '@/shared/ui/key-cap'
import { choosable, type PaletteAction, type PaletteItem } from './item'

/** The keys that reach an item's actions, in the order the actions are offered. */
export const PALETTE_KEYS: readonly PaletteKeys[] = [
  { icons: ['return'], letter: '' },
  { icons: ['shift', 'return'], letter: '' },
]

/** One action, and the key that reaches it straight from the list. */
export interface PaletteShortcut {
  readonly action: PaletteAction
  readonly key: PaletteKeys | null
}

/**
 * The actions of this item a key reaches, in the order they are offered. This
 * is what the foot of the palette says, and where the keys are decided.
 */
export const getShortcuts = (item: PaletteItem | undefined): readonly PaletteShortcut[] => {
  const actions = item && choosable(item) ? (item.actions ?? []) : []
  return actions
    .slice(0, PALETTE_KEYS.length)
    .map((action, at) => ({ action, key: PALETTE_KEYS[at] ?? null }))
}

/** What the action Enter reaches is, and Shift and Enter the second. */
export const actionAt = (item: PaletteItem | undefined, second: boolean): string => {
  const actions = item && choosable(item) ? item.actions : undefined
  return actions?.[second ? 1 : 0]?.id ?? ''
}

/** Whether this keystroke asks for the action panel. */
export const isActionsChord = (event: {
  readonly key: string
  readonly ctrlKey: boolean
  readonly metaKey: boolean
  readonly shiftKey: boolean
}): boolean =>
  (event.ctrlKey || event.metaKey) && !event.shiftKey && event.key.toLowerCase() === 'k'

/**
 * The key beside the space bar on the keyboard a browser says it is running
 * on: Command on Apple keyboards, Control everywhere else.
 */
export const overlayIcon = (agent: string): PaletteIcon =>
  /mac|iphone|ipad|ipod/i.test(agent) ? 'command' : 'control'

/**
 * A keystroke of one letter and the key beside the space bar, as it is drawn.
 * Shift stands between that key and the letter.
 */
export const keyChord = (letter: string, agent: string, shift = false): PaletteKeys => ({
  icons: shift ? [overlayIcon(agent), 'shift'] : [overlayIcon(agent)],
  letter: letter.toUpperCase(),
})

/** The keystroke that opens the action panel. */
export const commandKeyChord = (agent: string): PaletteKeys => keyChord('k', agent)
