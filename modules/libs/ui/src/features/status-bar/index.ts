export { default as StatusBar } from './ui/StatusBar.vue'
export { default as TaskPopover } from './ui/TaskPopover.vue'
export { default as TaskGroup } from './ui/TaskGroup.vue'
export { default as TaskItem } from './ui/TaskItem.vue'
export { default as CircularProgress } from './ui/CircularProgress.vue'

export { calculateAggregate, formatEta, groupTasks } from './lib/aggregate'
export type { StatusBarTask, TaskState, TaskGroupData, StatusBarAggregate } from './model/types'
