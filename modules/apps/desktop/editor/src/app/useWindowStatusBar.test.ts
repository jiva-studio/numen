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
