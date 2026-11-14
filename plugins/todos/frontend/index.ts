/**
 * Frontend entrypoint for the Todos plugin: a title bar chip that counts the overdue tasks and the
 * tasks due today, and a popover with the lists.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type PluginContext, type WorkspaceHost } from '@numen/plugin-sdk'
import { todosApi, type TodosApi } from './api/wire'
import { createRefreshRelay } from './model/createRefreshRelay'
import type { Timer } from './model/timer'
import { useTaskFeed, type TaskFeed } from './model/useTaskFeed'
import { TodosStatus } from './status'

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

function createTodosView(feed: TaskFeed, api: TodosApi, workspace: WorkspaceHost) {
  return defineComponent({
    name: 'TodosView',
    setup: () => () =>
      h(TodosStatus, {
        summary: feed.summary.value,
        api,
        getToday: () => workspace.window.getToday(),
        onOpen: (target: { path: string; line: number }) =>
          workspace.files.open(target.path, { line: target.line }),
      }),
  })
}

export const activate = (context: PluginContext, parts: TodosParts = browserParts): void => {
  const { workspace } = context.ui
  if (!workspace) {
    throw new Error('the todos plugin needs the window it is drawn in')
  }
  const feed = useTaskFeed(parts.api, () => workspace.window.getToday(), parts.timer)
  const refreshSummary = (): void => {
    feed.refresh().catch(() => undefined)
  }
  // The counts are read again once the backend has checked the changed notes, so the chip does not
  // depend on the stream alone.
  const relay = createRefreshRelay(parts.api, parts.timer, refreshSummary)

  context.register.slotView({
    id: TODOS_VIEW,
    component: createTodosView(feed, parts.api, workspace),
  })
  context.events.onVaultChange((event) => relay.push(event))
  window.addEventListener('focus', refreshSummary)
  context.plugin.subscriptions.push({
    dispose: () => {
      window.removeEventListener('focus', refreshSummary)
      relay.dispose()
      feed.stop()
    },
  })
  feed.start()
}

export default definePlugin({
  activate,
})
