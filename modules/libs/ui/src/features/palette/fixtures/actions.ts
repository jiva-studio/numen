/** Actions to offer an item in a test. */
import type { PaletteAction } from '../item'

/** Five actions, which is three more than there are keys. */
export const MANY: readonly PaletteAction[] = [
  { id: 'travel', text: 'Show in plex' },
  { id: 'open', text: 'Open the note' },
  { id: 'beside', text: 'Open beside' },
  { id: 'rename', text: 'Rename' },
  { id: 'remove', text: 'Move to trash' },
]
