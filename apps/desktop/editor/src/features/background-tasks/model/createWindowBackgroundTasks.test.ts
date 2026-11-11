import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { BackgroundTask } from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import type { Task } from '@/entities/window'
import { createMessageLog } from '@/shared/model/notices/messages'
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

const setup = (unwatched = '') => {
  const log = createMessageLog()
  const tasks = ref<readonly Task[]>([READING])
  const host = createWindowBackgroundTasks({
    log,
    plugins: createNoticeQueue(),
    window: { lost: ref(''), unwatched: ref(unwatched), error: ref(''), tasks },
    settings: { appearance: { state: { lost: ref('') } } },
    writeMessage: () => undefined,
    openSettings: () => undefined,
  })
  const read = (): readonly BackgroundTask[] => {
    let list: readonly BackgroundTask[] = []
    host.watch((one) => (list = one)).dispose()
    return list
  }
  return { log, read }
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
})
