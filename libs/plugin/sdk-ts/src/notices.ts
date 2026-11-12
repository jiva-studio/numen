// Severity is how a notice or a message reads: information, a warning, or an error.
export type Severity = 'info' | 'warning' | 'error'

// NoticeTask is one piece of work a plugin does behind the window, drawn while it runs.
export interface NoticeTask {
  readonly id: string
  readonly label: string
  readonly about?: string
  readonly error?: string
  readonly isUserStarted?: boolean
  readonly done?: number
  readonly total?: number
}

// NoticeHost is the corner of the window a plugin's notices are drawn in, one list for the window.
export interface NoticeHost {
  // A notice at a severity, info by default. An info notice is removed once read; any other stays until the
  // user dismisses it.
  show(text: string, severity?: Severity): void
  // The work the plugin is doing, replacing the list it gave last. An empty list takes it away.
  setTasks(tasks: readonly NoticeTask[]): void
}
