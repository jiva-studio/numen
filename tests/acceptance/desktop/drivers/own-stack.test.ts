import { afterEach, describe, expect, it, vi } from 'vitest'
import { OwnStack } from './own-stack.js'

const formatTraffic = (received: number, sent = 0): string => 'eth0: ' + received + ' 0 0 0 0 0 0 0 ' + sent

const formatStoppedSample = (received: number, sent = 0, queued = 0): string =>
  [
    'State:\tT (stopped)',
    'State:\tT (stopped)',
    'queues',
    '  sl  local_address rem_address st tx_queue:rx_queue tr:tm->when retrnsmt   uid  timeout inode',
    '   0: 00000000:2256 00000000:0000 01 ' + queued.toString(16) + ':00000000',
    'traffic',
    formatTraffic(received, sent),
  ].join('\n')

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
      if (args[3] === 'sh') {
        commands.push('sample stopped bucket')
        return formatStoppedSample(32)
      }
      commands.push(command)
      if (args[0] === 'ps') return 'bucket'
      if (command === 'kill bucket') {
        killStarted.open()
        await kill.opened
      }
      return ''
    })

    stack.bucketTransfer.breakAfter('file', 'received', 20)
    await stack.bucketTransfer.recordFile('file', 100)
    const recovery = stack.startServer()
    expect(commands).toEqual([])
    traffic.open()
    await killStarted.opened
    const interruption = ['kill -s SIGSTOP bucket', 'sample stopped bucket', 'ps --status running --quiet bucket', 'kill bucket']
    expect(commands).toEqual(interruption)
    kill.open()
    await recovery
    expect(commands).toEqual([...interruption, 'start bucket sync', 'exec -T sync /numen-sync healthcheck'])
  })

  it('fails when the upload reaches its full size before the bucket stops', async () => {
    const commands: string[] = []
    const stack = new OwnStack('test', async (args, _stdin, options) => {
      if (args.at(-1) === '/proc/net/dev') return formatTraffic(options?.signal === undefined ? 10 : 31)
      if (args[3] === 'sh') return formatStoppedSample(110)
      commands.push(args.join(' '))
      return args[0] === 'ps' ? 'bucket' : ''
    })

    stack.bucketTransfer.breakAfter('file', 'received', 20)
    await stack.bucketTransfer.recordFile('file', 100)

    await expect(stack.startServer()).rejects.toThrow('the full file could have transferred before the bucket stopped')
    expect(commands).toEqual(['kill -s SIGSTOP bucket', 'ps --status running --quiet bucket', 'kill bucket'])
  })

  it('fails when the stopped bucket has queued the rest of the download', async () => {
    const commands: string[] = []
    const stack = new OwnStack('test', async (args, _stdin, options) => {
      if (args.at(-1) === '/proc/net/dev') return formatTraffic(0, options?.signal === undefined ? 10 : 36)
      if (args[3] === 'sh') return formatStoppedSample(0, 80, 30)
      commands.push(args.join(' '))
      return args[0] === 'ps' ? 'bucket' : ''
    })

    await stack.bucketTransfer.recordFile('file', 100)
    stack.bucketTransfer.breakAfter('file', 'sent', 20)

    await expect(stack.startServer()).rejects.toThrow('the full file could have transferred before the bucket stopped')
    expect(commands).toEqual(['kill -s SIGSTOP bucket', 'ps --status running --quiet bucket', 'kill bucket'])
  })

  it('fails when the whole file transferred before the first interruption probe', async () => {
    const commands: string[] = []
    const stack = new OwnStack('test', async (args, _stdin, options) => {
      if (args.at(-1) === '/proc/net/dev') return formatTraffic(0, options?.signal === undefined ? 10 : 110)
      commands.push(args.join(' '))
      return ''
    })

    await stack.bucketTransfer.recordFile('file', 100)
    stack.bucketTransfer.breakAfter('file', 'sent', 20)

    await expect(stack.startServer()).rejects.toThrow('the file transfer reached its full size before the bucket could be stopped')
    expect(commands).toEqual([])
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

    await stack.bucketTransfer.recordFile('file', 100)
    stack.bucketTransfer.breakAfter('file', 'received', 20)
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

    await stack.bucketTransfer.recordFile('file', 100)
    stack.bucketTransfer.breakAfter('file', 'received', 20)
    await stack.resetServer()

    expect(probe?.aborted).toBe(true)
    expect(commands).toEqual(['start bucket sync', 'exec -T sync /numen-sync healthcheck'])
  })
})

describe('stand removal expiry', () => {
  const target = { vaultId: '1'.repeat(32), fileId: '2'.repeat(32) }

  it('waits for the exact removal, ages it and verifies its floor after pruning', async () => {
    const commands: string[] = []
    const answers = ['0', '1', '1', '0:1']
    const stack = new OwnStack('test', async (args, stdin) => {
      commands.push(stdin ?? args.join(' '))
      return stdin === undefined ? '' : (answers.shift() ?? '')
    })

    await stack.forgetRemovals([target])

    expect(commands).toHaveLength(5)
    expect(commands[0]).toContain('FROM files WHERE deleted AND')
    expect(commands[1]).toBe(commands[0])
    expect(commands[2]).toContain('UPDATE files SET committed_at')
    expect(commands[3]).toBe('exec -T sync /numen-sync admin prune-tombstones')
    expect(commands[4]).toContain('FROM file_floors WHERE')
    for (const index of [0, 1, 2, 4]) {
      expect(commands[index]).toContain(target.vaultId)
      expect(commands[index]).toContain(target.fileId)
    }
  })

  it.each(['1:1', '0:0'])('fails when the server reports removal state %s after pruning', async (state) => {
    const answers = ['1', '1', state]
    const stack = new OwnStack('test', async (_args, stdin) => (stdin === undefined ? '' : (answers.shift() ?? '')))

    await expect(stack.forgetRemovals([target])).rejects.toThrow('the server did not prune the scenario removal records')
  })

  it('leaves the server alone when the scenario has no removal', async () => {
    const compose = vi.fn<() => Promise<string>>().mockResolvedValue('')
    const stack = new OwnStack('test', compose)

    await stack.forgetRemovals([])

    expect(compose).not.toHaveBeenCalled()
  })
})
