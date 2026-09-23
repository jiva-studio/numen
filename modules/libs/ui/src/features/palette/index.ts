/** The command palette: the list a window offers, and the keystrokes on it. */
export { default as Palette } from './ui/Palette.vue'
export { commandKeyChord, keyChord } from './lib/keys'
export type { ActionWords } from './lib/actions'
export type { PaletteAction, PaletteGroup, PaletteItem } from './lib/item'
