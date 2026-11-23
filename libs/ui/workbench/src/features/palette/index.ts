/** The command palette: the list a window offers, and the keystrokes on it. */
export { default as Palette } from './ui/Palette.vue'
export { createCommandKeyChord } from './lib/keys'
export type { ActionWords } from './lib/actions'
export type { PaletteGroup, PaletteItem } from './lib/item'
export type { PaletteMultiProps } from './lib/multi'
