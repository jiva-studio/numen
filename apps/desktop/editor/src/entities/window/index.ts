/** The window itself: its background tasks, the quit request, the focus requests and its runtime. */
export type { Task } from './lib/task'
export { watchTasks, cancelTask } from './api/tasks'
export { watchQuit, reportFlush } from './api/quit'
export { watchFocus } from './api/focus'
export { getShownVault } from './api/shownVault'
export { closeWindow, closeWindowWhenEmpty, focusWindow } from './model/windowRuntime'
export { getDetachedTabId, isSecondaryWindow } from './model/windowKind'
