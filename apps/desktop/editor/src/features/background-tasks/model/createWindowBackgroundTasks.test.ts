import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import type { BackgroundTask } from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import type { PluginHealthItem } from '@/entities/plugin'
import type { Task } from '@/entities/window'
import { createMessageLog } from '@/shared/model/notices/messages'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createWindowBackgroundTasks } from './createWindowBackgroundTasks'

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
    cancelTask: createAnsweredApi({}).tasks.cancelTask,
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
})
