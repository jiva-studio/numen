import { ref, type Ref } from 'vue'
import { WORDS } from '../../words'
import type { DueFilter, Filter, Task, TaskReader } from '../../types'

/** How many tasks of a group the popover shows. */
export const GROUP_LIMIT = 20

/** One list of the popover: the tasks of a due filter and how many there are in all. */
export interface TaskGroup {
  readonly key: DueFilter
  readonly label: string
  readonly tasks: readonly Task[]
  readonly total: number
}

const GROUPS: readonly { key: DueFilter; label: string }[] = [
  { key: 'overdue', label: WORDS.groupOverdue },
  { key: 'today', label: WORDS.groupToday },
  { key: 'none', label: WORDS.groupNoDate },
]

const openTasksDue = (due: DueFilter): Filter => ({
  status: ['open'],
  due,
  from: '',
  to: '',
  projects: [],
  contexts: [],
  priorities: [],
  path: '',
  text: '',
})

/** Returns the title of the note at a path: its file name without the extension. */
export function getNoteTitle(path: string): string {
  const name = path.slice(path.lastIndexOf('/') + 1)
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(0, dot) : name
}

export interface TaskGroups {
  /** The groups that hold a task, in the order Overdue, Today, No date. */
  readonly groups: Readonly<Ref<readonly TaskGroup[]>>
  /** Reads the three lists at the day getToday gives now. */
  load(): Promise<void>
}

export function useTaskGroups(api: TaskReader, getToday: () => string): TaskGroups {
  const groups = ref<readonly TaskGroup[]>([])

  const load = async (): Promise<void> => {
    const today = getToday()
    const read = await Promise.all(
      GROUPS.map(async ({ key, label }): Promise<TaskGroup> => {
        const result = await api.listTasks(today, openTasksDue(key), GROUP_LIMIT)
        return result.ok
          ? { key, label, tasks: result.value.tasks, total: result.value.total }
          : { key, label, tasks: [], total: 0 }
      }),
    )
    groups.value = read.filter((group) => group.total > 0)
  }

  return { groups, load }
}
