/** The window itself: its background tasks, the quit request, the focus requests and its runtime. */
export type { Task } from './lib/task'
export { createTaskApi, type TaskApi } from './api/tasks'
export { createQuitApi, type QuitApi } from './api/quit'
export { createFocusApi, type FocusApi } from './api/focus'
export { createShownVaultApi, type ShownVaultApi } from './api/shownVault'
export { closeWindow, closeWindowWhenEmpty, focusWindow } from './model/windowRuntime'
export { getDetachedTabId, isSecondaryWindow } from './model/windowKind'
