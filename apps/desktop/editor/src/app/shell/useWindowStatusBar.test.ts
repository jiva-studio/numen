import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { useWindowStatusBar } from './useWindowStatusBar'
import type { Task } from '@/shared/notices/task'
import { pick } from '@/testing/pick'

const base: Task = {
  id: 't',
  label: 'Reading',
  about: '',
  completedCount: 1,
  total: 4,
  counting: 'items',
  error: '',
  isUserStarted: false,
  pluginId: '',
  pluginName: '',
  isCancellable: true,
}

const first = (task: Task) => pick(useWindowStatusBar({ tasks: ref([task]) }).tasks.value, 0)

describe('useWindowStatusBar', () => {
  it('maps empty tasks to empty status bar tasks', () => {
    const tasks = ref<readonly Task[]>([])
    const statusBar = useWindowStatusBar({ tasks })

    expect(statusBar.tasks.value).toEqual([])
  })

  it('maps a running task to its progress percentage', () => {
    const task = first({ ...base, completedCount: 50, total: 100 })

    expect(task).toMatchObject({ id: 't', label: 'Reading', progress: 50, state: 'running' })
  })

  it('handles failed tasks and allows dismissing them', () => {
    const tasks = ref<readonly Task[]>([{ ...base, id: 'failed-task', error: 'File is corrupted' }])

    const statusBar = useWindowStatusBar({ tasks })

    expect(statusBar.tasks.value).toHaveLength(1)
    expect(statusBar.tasks.value[0]?.state).toBe('failed')
    expect(statusBar.tasks.value[0]?.error).toBe('File is corrupted')

    statusBar.onDismissTask('failed-task')
    expect(statusBar.tasks.value).toHaveLength(0)
  })
})

describe('useWindowStatusBar detail and unit', () => {
  it('carries what the task is about, and nothing where it is about nothing', () => {
    expect(first({ ...base, about: 'doc.pdf' }).about).toBe('doc.pdf')
    expect(first(base).about).toBeUndefined()
  })

  it('names what a task counts, and none where it counts things', () => {
    expect(first({ ...base, counting: 'seconds' }).counting).toBe('seconds')
    expect(first(base).counting).toBeUndefined()
  })

  it('lets a running task be cancelled and a failed one not', () => {
    expect(first(base).canCancel).toBe(true)
    expect(first({ ...base, error: 'broken' }).canCancel).toBe(false)
  })

  it('carries the counts of a task that has a total, and none of one that has not', () => {
    expect(first(base)).toMatchObject({ done: 1, total: 4, progress: 25 })
    const none = first({ ...base, total: 0 })
    expect(none.progress).toBeUndefined()
    expect(none.total).toBeUndefined()
  })
})

describe('useWindowStatusBar handlers', () => {
  it('passes cancel and retry to the callbacks given, and does nothing without them', async () => {
    const cancelTask = vi.fn(() => Promise.resolve())
    const onRetry = vi.fn()
    const bar = useWindowStatusBar({ tasks: ref([base]), cancelTask, onRetry })
    await bar.onCancelTask('t')
    bar.onRetryTask('t')
    expect(cancelTask).toHaveBeenCalledWith('t')
    expect(onRetry).toHaveBeenCalledWith('t')

    const barWithoutCallbacks = useWindowStatusBar({ tasks: ref([base]) })
    await expect(barWithoutCallbacks.onCancelTask('t')).resolves.toBeUndefined()
    expect(barWithoutCallbacks.tasks.value[0]?.state).toBe('running')
    expect(() => barWithoutCallbacks.onRetryTask('t')).not.toThrow()
  })

  it('shows a cancelled task as cancelling until the core takes it out of the list', async () => {
    const tasks = ref<readonly Task[]>([base])
    const cancelTask = vi.fn(() => Promise.resolve())
    const bar = useWindowStatusBar({ tasks, cancelTask })

    await bar.onCancelTask('t')
    expect(bar.tasks.value[0]?.state).toBe('cancelling')

    await bar.onCancelTask('t')
    expect(cancelTask).toHaveBeenCalledTimes(1)

    tasks.value = [{ ...base, completedCount: 2 }]
    expect(bar.tasks.value[0]?.state).toBe('cancelling')

    tasks.value = []
    await nextTick()
    tasks.value = [base]
    expect(bar.tasks.value[0]?.state).toBe('running')
  })

  it('shows the task running again and reports the error when the core does not cancel it', async () => {
    const error = new Error('no cancellable task has that id')
    const reportCancelError = vi.fn()
    const bar = useWindowStatusBar({
      tasks: ref([base]),
      cancelTask: () => Promise.reject(error),
      reportCancelError,
    })

    await bar.onCancelTask('t')

    expect(bar.tasks.value[0]?.state).toBe('running')
    expect(reportCancelError).toHaveBeenCalledWith(error)
  })

  it('offers no cancel for a task the core cannot cancel', () => {
    expect(first({ ...base, isCancellable: false }).canCancel).toBe(false)
  })

  it('shows the action and passes the task plugin id to onAction', () => {
    const onAction = vi.fn()
    const task: Task = {
      ...base,
      id: 'task-err',
      error: 'OAuth session expired',
      pluginId: 'md.example.agent',
    }
    const statusBar = useWindowStatusBar({
      tasks: ref([task]),
      resolveAction: (one) => ({ label: 'Configure in settings', pluginId: one.pluginId }),
      onAction,
    })

    const [item] = statusBar.tasks.value
    expect(item?.actionLabel).toBe('Configure in settings')
    expect(item?.actionId).toBe('md.example.agent')

    statusBar.onActionTask('task-err')
    expect(onAction).toHaveBeenCalledWith('md.example.agent')
  })

  it('does not call onAction for a core task', () => {
    const onAction = vi.fn()
    const statusBar = useWindowStatusBar({ tasks: ref([base]), onAction })

    statusBar.onActionTask('t')

    expect(onAction).not.toHaveBeenCalled()
  })

  it('lists the same ids in the same order through every report of a task', () => {
    const report = (step: number): readonly Task[] => [
      { ...base, id: 'vectors', label: 'Indexing', about: `file ${step}`, completedCount: step },
      { ...base, id: 'texts', label: 'Reading', pluginName: 'Reader' },
      { ...base, id: 'lost', error: 'The vault is gone' },
    ]
    const tasks = ref<readonly Task[]>(report(1))
    const statusBar = useWindowStatusBar({ tasks })
    const ids = () => statusBar.tasks.value.map((one) => one.id)
    const first = ids()

    for (let step = 2; step <= 20; step++) {
      tasks.value = report(step)
      expect(ids()).toEqual(first)
    }
    expect(first).toEqual(['vectors', 'texts', 'lost'])
    expect(pick(statusBar.tasks.value, 0).about).toBe('file 20')
  })
})
