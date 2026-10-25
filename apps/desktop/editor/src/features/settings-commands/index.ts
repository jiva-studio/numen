/** The settings the window offers as commands: appearance, syncing, and how headings nest under notes. */
export {
  APPEARANCE,
  APPEARANCE_COMMANDS,
  INTERFACE_SCALE,
  MODE,
  TEXT_SCALE,
} from './lib/appearanceValues'
export { createWindowAppearance } from './model/appearance'
export { NESTING, HEADINGS, useNestingSetting } from './model/nesting'
export { OFF, ON, SYNCING, syncSetting } from './model/sync'
export { IS_MODE, IS_SIZES, IS_THEME, MARKER } from './lib/head'
