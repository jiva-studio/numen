import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { BackgroundTask, WorkTask as Task } from '../lib/backgroundTask'
import { createBackgroundTaskHost } from './backgroundTasks'

const failed: Task = {
  id: 'copy:Talks/mill.md',
  label: 'Copying a video',
  about: 'Talks/mill.md',
  progress: { completedCount: 0, total: 0, counting: 'bytes' },
  error: 'the site hung up',
  source: { isUserStarted: true, pluginId: 'org.example.links', pluginName: 'Links' },
  isCancellable: false,
  kind: 'work',
}

describe('a dismissed failed task', () => {
  it('leaves the next run of the same work in the list', async () => {
    const tasks = ref<readonly Task[]>([failed])
    const host = createBackgroundTaskHost({
      tasks,
      cancelTask: vi.fn(() => Promise.resolve()),
      reportCancelError: vi.fn(),
      runAction: vi.fn(),
    })
    let list: readonly BackgroundTask[] = []
    host.watch((next) => (list = next))

    host.dismiss(failed.id)
    await nextTick()
    tasks.value = []
    await nextTick()
    tasks.value = [{ ...failed, error: '', isCancellable: true }]
    await nextTick()

    expect(list.map((one) => [one.id, one.state])).toStrictEqual([[failed.id, 'running']])
  })
})
