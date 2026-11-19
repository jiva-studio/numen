import type { LogEntry, LogLevel } from '@numen/plugin-sdk'

export type { LogEntry }

/** The level filter selected in the top bar: `'all'` or one of the four severity levels. */
export type LevelFilter = 'all' | LogLevel

/** One syntax-highlighted token inside a formatted SQL or JSON property. */
export interface SyntaxToken {
  readonly kind:
    'plain' | 'keyword' | 'key' | 'string' | 'number' | 'type' | 'punctuation' | 'comment'
  readonly text: string
}

/** One formatted property of an expanded log entry. */
export interface FormattedProperty {
  readonly key: string
  readonly kind: 'sql' | 'json' | 'scalar'
  readonly display: string
  readonly tokens: readonly SyntaxToken[]
}

/** One log entry with its stable row identifier and formatted timestamp. */
export interface LogRow {
  readonly id: number
  readonly entry: LogEntry
  readonly timeText: string
}
