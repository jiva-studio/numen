import type { LogEntry, LogLevel } from '@numen/plugin-sdk'

export type { LogEntry }

/** The level filter selected in the top bar: `'all'` or one of the four severity levels. */
export type LevelFilter = 'all' | LogLevel

/** One formatted property of an expanded log entry. */
export interface FormattedProperty {
  readonly key: string
  readonly kind: 'sql' | 'json' | 'scalar'
  readonly display: string
}

/** One log entry with its stable row identifier and precomputed property keys. */
export interface LogRow {
  readonly id: number
  readonly entry: LogEntry
  readonly timeText: string
  readonly keys: readonly string[]
}
