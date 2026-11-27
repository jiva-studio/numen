// Stryker disable StringLiteral: interface copy
/** Every text the plugin shows. */
export const WORDS = {
  groupOverdue: 'Overdue',
  groupToday: 'Today',
  groupNextSevenDays: 'Next 7 days',
  groupNoDate: 'No date',
  chipLabel: 'Tasks due',
  changedNote: 'The note changed. Open it to tick the task.',
  recurringNotTickable: 'Recurring tasks cannot be ticked here yet',
  taskBox: 'Task',
  taskNotRead: 'The note could not be read again; it opens without the task line.',
  noteNotOpened: 'The note could not be opened.',
  tickFailed: 'The task could not be ticked',
} as const

/** The label of the checkbox that marks the task as done. */
export const taskDoneLabel = (text: string): string => `Done: ${text}`
