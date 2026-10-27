// Severity is how a notice or a message reads: information, a warning, or an error.
export type Severity = 'info' | 'warning' | 'error'

// NoticeTask is one piece of work a plugin does behind the window, drawn while it runs.
export interface NoticeTask {
  readonly id: string
  // What is being done, in the words it is shown by.
  readonly label: string
  // What it is being done to.
  readonly about?: string
  // Why it stopped; empty while it runs.
  readonly error?: string
  // A person asked for it and is waiting to be told it began.
  readonly isUserStarted?: boolean
  // How far it has got, where there is a total to count against.
  readonly done?: number
  readonly total?: number
}

// NoticeHost is the corner of the window a plugin's notices are drawn in, one list for the window.
export interface NoticeHost {
  // A notice at a severity, info by default. An info notice goes once read; any other stands until the
  // person dismisses it.
  show(text: string, severity?: Severity): void
  // The work the plugin is doing, replacing the list it gave last. An empty list takes it away.
  setTasks(tasks: readonly NoticeTask[]): void
}
