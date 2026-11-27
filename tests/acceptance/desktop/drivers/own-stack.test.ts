import { afterEach, describe, expect, it, vi } from 'vitest'
import { OwnStack } from './own-stack.js'

const formatTraffic = (received: number): string => 'eth0: ' + received + ' 0 0 0 0 0 0 0 0'

const createGate = () => {
  let open!: () => void
  const opened = new Promise<void>((resolve) => {
    open = resolve
  })
  return { open, opened }
}

const waitForAbort = (signal: AbortSignal): Promise<string> =>
  new Promise((_resolve, reject) => {
    signal.addEventListener('abort', () => reject(new Error('the traffic probe was cancelled')), { once: true })
  })

afterEach(() => {
  vi.useRealTimers()
})

describe('stand recovery', () => {
  it('restarts the bucket only after the transfer was interrupted', async () => {
    const traffic = createGate()
    const kill = createGate()
    const killStarted = createGate()
    const commands: string[] = []
    const stack = new OwnStack('test', async (args, _stdin, options) => {
      if (args.at(-1) === '/proc/net/dev') {
        if (options?.signal === undefined) return formatTraffic(10)
        await traffic.opened
        return formatTraffic(30)
      }
      const command = args.join(' ')
      commands.push(command)
      if (command === 'kill bucket') {
        killStarted.open()
        await kill.opened
      }
      return ''
    })

    await stack.breakAfter('received', 20)
    const recovery = stack.startServer()
    expect(commands).toEqual([])
    traffic.open()
    await killStarted.opened
    expect(commands).toEqual(['kill bucket'])
    kill.open()
    await recovery
    expect(commands).toEqual(['kill bucket', 'start bucket sync', 'exec -T sync /numen-sync healthcheck'])
  })

  it('reports a timeout and leaves the stand stopped when the transfer never breaks', async () => {
    vi.useFakeTimers()
    const commands: string[] = []
    const stack = new OwnStack('test', async (args, _stdin, options) => {
      if (args.at(-1) === '/proc/net/dev') {
        return options?.signal === undefined ? formatTraffic(10) : waitForAbort(options.signal)
      }
      commands.push(args.join(' '))
      return ''
    })

    await stack.breakAfter('received', 20)
    const failure = expect(stack.startServer()).rejects.toThrow('the transfer did not break within 30 seconds')
    await vi.advanceTimersByTimeAsync(30_000)
    await failure
    expect(commands).toEqual([])
  })

  it('cancels a pending interruption during teardown and restores the stand', async () => {
    const commands: string[] = []
    let probe: AbortSignal | undefined
    const stack = new OwnStack('test', async (args, _stdin, options) => {
      if (args.at(-1) === '/proc/net/dev') {
        probe = options?.signal
        return probe === undefined ? formatTraffic(10) : waitForAbort(probe)
      }
      commands.push(args.join(' '))
      return ''
    })

    await stack.breakAfter('received', 20)
    await stack.resetServer()

    expect(probe?.aborted).toBe(true)
    expect(commands).toEqual(['start bucket sync', 'exec -T sync /numen-sync healthcheck'])
  })
})
