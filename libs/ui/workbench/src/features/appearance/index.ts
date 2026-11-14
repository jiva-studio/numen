/** The appearance of a window: the theme, the mode and the sizes it is drawn at, and the palette they are chosen in. */
export { createWindowAppearance, type WindowAppearance } from './model/appearance'
export { useAppearancePalette, type AppearancePaletteDeps } from './model/useAppearancePalette'
export { MODES } from './lib/appearance'
export type {
  Appearance,
  AppearanceWords,
  AppearanceWriter,
  Mode,
  Sizes,
  Theme,
  ThemeOrigin,
  Themes,
} from './lib/appearance'
export { INTERFACE_SCALE, TEXT_SCALE, type ScaleKind } from './lib/appearanceValues'
export type { AppearanceChoiceWords } from './lib/choices'
