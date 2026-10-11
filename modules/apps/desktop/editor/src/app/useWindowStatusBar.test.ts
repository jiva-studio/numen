import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useWindowStatusBar } from './useWindowStatusBar'
import type { Task } from '@/shared/notices/task'

describe('useWindowStatusBar', () => {
  it('maps empty tasks to empty status bar tasks and idle aggregate', () => {
    const tasks = ref<readonly Task[]>([])
    const statusBar = useWindowStatusBar({ tasks })

    expect(statusBar.tasks.value).toEqual([])
    expect(statusBar.aggregate.value.isIdle).toBe(true)
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
        isAsked: false,
      },
      {
        id: 'task-2',
        label: 'Transcribing speech',
        about: 'audio.mp3',
        done: 0,
        total: 0,
        counting: 'seconds',
        error: '',
        isAsked: false,
      },
    ])

    const statusBar = useWindowStatusBar({ tasks })

    expect(statusBar.tasks.value).toHaveLength(2)
    const task1 = statusBar.tasks.value[0]!
    expect(task1.id).toBe('task-1')
    expect(task1.label).toBe('Indexing book')
    expect(task1.progress).toBe(50)
    expect(task1.state).toBe('running')
    expect(task1.group).toBe('Indexing')

    const task2 = statusBar.tasks.value[1]!
    expect(task2.progress).toBeUndefined()
    expect(task2.state).toBe('running')
    expect(task2.group).toBe('Audio & Transcription')

    expect(statusBar.aggregate.value.activeCount).toBe(2)
    expect(statusBar.aggregate.value.isIdle).toBe(false)
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
        isAsked: false,
      },
    ])

    const statusBar = useWindowStatusBar({ tasks })

    expect(statusBar.tasks.value).toHaveLength(1)
    expect(statusBar.tasks.value[0]?.state).toBe('failed')
    expect(statusBar.tasks.value[0]?.errorReason).toBe('File is corrupted')
    expect(statusBar.aggregate.value.failedCount).toBe(1)

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
    isAsked: false,
  }
  const first = (task: Task) => useWindowStatusBar({ tasks: ref([task]) }).tasks.value[0]!

  it('carries what the task is about as its detail, and nothing where it is about nothing', () => {
    expect(first({ ...base, about: 'doc.pdf' }).detail).toBe('doc.pdf')
    expect(first(base).detail).toBeUndefined()
  })

  it('names the unit a task counts, and none where it counts things', () => {
    expect(first({ ...base, counting: 'seconds' }).unit).toBe('seconds')
    expect(first(base).unit).toBeUndefined()
  })

  it('lets a running task be cancelled and a failed one not', () => {
    expect(first(base).canCancel).toBe(true)
    expect(first({ ...base, error: 'broken' }).canCancel).toBe(false)
  })

  it('carries the counts of a task that has a total, and none of one that has not', () => {
    expect(first(base)).toMatchObject({ currentCount: 1, totalCount: 4, progress: 25 })
    const none = first({ ...base, total: 0 })
    expect(none.currentCount).toBeUndefined()
    expect(none.totalCount).toBeUndefined()
  })
})
