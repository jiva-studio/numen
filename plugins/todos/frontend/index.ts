/**
 * Frontend entrypoint for the Todos plugin: a title bar chip that counts the overdue tasks and the
 * tasks due today, and a popover with the lists.
 */
import { defineComponent, h, watch } from 'vue'
import {
  definePlugin,
  type Disposable,
  type HostServices,
  type PluginContext,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import { todosApi, type TodosApi } from './api/wire'
import { createRefreshRelay } from './model/createRefreshRelay'
import type { Timer } from './model/timer'
import { createFieldCache } from './model/createFieldCache'
import { freshTaskLine } from './model/freshTaskLine'
import { useTaskFeed, type TaskFeed } from './model/useTaskFeed'
import type { Filter, Task } from './types'
import { TodosStatus } from './status'
import { createTaskEditorExtension, TASKS_EXTENSION } from './editor'

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

/** The tasks of the note as it is now, and the stored ones where the note cannot be read again. */
async function currentTasks(
  api: TodosApi,
  today: string,
  target: { path: string; line: number; raw: string },
): Promise<readonly Task[]> {
  try {
    await api.refreshIndex([target.path], [])
    const filter: Filter = {
      status: ['open', 'done', 'other'],
      due: 'any',
      from: '',
      to: '',
      projects: [],
      contexts: [],
      priorities: [],
      path: target.path,
      text: '',
    }
    const result = await api.listTasks(today, filter, 2000)
    return result.ok ? result.value.tasks : []
  } catch {
    return []
  }
}

/** The 1-based line the editor of the note counts the task on: from the prose, below the frontmatter. */
async function editorLine(
  api: TodosApi,
  today: string,
  target: { path: string; line: number; raw: string },
): Promise<number> {
  const tasks = await currentTasks(api, today, target)
  return freshTaskLine(tasks, target)?.proseLine ?? target.line
}

function createTodosView(feed: TaskFeed, api: TodosApi, workspace: WorkspaceHost) {
  return defineComponent({
    name: 'TodosView',
    setup: () => () =>
      h(TodosStatus, {
        summary: feed.summary.value,
        api,
        getToday: () => workspace.window.getToday(),
        onOpen: async (target: { path: string; line: number; raw: string }) => {
          const line = await editorLine(api, workspace.window.getToday(), target)
          return workspace.files.open(target.path, { line: line - 1 })
        },
        onShowError: (message: string) => workspace.messages.showError(message),
      }),
  })
}

export const activate = (
  plugin: PluginContext,
  host: HostServices,
  parts: TodosParts = browserParts,
): void => {
  const { workspace } = host.ui
  if (!workspace) {
    throw new Error('the todos plugin needs the window it is drawn in')
  }
  const fields = createFieldCache(parts.api)
  const refreshFields = (): void => {
    fields.refresh().catch(() => undefined) // the lists stay as last read; the next change or focus reads again
  }
  const feed = useTaskFeed(parts.api, () => workspace.window.getToday(), parts.timer, refreshFields)
  const refreshSummary = (): void => {
    feed.refresh().catch(() => undefined) // the counts stay as last read; the next change or focus reads again
    refreshFields()
  }
  // The counts are read again once the backend has checked the changed notes, so the chip does not
  // depend on the stream alone.
  const relay = createRefreshRelay(parts.api, parts.timer, refreshSummary)

  plugin.register.slotView({
    id: TODOS_VIEW,
    component: createTodosView(feed, parts.api, workspace),
  })
  const editorExtension = plugin.register.editorExtension?.bind(plugin.register)
  if (!editorExtension) {
    throw new Error('the todos plugin needs the editor extensions of the host')
  }
  let registration: Disposable | undefined
  const registerTasks = (): void => {
    registration?.dispose()
    registration = editorExtension({
      id: TASKS_EXTENSION,
      extension: createTaskEditorExtension({
        api: parts.api,
        getFields: fields.get,
        loadFields: fields.ensure,
        hideFields: plugin.settings.get<boolean>('hideFields') === true,
        getToday: () => workspace.window.getToday(),
        showError: (message) => workspace.messages.showError(message),
      }),
    })
  }
  registerTasks()
  plugin.settings.onChange<boolean>('hideFields', registerTasks)
  const stopWatchingReady = watch(
    () => feed.summary.value?.isReady === true,
    (isReady) => {
      if (isReady) refreshFields()
    },
  )
  host.events.onVaultChange((event) => relay.push(event))
  window.addEventListener('focus', refreshSummary)
  plugin.identity.subscriptions.push({
    dispose: () => {
      window.removeEventListener('focus', refreshSummary)
      stopWatchingReady()
      relay.dispose()
      feed.stop()
    },
  })
  feed.start()
  refreshFields()
}

export default definePlugin({
  activate,
})
