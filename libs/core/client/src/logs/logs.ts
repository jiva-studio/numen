/**
 * Structured application log entries, as a window watches and records them.
 */
import type { Client } from '@connectrpc/connect'
import { LogLevel, type LogEntry as LogEntryMessage, type LogService } from '@numen/protocol'

/** Severity level of a structured log entry. */
export type LogLevelName = 'debug' | 'info' | 'warning' | 'error'

/**
 * One structured log entry received from the core. `time` is milliseconds since the Unix epoch,
 * and `properties` holds each key's JSON-encoded value string.
 */
export interface LogEntry {
  readonly time: number
  readonly level: LogLevelName
  readonly source: string
  readonly message: string
  readonly properties: Readonly<Record<string, string>>
}

/** A JSON-serializable property value of a structured log entry. */
export type LogPropertyValue =
  | string
  | number
  | boolean
  | null
  | readonly LogPropertyValue[]
  | { readonly [key: string]: LogPropertyValue }

/** An entry reported by a window or a plugin frontend. */
export interface LogWrite {
  readonly level: LogLevelName
  readonly source?: string | undefined
  readonly message: string
  readonly properties?: Readonly<Record<string, LogPropertyValue>> | undefined
}

/** The calls of LogService a window makes. */
export type LogRpc = Pick<
  Client<typeof LogService>,
  'getLogLevel' | 'setLogLevel' | 'watchLogs' | 'writeLog'
>

/** The structured logs of the installation, as a window watches and writes them. */
export interface LogClient {
  getLogLevel(): Promise<LogLevelName>
  setLogLevel(level: LogLevelName): Promise<void>
  /** Streams batches of recent and incoming log entries until the signal aborts. */
  watchLogs(signal: AbortSignal): AsyncGenerator<readonly LogEntry[]>
  /** Appends one structured log entry to the core log. */
  writeLog(entry: LogWrite): Promise<void>
}

const WRITE_LEVEL: Readonly<Record<LogLevelName, LogLevel>> = {
  debug: LogLevel.DEBUG,
  info: LogLevel.INFO,
  warning: LogLevel.WARNING,
  error: LogLevel.ERROR,
}

const READ_LEVEL: Readonly<Partial<Record<LogLevel, LogLevelName>>> = {
  [LogLevel.DEBUG]: 'debug',
  [LogLevel.INFO]: 'info',
  [LogLevel.WARNING]: 'warning',
  [LogLevel.ERROR]: 'error',
}

const readEntry = (one: LogEntryMessage): LogEntry => ({
  time: Number(one.timeUnixMs),
  level: READ_LEVEL[one.level] ?? 'info',
  source: one.source,
  message: one.message,
  properties: { ...one.propertiesJson },
})

const encodeProperties = (
  input: Readonly<Record<string, LogPropertyValue>> | undefined,
): Record<string, string> => {
  if (!input) return {}
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(input)) {
    if (!key) continue
    try {
      out[key] = JSON.stringify(value)
    } catch {
      out[key] = JSON.stringify(String(value))
    }
  }
  return out
}

/** Creates the log client over LogService. */
export function createLogClient(rpc: LogRpc): LogClient {
  return {
    getLogLevel: async () => READ_LEVEL[(await rpc.getLogLevel({})).level] ?? 'info',
    setLogLevel: async (level) => {
      await rpc.setLogLevel({ level: WRITE_LEVEL[level] })
    },
    watchLogs: async function* (signal) {
      for await (const update of rpc.watchLogs({}, { signal })) {
        yield update.entries.map(readEntry)
      }
    },
    writeLog: async (entry) => {
      await rpc.writeLog({
        level: WRITE_LEVEL[entry.level],
        source: entry.source ?? 'window',
        message: entry.message,
        propertiesJson: encodeProperties(entry.properties),
      })
    },
  }
}
