import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useWindowStatusBar } from './useWindowStatusBar'
import type { Task } from '@/shared/notices/task'
import { WORDS as words } from '@/shared/words'
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

describe('useWindowStatusBar groups', () => {
  it('groups tasks by the name of the plugin that reported them', () => {
    const tasks = ref<readonly Task[]>([
      { ...base, id: 'a', label: 'Step one', pluginId: 'md.example.first', pluginName: 'First' },
      { ...base, id: 'b', label: 'Step two', pluginId: 'md.example.second', pluginName: 'Second' },
      { ...base, id: 'c', label: 'Step three', pluginId: 'md.example.first', pluginName: 'First' },
    ])

    const groups = useWindowStatusBar({ tasks }).tasks.value.map((one) => one.group)

    expect(groups).toEqual(['First', 'Second', 'First'])
  })

  it('groups a plugin task without a name under the plugin id', () => {
    expect(first({ ...base, pluginId: 'md.example.first' }).group).toBe('md.example.first')
  })

  it('groups the core work under one group whatever its label says', () => {
    const tasks = ref<readonly Task[]>([
      { ...base, id: 'a', label: 'Transcribing speech' },
      { ...base, id: 'b', label: 'Recognising a scan' },
    ])

    const groups = useWindowStatusBar({ tasks }).tasks.value.map((one) => one.group)

    expect(groups).toEqual([words.coreTasks, words.coreTasks])
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
  it('passes cancel and retry to the callbacks given, and does nothing without them', () => {
    const onCancel = vi.fn()
    const onRetry = vi.fn()
    const bar = useWindowStatusBar({ tasks: ref([base]), onCancel, onRetry })
    bar.onCancelTask('t')
    bar.onRetryTask('t')
    expect(onCancel).toHaveBeenCalledWith('t')
    expect(onRetry).toHaveBeenCalledWith('t')

    const barWithoutCallbacks = useWindowStatusBar({ tasks: ref([base]) })
    expect(() => barWithoutCallbacks.onCancelTask('t')).not.toThrow()
    expect(() => barWithoutCallbacks.onRetryTask('t')).not.toThrow()
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
})
