/**
 * Frontend entrypoint for the Todos plugin: a title bar chip that counts the overdue tasks and the
 * tasks due today, and a popover with the lists.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { todosApi, type TodosApi } from './api/wire'
import type { Timer } from './model/timer'

/** The id the manifest declares the title bar view under. */
export const TODOS_VIEW = 'status'

/** What the plugin is made of outside the window, replaced by a test. */
export interface TodosParts {
  readonly api: TodosApi
  readonly timer: Timer
}

const browserParts: TodosParts = {
  api: todosApi,
  timer: {
    set: (run, delayMs) => window.setTimeout(run, delayMs),
    clear: (handle) => window.clearTimeout(handle),
  },
}

export const activate = (context: PluginContext, parts: TodosParts = browserParts): void => {
  void context
  void parts
}

export default definePlugin({
  activate,
})
