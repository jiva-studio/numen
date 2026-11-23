/** Structured application logs, in the windows' terms. */
import { describe, expect, it, vi } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { LogLevel, WatchLogsResponseSchema } from '@numen/protocol'
import { createLogRpc } from '../clients'
import { createCoreTransport, type CoreAnswers } from '../testing'
import { createLogClient, type LogRpc } from './logs'

const createLogs = (answers: CoreAnswers) =>
  createLogClient(createLogRpc(createCoreTransport(answers)))

const readFirst = async <T>(stream: AsyncIterable<T>): Promise<T | undefined> => {
  for await (const one of stream) return one
  return undefined
}

describe('structured application logs', () => {
  it('reads and writes the application log level', async () => {
    const getLogLevel = vi.fn(async () => ({ level: LogLevel.WARNING }))
    const setLogLevel = vi.fn(async () => ({}))
    const logs = createLogClient({
      getLogLevel: getLogLevel as never,
      setLogLevel: setLogLevel as never,
      watchLogs: vi.fn(),
      writeLog: vi.fn(),
    })

    await expect(logs.getLogLevel()).resolves.toBe('warning')
    await logs.setLogLevel('debug')

    expect(getLogLevel).toHaveBeenCalledWith({})
    expect(setLogLevel).toHaveBeenCalledWith({ level: LogLevel.DEBUG })
  })

  it('streams batches of log entries with their timestamp, severity level, source, message and JSON properties', async () => {
    const logs = createLogs({
      logService: {
        watchLogs: async function* () {
          yield {
            entries: [
              {
                timeUnixMs: 1_700_000_000_000n,
                level: LogLevel.DEBUG,
                source: 'sql',
                message: 'SQL query',
                propertiesJson: {
                  pool: '"read"',
                  sql: '"SELECT id FROM notes WHERE path = ?"',
                  durationMs: '0.42',
                },
              },
              {
                timeUnixMs: 1_700_000_000_500n,
                level: LogLevel.ERROR,
                source: 'rpc',
                message: 'NoteService/ReadNote',
                propertiesJson: {
                  error: '"not found"',
                },
              },
            ],
          }
        },
      },
    })

    const batch = await readFirst(logs.watchLogs(new AbortController().signal))

    expect(batch).toStrictEqual([
      {
        time: 1_700_000_000_000,
        level: 'debug',
        source: 'sql',
        message: 'SQL query',
        properties: {
          pool: '"read"',
          sql: '"SELECT id FROM notes WHERE path = ?"',
          durationMs: '0.42',
        },
      },
      {
        time: 1_700_000_000_500,
        level: 'error',
        source: 'rpc',
        message: 'NoteService/ReadNote',
        properties: {
          error: '"not found"',
        },
      },
    ])
  })

  it('writes a log entry with JSON-serialized properties and defaults source to window', async () => {
    const writeLog = vi.fn(async () => ({}))
    const logs = createLogs({
      logService: {
        writeLog: writeLog as never,
      },
    })

    await logs.writeLog({
      level: 'warning',
      message: 'slow frame',
      properties: { durationMs: 48, frame: { tab: 'notes' } },
    })

    expect(writeLog).toHaveBeenCalledOnce()
    const request = (
      writeLog.mock.calls[0] as unknown as [
        {
          level: LogLevel
          source: string
          message: string
          propertiesJson: Record<string, string>
        },
      ]
    )[0]
    expect(request.level).toBe(LogLevel.WARNING)
    expect(request.source).toBe('window')
    expect(request.message).toBe('slow frame')
    expect(request.propertiesJson).toStrictEqual({
      durationMs: '48',
      frame: '{"tab":"notes"}',
    })
  })

  it('forwards the abort signal to watchLogs', async () => {
    const watchLogs = vi.fn(async function* () {
      yield create(WatchLogsResponseSchema, {})
    })
    const rpc: LogRpc = {
      getLogLevel: vi.fn(),
      setLogLevel: vi.fn(),
      watchLogs,
      writeLog: vi.fn(),
    }
    const signal = new AbortController().signal

    await readFirst(createLogClient(rpc).watchLogs(signal))

    expect(watchLogs).toHaveBeenCalledWith({}, { signal })
  })
})
