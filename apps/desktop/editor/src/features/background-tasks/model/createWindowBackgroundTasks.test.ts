import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { BackgroundTask } from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import type { PluginHealthItem } from '@/entities/plugin'
import type { Task } from '@/entities/window'
import { createMessageLog } from '@/shared/model/notices/messages'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createWindowBackgroundTasks } from './createWindowBackgroundTasks'
import { getHealthTasks, runHealthAction } from './healthTasks'

const READING: Task = {
  id: 'reading',
  label: 'Reading',
  about: '',
  progress: { completedCount: 1, total: 4, counting: 'items' },
  error: '',
  source: { isUserStarted: false, pluginId: '', pluginName: '' },
  isCancellable: true,
  kind: 'work',
}

const CONNECTION: PluginHealthItem = {
  pluginId: 'org.example.atlas',
  key: 'connection',
  severity: 'error',
  text: 'Choose a connection',
  detail: 'The atlas has no connection to ask.',
  action: {
    command: 'app:settings.open',
    values: { plugin: 'org.example.atlas', setting: 'source' },
    title: '',
  },
}

const QUOTA: PluginHealthItem = {
  pluginId: 'org.example.atlas',
  key: 'quota',
  severity: 'warning',
  text: 'Half of the quota is used',
  detail: '',
}

const setup = (
  unwatched = '',
  items: readonly PluginHealthItem[] = [],
  ids = ['app:settings.open'],
) => {
  const log = createMessageLog()
  const tasks = ref<readonly Task[]>([READING])
  const runs: { command: string; values: Readonly<Record<string, string>> | undefined }[] = []
  const host = createWindowBackgroundTasks({
    cancelTask: createAnsweredApi({}).window.tasks.cancelTask,
    log,
    plugins: createNoticeQueue(),
    window: { lost: ref(''), unwatched: ref(unwatched), error: ref(''), tasks },
    settings: { appearance: { state: { lost: ref('') } } },
    health: {
      items: ref(items),
      commands: {
        getIds: () => ids,
        run: (command, _target, _item, values) => {
          runs.push({ command, values })
          return Promise.resolve('')
        },
      },
      getCommandTitle: (command) => (command === 'app:settings.open' ? 'Open settings' : ''),
      getPluginName: (pluginId) => (pluginId === 'org.example.atlas' ? 'Atlas' : pluginId),
    },
    writeMessage: () => undefined,
  })
  const read = (): readonly BackgroundTask[] => {
    let list: readonly BackgroundTask[] = []
    host.watch((one) => (list = one)).dispose()
    return list
  }
  return { log, read, host, runs }
}

describe('the background tasks of the window', () => {
  it('list the work the window does', () => {
    expect(
      setup()
        .read()
        .map((one) => one.id),
    ).toStrictEqual(['reading'])
  })

  it('list the failures of the window and of its log after the work', () => {
    const { log, read } = setup('the vault folder is not watched')
    log.getWriter('command')('The tabs did not open', 'error')

    expect(read().map((one) => one.state)).toStrictEqual(['running', 'failed', 'failed'])
  })

  it('list an error of a plugin as a failed task with its fix, and leave a warning out', () => {
    const listed = setup('', [CONNECTION, QUOTA]).read()

    expect(listed.map((one) => one.id)).toStrictEqual([
      'reading',
      'health:org.example.atlas:connection',
    ])
    expect(listed[1]).toMatchObject({
      label: 'Atlas',
      state: 'failed',
      error: 'Choose a connection',
      actionLabel: 'Open settings',
    })
  })

  it('run the fix of an error with its values', async () => {
    const { host, runs } = setup('', [CONNECTION])

    host.runAction('health:org.example.atlas:connection')
    await nextTick()

    expect(runs).toStrictEqual([
      {
        command: 'app:settings.open',
        values: { plugin: 'org.example.atlas', setting: 'source' },
      },
    ])
  })

  it('list an error without a fix where the window has no handler for its command', async () => {
    const { read, host, runs } = setup('', [CONNECTION], [])

    expect(read()[1]?.actionLabel).toBe('')
    host.runAction('health:org.example.atlas:connection')
    await nextTick()
    expect(runs).toStrictEqual([])
  })

  it('reports cancel and action errors through writeMessage', async () => {
    const writeMessage = vi.fn()
    const host = createWindowBackgroundTasks({
      cancelTask: () => Promise.reject(new Error('cannot cancel')),
      log: createMessageLog(),
      plugins: createNoticeQueue(),
      window: { lost: ref(''), unwatched: ref(''), error: ref(''), tasks: ref([READING]) },
      settings: { appearance: { state: { lost: ref('') } } },
      health: {
        items: ref([CONNECTION]),
        commands: {
          getIds: () => ['app:settings.open'],
          run: () => Promise.reject(new Error('action failed')),
        },
        getCommandTitle: () => 'Open settings',
        getPluginName: () => 'Atlas',
      },
      writeMessage,
    })

    await host.cancel('reading')
    host.runAction('health:org.example.atlas:connection')
    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith(
        'The task was not cancelled: cannot cancel',
        'error',
      ),
    )
    await vi.waitFor(() => expect(writeMessage).toHaveBeenCalledWith('action failed', 'error'))
  })

  it('builds full health tasks and matches the target health item by id', async () => {
    const secondItem: PluginHealthItem = {
      pluginId: 'org.example.other',
      key: 'auth',
      severity: 'error',
      text: 'Sign in',
      detail: '',
      action: { command: 'app:settings.open', values: { plugin: 'other' }, title: 'Fix auth' },
    }
    const runs: string[] = []
    const items: readonly PluginHealthItem[] = [CONNECTION, secondItem]
    const deps = {
      commands: {
        getIds: () => ['app:settings.open'],
        run: (
          command: string,
          _target?: object,
          _item?: string,
          values?: Readonly<Record<string, string>>,
        ) => {
          runs.push(`${command}:${values?.plugin ?? ''}`)
          return Promise.resolve('')
        },
      },
      getCommandTitle: () => 'Open settings',
      getPluginName: (id: string) => (id === 'org.example.atlas' ? 'Atlas' : 'Other'),
    }
    const healthTasks = getHealthTasks(items, deps)
    expect(healthTasks[0]).toStrictEqual({
      id: 'health:org.example.atlas:connection',
      label: 'Atlas',
      about: 'The atlas has no connection to ask.',
      progress: { completedCount: 0, total: 0, counting: 'items' },
      error: 'Choose a connection',
      action: { label: 'Open settings' },
      source: { isUserStarted: false, pluginId: 'org.example.atlas', pluginName: 'Atlas' },
      isCancellable: false,
      kind: 'work',
    })

    await runHealthAction('health:org.example.other:auth', items, deps)
    await runHealthAction('health:missing', items, deps)
    expect(runs).toStrictEqual(['app:settings.open:other'])
  })
})
