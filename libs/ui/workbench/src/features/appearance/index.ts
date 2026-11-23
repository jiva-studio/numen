/** The appearance of a window: the theme, the mode and the sizes it is drawn at, and the palette they are chosen in. */
export { createAppearanceOutput } from './api/appearanceOutput'
export { useAppearancePalette } from './model/useAppearancePalette'
export { MODES } from './lib/appearance'
export type { Mode } from './lib/appearance'
export { createWindowAppearance, type WindowAppearance } from './model/windowAppearance'
export { INTERFACE_SCALE, TEXT_SCALE } from './lib/appearanceValues'
export type { AppearanceWords, ThemeSource } from './lib/appearance'
