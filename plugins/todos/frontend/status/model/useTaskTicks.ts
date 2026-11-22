import { ref } from 'vue'
import { WORDS as words } from '../../words'
import type { Task, TaskWriter } from '../../types'

interface TaskTicks {
  /** A recurring task is not ticked: a tick would move its due date, which the backend does not do yet. */
  isRecurring(task: Task): boolean
  isTickable(task: Task): boolean
  /** Ticks the task, shows what the backend refused, then reads the lists again. */
  tick(task: Task): Promise<void>
}

const getKey = (task: Task): string => `${task.path}:${task.line}`

export function useTaskTicks(
  api: TaskWriter,
  getToday: () => string,
  reload: () => Promise<void>,
  showError: (message: string) => void,
): TaskTicks {
  const pending = ref<ReadonlySet<string>>(new Set())

  const isRecurring = (task: Task): boolean =>
    task.recurrence !== '' || task.issues.includes('invalidRecurrence')

  const isTickable = (task: Task): boolean => !isRecurring(task) && !pending.value.has(getKey(task))

  const tick = async (task: Task): Promise<void> => {
    if (!isTickable(task)) return
    const key = getKey(task)
    pending.value = new Set([...pending.value, key])
    try {
      const result = await api.setTaskDone({
        path: task.path,
        line: task.line,
        raw: task.raw,
        isDone: true,
        today: getToday(),
      })
      if (!result.ok) showError(words.tickFailed)
      else if (result.value.error !== '') showError(words.changedNote)
      await reload()
    } finally {
      pending.value = new Set([...pending.value].filter((one) => one !== key))
    }
  }

  return { isRecurring, isTickable, tick }
}
