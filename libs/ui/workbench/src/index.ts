/**
 * The chrome of a desktop window: the panes and their tabs, the command
 * palette, and the status bar.
 */
import './styles.css'
// The package marks only stylesheets as side effects, so the frame's stylesheets are imported by the
// entry, which every window loads, and not by the module of the frame's sizes.
import './features/window-frame/tokens/frame.css'
import './features/window-frame/ui/windowFrame.css'

export { WorkspaceLayout } from './features/workspace'
export { closeTab, openTab, openTabAdjacent } from './features/workspace'

/** For arranging without drawing, or reading a gesture without this renderer. */
export { branch, pane } from './features/workspace'
export { paneById, getPanes, getTabPaneRect } from './features/workspace'
export type { Tab, TabMenu, WindowEdge, Workspace } from './features/workspace'

/** The theme, the mode and the sizes a window is drawn at, and the palette they are chosen in. */
export {
  createWindowAppearance,
  useAppearancePalette,
  MODES,
  INTERFACE_SCALE,
  TEXT_SCALE,
} from './features/appearance'
export type {
  Appearance,
  AppearanceChoiceWords,
  AppearancePaletteDeps,
  AppearanceWords,
  AppearanceWriter,
  Mode,
  ScaleKind,
  Sizes,
  Theme,
  ThemeOrigin,
  Themes,
  WindowAppearance,
} from './features/appearance'

export { Palette } from './features/palette'
export { commandKeyChord } from './features/palette'
export type { ActionWords, PaletteItem, PaletteGroup } from './features/palette'

/** What a count counts, and how a notice about it reads. */
export type { TallyUnit, Severity } from './features/notices'

export { getTaskNotice } from './features/notices'
/** One list of notices for a window, given by the parts it hosts. */
export { createNoticeQueue, type NoticeQueue } from './features/notices'
export type { Notice, Stay, Task } from './features/notices'
/** The background tasks of a window, and the failures it reports, as the list a plugin draws. */
export { createBackgroundTaskHost, createFailedTask, getFailedTasks } from './features/notices'
export type {
  BackgroundTask,
  BackgroundTaskAction,
  BackgroundTaskHost,
  BackgroundTaskHostDeps,
  WorkTask,
} from './features/notices'

/** The screen a window opens on, and the letter a vault on it is opened by. */
export { WelcomePage, getVaultForKey } from './features/welcome'
export type { VaultRow, WelcomeAction } from './features/welcome'

/**
 * The room the macOS buttons take at the left of a strip of tabs at the top of a window without a
 * title bar. The shell draws no bar and no window buttons.
 */
export { MAC_BUTTONS_INSET } from './features/window-frame'

/** The frame of the desktop window a page is drawn in, as both windows read it. */
export { getWindowEdge, useWindowFrame } from './features/window-frame'
/** The runtime the desktop shell serves beside the page, and the system browser it opens an address in. */
export { loadFrameRuntime, openExternalUrl } from './features/window-frame'
export type {
  FrameEdge,
  FrameRuntime,
  FrameRuntimeLoader,
  WindowFrame,
  WindowPlatform,
} from './features/window-frame'
