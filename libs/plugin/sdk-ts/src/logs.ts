import type { Disposable, JsonValue } from './types'

// LogLevel is the severity of a structured log entry.
export type LogLevel = 'debug' | 'info' | 'warning' | 'error'

// LogEntry is one structured log entry received from the core. `time` is milliseconds since the Unix
// epoch, and `properties` holds each key's JSON-encoded value string.
export interface LogEntry {
  readonly time: number
  readonly level: LogLevel
  readonly source: string
  readonly message: string
  readonly properties: Readonly<Record<string, string>>
}

// LogWrite is one structured log entry reported by a plugin frontend.
export interface LogWrite {
  readonly level: LogLevel
  readonly message: string
  readonly properties?: Readonly<Record<string, JsonValue>>
}

// LogHost streams the structured log entries of the installation (`logs:read`) and records entries
// from the plugin's frontend.
export interface LogHost {
  // Calls the handler with the recent log entries at once and with each incoming batch as entries
  // arrive, until the returned disposable is disposed. Requires `logs:read`.
  watch(handler: (entries: readonly LogEntry[]) => void): Disposable
  // Appends one structured log entry under the plugin's id.
  write(entry: LogWrite): Promise<void>
}
