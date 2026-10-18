import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useWindowStatusBar } from './useWindowStatusBar'
import type { Task } from '@/shared/notices/task'
import { pick } from '@/testing/pick'

describe('useWindowStatusBar', () => {
  it('maps empty tasks to empty status bar tasks', () => {
    const tasks = ref<readonly Task[]>([])
    const statusBar = useWindowStatusBar({ tasks })

    expect(statusBar.tasks.value).toEqual([])
  })

  it('maps running tasks with progress percentage and groups them', () => {
    const tasks = ref<readonly Task[]>([
      {
        id: 'task-1',
        label: 'Indexing book',
        about: 'War and Peace.epub',
        done: 50,
        total: 100,
        counting: 'things',
        error: '',
        isUserStarted: false,
      },
      {
        id: 'task-2',
        label: 'Transcribing speech',
        about: 'audio.mp3',
        done: 0,
        total: 0,
        counting: 'seconds',
        error: '',
        isUserStarted: false,
      },
    ])

    const statusBar = useWindowStatusBar({ tasks })

    expect(statusBar.tasks.value).toHaveLength(2)
    const task1 = pick(statusBar.tasks.value, 0)
    expect(task1.id).toBe('task-1')
    expect(task1.label).toBe('Indexing book')
    expect(task1.progress).toBe(50)
    expect(task1.state).toBe('running')
    expect(task1.group).toBe('Indexing')

    const task2 = pick(statusBar.tasks.value, 1)
    expect(task2.progress).toBeUndefined()
    expect(task2.state).toBe('running')
    expect(task2.group).toBe('Audio & Transcription')
  })

  it('handles failed tasks and allows dismissing them', () => {
    const tasks = ref<readonly Task[]>([
      {
        id: 'failed-task',
        label: 'Reading file',
        about: 'doc.pdf',
        done: 0,
        total: 10,
        counting: 'things',
        error: 'File is corrupted',
        isUserStarted: false,
      },
    ])

    const statusBar = useWindowStatusBar({ tasks })

    expect(statusBar.tasks.value).toHaveLength(1)
    expect(statusBar.tasks.value[0]?.state).toBe('failed')
    expect(statusBar.tasks.value[0]?.error).toBe('File is corrupted')

    // Dismissing the failed task removes it from the status bar display
    statusBar.onDismissTask('failed-task')
    expect(statusBar.tasks.value).toHaveLength(0)
  })
})

describe('useWindowStatusBar detail and unit', () => {
  const base: Task = {
    id: 't',
    label: 'Reading',
    about: '',
    done: 1,
    total: 4,
    counting: 'things',
    error: '',
    isUserStarted: false,
  }
  const first = (task: Task) => pick(useWindowStatusBar({ tasks: ref([task]) }).tasks.value, 0)

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
    expect(none.done).toBeUndefined()
    expect(none.total).toBeUndefined()
  })
})

describe('useWindowStatusBar groups and handlers', () => {
  const plain: Task = {
    id: 't',
    label: 'Sorting',
    about: 'shelf',
    done: 0,
    total: 0,
    counting: 'things',
    error: '',
    isUserStarted: false,
  }

  it('groups a task that matches no pattern under its label, or under a general name without one', () => {
    const bar = useWindowStatusBar({ tasks: ref([plain, { ...plain, id: 'u', label: '' }]) })
    expect(bar.tasks.value.map((one) => one.group)).toEqual(['Sorting', 'Background Tasks'])
  })

  it('hands cancel and retry to the callbacks given, and does nothing without them', () => {
    const onCancel = vi.fn()
    const onRetry = vi.fn()
    const bar = useWindowStatusBar({ tasks: ref([plain]), onCancel, onRetry })
    bar.onCancelTask('t')
    bar.onRetryTask('t')
    expect(onCancel).toHaveBeenCalledWith('t')
    expect(onRetry).toHaveBeenCalledWith('t')

    const bare = useWindowStatusBar({ tasks: ref([plain]) })
    expect(() => bare.onCancelTask('t')).not.toThrow()
    expect(() => bare.onRetryTask('t')).not.toThrow()
  })
})

describe('a proofreading the person has not allowed', () => {
  it('stands failed in the status bar, saying where to allow it', () => {
    const reason = 'AI access not allowed — allow it in Settings → AI'
    const tasks = ref<readonly Task[]>([
      {
        id: 'proofread-talks/one.mp3',
        label: 'Proofreading a transcript',
        about: 'talks/one.mp3',
        done: 0,
        total: 0,
        counting: 'things',
        error: reason,
        isUserStarted: true,
      },
    ])

    const [shown] = useWindowStatusBar({ tasks }).tasks.value

    expect(shown).toMatchObject({ state: 'failed', error: reason, canCancel: false })
  })
})
