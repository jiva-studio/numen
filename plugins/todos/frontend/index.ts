/**
 * Frontend entrypoint for the Todos plugin: a title bar chip that counts the overdue tasks and the
 * tasks due today, and a popover with the lists.
 */
import { defineComponent, h, ref, watch, type Ref } from 'vue'
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
import { createTaskLine } from './model/freshTaskLine'
import { useTaskFeed, type TaskFeed } from './model/useTaskFeed'
import type { Filter, Task } from './types'
import { TodosStatus, type ShownGroups } from './status'
import { WORDS } from './words'
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

type OpenTarget = { path: string; line: number; raw: string }

async function listCurrentTasks(
  api: TodosApi,
  today: string,
  target: OpenTarget,
): Promise<readonly Task[] | null> {
  const refreshed = await api.refreshIndex([target.path], [])
  if (!refreshed.ok) return null
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
  return result.ok ? result.value.tasks : null
}

async function readEditorLine(
  api: TodosApi,
  today: string,
  target: OpenTarget,
  showError: (message: string) => void,
): Promise<number | undefined> {
  // an unreachable backend is the same as a refused read: reported below, the note still opens
  const tasks = await listCurrentTasks(api, today, target).catch((): null => null)
  if (tasks === null) {
    showError(WORDS.taskNotRead)
    return undefined
  }
  const proseLine = createTaskLine(tasks, target)?.proseLine
  return Number.isInteger(proseLine) && proseLine !== undefined && proseLine >= 1
    ? proseLine
    : undefined
}

const SHOWN_SETTINGS = ['showOverdue', 'showToday', 'showNextWeek', 'showNoDate'] as const

function readShownGroups(plugin: PluginContext): ShownGroups {
  const isShown = (setting: string): boolean => plugin.settings.get<boolean>(setting) !== false
  return {
    overdue: isShown('showOverdue'),
    today: isShown('showToday'),
    nextSevenDays: isShown('showNextWeek'),
    noDate: isShown('showNoDate'),
  }
}

function createTodosView(
  feed: TaskFeed,
  api: TodosApi,
  workspace: WorkspaceHost,
  shown: Readonly<Ref<ShownGroups>>,
) {
  return defineComponent({
    name: 'TodosView',
    setup: () => () =>
      h(TodosStatus, {
        summary: feed.summary.value,
        shown: shown.value,
        api,
        getToday: () => workspace.window.getToday(),
        onOpen: async (target: OpenTarget) => {
          const showError = (message: string) => workspace.messages.showError(message)
          try {
            const line = await readEditorLine(api, workspace.window.getToday(), target, showError)
            workspace.files.open(target.path, line === undefined ? undefined : { line: line - 1 })
          } catch {
            showError(WORDS.noteNotOpened)
          }
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
  }
  const refreshAll = (): void => {
    refreshSummary()
    refreshFields()
  }
  // The counts are read again once the backend has checked the changed notes, so the chip does not
  // depend on the stream alone; the fields are read on the change the stream reports.
  const relay = createRefreshRelay(parts.api, parts.timer, refreshSummary)

  const shown = ref(readShownGroups(plugin))
  const settingWatches = SHOWN_SETTINGS.map((key) =>
    plugin.settings.onChange<boolean>(key, () => {
      shown.value = readShownGroups(plugin)
    }),
  )

  plugin.register.slotView({
    id: TODOS_VIEW,
    component: createTodosView(feed, parts.api, workspace, shown),
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
  settingWatches.push(plugin.settings.onChange<boolean>('hideFields', registerTasks))
  const stopWatchingReady = watch(
    () => feed.summary.value?.isReady === true,
    (isReady) => {
      if (isReady) refreshFields()
    },
  )
  host.events.onVaultChange((event) => relay.push(event))
  window.addEventListener('focus', refreshAll)
  plugin.identity.subscriptions.push({
    dispose: () => {
      window.removeEventListener('focus', refreshAll)
      for (const settingWatch of settingWatches) settingWatch.dispose()
      registration?.dispose()
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
