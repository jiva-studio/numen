/**
 * Invented chip faces and popover words for the tests and stories of the bar chip. Not shipped.
 */
import { DEVANAGARI, EMPTY } from '@/shared/fixtures/prose'

export const LABEL = 'Tide level, rising'
export const TOOLTIP = 'Tide level at the harbour gauge'
export const FACE = 'Tide 4.2 m'
export const POPOVER_LABEL = 'Tide details'
export const POPOVER_TEXT = 'High water at 14:20, low water at 20:35.'
export const POPOVER_ACTION = 'Mark the reading'

/** 200 characters with nowhere to break. */
export const UNBROKEN_200 = 'x'.repeat(200)

/** Faces that test the height of a chip: none, one character, a non-Latin script, too long. */
export const AWKWARD_FACES: readonly { readonly name: string; readonly text: string }[] = [
  { name: 'empty', text: EMPTY },
  { name: 'one character', text: '7' },
  { name: 'devanagari', text: DEVANAGARI },
  { name: 'unbroken', text: UNBROKEN_200 },
]
