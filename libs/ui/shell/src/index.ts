/**
 * The chrome of a desktop window: the panes and their tabs, the command
 * palette, the status bar and the notices in the corner.
 */
import './styles.css'

export { WorkspaceLayout } from './features/workspace'
export { closeTab, openTab, openTabBeside } from './features/workspace'

/** For arranging without drawing, or reading a gesture without this renderer. */
export { branch, pane } from './features/workspace'
export { paneById, getPanes } from './features/workspace'
export type { Tab, Workspace } from './features/workspace'

export { Palette } from './features/palette'
export { commandKeyChord } from './features/palette'
export type { ActionWords, PaletteItem, PaletteGroup } from './features/palette'

/** What a count counts, and how a notice about it reads. */
export type { TallyUnit, Severity } from './features/notices'

export { Notices } from './features/notices'
export { createNotice } from './features/notices'
/** One list of notices for a window, given by the parts it hosts. */
export { createNoticeQueue, type NoticeQueue } from './features/notices'
export type { Notice, Stay, Task } from './features/notices'

export { StatusBar } from './features/status-bar'
export type { StatusBarTask } from './features/status-bar'

/** The screen a window opens on, and the letter a vault on it is opened by. */
export { WelcomePage, getVaultForKey } from './features/welcome'
export type { VaultRow, WelcomeAction } from './features/welcome'
