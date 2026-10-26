/** The tabs a window keeps: what kinds there are, what opens one, and what each is called. */
export { findTabKindIcon } from './lib/icons'
export { createFileCreators } from './model/makers'
export { fileOpeners } from './model/openers'
export type {
  EditorKind,
  FileOpener,
  FilePlacement,
  FileOpeners,
  OpenPlacement,
  SourceReader,
} from './model/openers'
export type { OpenTabs, Tab } from './lib/tab'
export type { AnyTabKind, OpenTab, TabKind, WindowHandle } from './lib/kinds'
export { useWindowTabs } from './model/windowTabs'
export {
  BOOK,
  CONVERSATION,
  createWorkspace,
  DOCUMENT,
  generateId,
  NOTE,
  RECORDING,
  SETTINGS,
  SETTINGS_FILE,
  URL,
} from './lib/workspace'
