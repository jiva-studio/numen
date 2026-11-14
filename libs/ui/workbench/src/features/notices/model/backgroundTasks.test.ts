import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref, type Ref } from 'vue'
import type { BackgroundTask, WorkTask as Task } from '../lib/backgroundTask'
import { createBackgroundTaskHost, type BackgroundTaskHostDeps } from './backgroundTasks'

const pick = <T>(list: readonly T[], pos: number): T => {
  const found = list[pos]
  if (found === undefined) throw new Error(`nothing at ${pos} of ${list.length}`)
  return found
}

const base: Task = {
  id: 't',
  label: 'Reading',
  about: '',
  progress: { completedCount: 1, total: 4, counting: 'items' },
  error: '',
  source: { isUserStarted: false, pluginId: '', pluginName: '' },
  isCancellable: true,
  kind: 'work',
}

const createDeps = (
  tasks: readonly Task[],
  extra: Partial<BackgroundTaskHostDeps> = {},
): BackgroundTaskHostDeps & { tasks: Ref<readonly Task[]> } => ({
  tasks: ref<readonly Task[]>(tasks),
  cancelTask: vi.fn(() => Promise.resolve()),
  reportCancelError: vi.fn(),
  runAction: vi.fn(),
  ...extra,
})

const readTasks = (
  host: ReturnType<typeof createBackgroundTaskHost>,
): readonly BackgroundTask[] => {
  let list: readonly BackgroundTask[] = []
  host.watch((tasks) => (list = tasks)).dispose()
  return list
}

const first = (task: Task): BackgroundTask =>
  pick(readTasks(createBackgroundTaskHost(createDeps([task]))), 0)

describe('the tasks of the window as background tasks', () => {
  it('is an empty list while the window has no task', () => {
    expect(readTasks(createBackgroundTaskHost(createDeps([])))).toStrictEqual([])
  })

  it('carries a running task with its counts', () => {
    expect(first(base)).toStrictEqual({
      id: 't',
      label: 'Reading',
      about: '',
      state: 'running',
      progress: { completedCount: 1, total: 4, counting: '' },
      error: '',
      allowed: { cancel: true, retry: false },
      actionLabel: '',
    })
  })

  it('counts nothing for a task without a total', () => {
    expect(
      first({ ...base, progress: { ...base.progress, completedCount: 3, total: 0 } }),
    ).toMatchObject({
      progress: { completedCount: 0, total: 0 },
    })
  })

  it('carries what the task is about and what it counts', () => {
    expect(
      first({ ...base, about: 'doc.pdf', progress: { ...base.progress, counting: 'seconds' } }),
    ).toMatchObject({
      about: 'doc.pdf',
      progress: { counting: 'seconds' },
    })
  })

  it('shows a task with an error as failed, which cannot be cancelled', () => {
    expect(first({ ...base, error: 'File is corrupted' })).toMatchObject({
      state: 'failed',
      error: 'File is corrupted',
      allowed: { cancel: false },
    })
  })

  it('shows a notice of the core as a notice, neither running nor failed', () => {
    expect(first({ ...base, kind: 'notice', about: 'a key is ignored' })).toMatchObject({
      state: 'notice',
      error: '',
    })
  })

  it('offers no cancel for a task the core cannot cancel', () => {
    expect(first({ ...base, isCancellable: false }).allowed.cancel).toBe(false)
  })

  it('offers the action of a task only while it is failed', () => {
    const action = { label: 'Open settings' }
    const host = createBackgroundTaskHost(
      createDeps([
        { ...base, action },
        { ...base, id: 'u', error: 'Choose a connection', action },
      ]),
    )

    const [running, failed] = readTasks(host)

    expect(running).toMatchObject({ actionLabel: '' })
    expect(failed).toMatchObject({ actionLabel: 'Open settings' })
  })

  it('calls a watcher again when the tasks of the window change, and not after dispose', async () => {
    const deps = createDeps([base])
    const host = createBackgroundTaskHost(deps)
    const lists: (readonly BackgroundTask[])[] = []
    const watching = host.watch((tasks) => lists.push(tasks))

    deps.tasks.value = [base, { ...base, id: 'u' }]
    await nextTick()
    watching.dispose()
    deps.tasks.value = []
    await nextTick()

    expect(lists.map((one) => one.map((task) => task.id))).toStrictEqual([['t'], ['t', 'u']])
  })

  it('lists the same ids in the same order through every report of a task', async () => {
    const report = (step: number): readonly Task[] => [
      {
        ...base,
        id: 'vectors',
        label: 'Indexing',
        about: `file ${step}`,
        progress: { ...base.progress, completedCount: step },
      },
      { ...base, id: 'texts', label: 'Reading', source: { ...base.source, pluginName: 'Reader' } },
      { ...base, id: 'lost', error: 'The vault is gone' },
    ]
    const deps = createDeps(report(1))
    const host = createBackgroundTaskHost(deps)

    for (let step = 2; step <= 20; step++) {
      deps.tasks.value = report(step)
      await nextTick()
      expect(readTasks(host).map((one) => one.id)).toStrictEqual(['vectors', 'texts', 'lost'])
    }
    expect(pick(readTasks(host), 0).about).toBe('file 20')
  })
})

describe('the actions on a background task', () => {
  it('takes a dismissed task out of the list', async () => {
    const deps = createDeps([base, { ...base, id: 'failed', error: 'broken' }])
    const host = createBackgroundTaskHost(deps)

    host.dismiss('failed')
    await nextTick()

    expect(readTasks(host).map((one) => one.id)).toStrictEqual(['t'])
  })

  it('asks the core to cancel a task, and shows it as cancelling until the core takes it out', async () => {
    const deps = createDeps([base])
    const host = createBackgroundTaskHost(deps)

    await host.cancel('t')
    await nextTick()
    expect(deps.cancelTask).toHaveBeenCalledWith('t')
    expect(pick(readTasks(host), 0).state).toBe('cancelling')

    await host.cancel('t')
    expect(deps.cancelTask).toHaveBeenCalledTimes(1)

    deps.tasks.value = [{ ...base, progress: { ...base.progress, completedCount: 2 } }]
    await nextTick()
    expect(pick(readTasks(host), 0).state).toBe('cancelling')

    deps.tasks.value = []
    await nextTick()
    deps.tasks.value = [base]
    await nextTick()
    expect(pick(readTasks(host), 0).state).toBe('running')
  })

  it('shows the task running again, reports the error and rejects when the core does not cancel it', async () => {
    const error = new Error('no cancellable task has that id')
    const deps = createDeps([base], { cancelTask: () => Promise.reject(error) })
    const host = createBackgroundTaskHost(deps)

    await expect(host.cancel('t')).rejects.toBe(error)
    await nextTick()

    expect(pick(readTasks(host), 0).state).toBe('running')
    expect(deps.reportCancelError).toHaveBeenCalledWith(error)
  })

  it('runs the action of the requested task', () => {
    const deps = createDeps([])
    const host = createBackgroundTaskHost(deps)

    host.runAction('health:md.example.atlas:connection')

    expect(deps.runAction).toHaveBeenCalledWith('health:md.example.atlas:connection')
  })
})
