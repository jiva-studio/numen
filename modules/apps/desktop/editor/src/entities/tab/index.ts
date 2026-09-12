/** The tabs a window keeps: what kinds there are, what opens one, and what each is called. */
export { iconOfKind } from './icons'
export { createFileCreators } from './makers'
export { fileOpeners } from './openers'
export type { EditorKind, FileOpeners, SourceReader } from './openers'
export type { Attention, Tab } from './tab'
export { useWindowTabs } from './windowTabs'
export type { AnyTabKind, OpenTab, TabKind, WindowHandle } from './windowTabs'
export {
  AGENT,
  begun,
  BOOK,
  CONVERSATION,
  DECK,
  DOCUMENT,
  FILES,
  generateId,
  NOTE,
  PLEX,
  PRESET,
  RECORDING,
  SETTINGS,
  SETTINGS_FILE,
  STENCIL,
  URL,
} from './workspace'
