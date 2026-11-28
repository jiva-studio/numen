export interface ComposeOptions {
  readonly timeout?: number
  readonly signal?: AbortSignal
}

export type ComposeCommand = (args: readonly string[], stdin?: string, options?: ComposeOptions) => Promise<string>

const BREAK_LIMIT_MS = 30_000
const PROBE_LIMIT_MS = 10_000
const STOPPED_SAMPLE = [
  'for status in /proc/1/task/*/status; do grep "^State:" "$status" || exit 1; done',
  'printf "\nqueues\n"',
  'cat /proc/net/tcp /proc/net/tcp6 || exit 1',
  'printf "\ntraffic\n"',
  'cat /proc/net/dev',
].join('\n')

interface Traffic {
  readonly received: number
  readonly sent: number
}

type TransferDirection = keyof Traffic

interface TransferBaseline {
  readonly fileBytes: number
  readonly traffic: Traffic
}

interface InterruptionRequest {
  readonly filePath: string
  readonly direction: TransferDirection
  readonly minimum: number
}

interface InterruptionWatch {
  readonly controller: AbortController
  readonly result: Promise<Error | undefined>
}

const readTraffic = (text: string): Traffic => {
  const match = /(?:^|\n)\s*eth0:\s*(\d+(?:\s+\d+){8})\b/.exec(text)
  const fields = match?.[1]?.trim().split(/\s+/).map(Number)
  const received = fields?.[0]
  const sent = fields?.[8]
  if (received === undefined || sent === undefined || !Number.isSafeInteger(received) || !Number.isSafeInteger(sent)) {
    throw new Error('the bucket container shows no eth0 traffic')
  }
  return { received, sent }
}

const readSendQueue = (text: string): number => {
  const lines = text.trim().split('\n').map((line) => line.trim())
  if (!lines.some((line) => line.startsWith('sl'))) throw new Error('the bucket container shows no TCP socket counters')
  let bytes = 0
  for (const line of lines) {
    if (line === '' || line.startsWith('sl')) continue
    const match = /^\d+:\s+\S+\s+\S+\s+[\da-f]{2}\s+([\da-f]+):[\da-f]+\b/i.exec(line)
    if (match?.[1] === undefined) throw new Error('the bucket container shows an invalid TCP socket counter')
    bytes += Number.parseInt(match[1], 16)
  }
  return bytes
}

/** Measures and interrupts file transfers on the scenario's bucket container. */
export class BucketTransfer {
  private readonly baselines = new Map<string, TransferBaseline>()
  private request: InterruptionRequest | undefined
  private watcher: InterruptionWatch | undefined
  private hasSuspendedBucket = false

  constructor(private readonly compose: ComposeCommand) {}

  async recordFile(filePath: string, fileBytes: number): Promise<void> {
    const output = await this.compose(['exec', '-T', 'bucket', 'cat', '/proc/net/dev'], undefined, { timeout: PROBE_LIMIT_MS })
    const baseline = { fileBytes, traffic: readTraffic(output) }
    this.baselines.set(filePath, baseline)
    if (this.request?.filePath === filePath) this.startWatch(this.request, baseline)
  }

  breakAfter(filePath: string, direction: TransferDirection, minimum: number): void {
    if (this.request !== undefined || this.watcher !== undefined) throw new Error('the stand already waits for a transfer interruption')
    const request = { filePath, direction, minimum }
    const baseline = this.baselines.get(filePath)
    if (baseline === undefined) this.request = request
    else this.startWatch(request, baseline)
  }

  async waitForInterruption(): Promise<void> {
    if (this.request !== undefined) throw new Error('no transfer was recorded for ' + this.request.filePath)
    const error = await this.watcher?.result
    if (error !== undefined) throw error
  }

  async finish(): Promise<void> {
    await this.waitForInterruption()
    this.watcher = undefined
  }

  async reset(): Promise<void> {
    this.request = undefined
    this.baselines.clear()
    const watcher = this.watcher
    this.watcher = undefined
    if (watcher !== undefined) {
      watcher.controller.abort(new Error('the scenario ended before the transfer broke'))
      await watcher.result
    }
    if (this.hasSuspendedBucket) await this.killBucket()
  }

  private startWatch(request: InterruptionRequest, baseline: TransferBaseline): void {
    this.request = undefined
    if (request.minimum >= baseline.fileBytes) throw new Error('the file is too small for a transfer interruption')
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(new Error('the transfer did not break within 30 seconds')), BREAK_LIMIT_MS)
    const result = this.watchTraffic(request, baseline, controller.signal)
      .catch((cause: unknown) => {
        const reason: unknown = controller.signal.reason
        if (reason instanceof Error) return reason
        return cause instanceof Error ? cause : new Error('the bucket traffic could not be interrupted', { cause })
      })
      .finally(() => clearTimeout(timeout))
    this.watcher = { controller, result }
  }

  private async watchTraffic(request: InterruptionRequest, baseline: TransferBaseline, signal: AbortSignal): Promise<undefined> {
    for (;;) {
      const output = await this.compose(['exec', '-T', 'bucket', 'cat', '/proc/net/dev'], undefined, { signal, timeout: PROBE_LIMIT_MS })
      signal.throwIfAborted()
      const transferred = readTraffic(output)[request.direction] - baseline.traffic[request.direction]
      if (transferred < 0) throw new Error('the bucket traffic counters restarted during the transfer')
      if (transferred >= request.minimum) {
        if (transferred >= baseline.fileBytes) throw new Error('the file transfer reached its full size before the bucket could be stopped')
        await this.interruptTransfer(request.direction, baseline, signal)
        return
      }
    }
  }

  private async interruptTransfer(direction: TransferDirection, baseline: TransferBaseline, signal: AbortSignal): Promise<void> {
    this.hasSuspendedBucket = true
    try {
      await this.compose(['kill', '-s', 'SIGSTOP', 'bucket'], undefined, { signal, timeout: PROBE_LIMIT_MS })
      const upperBound = await this.readStoppedTransfer(direction, baseline, signal)
      if (upperBound >= baseline.fileBytes) throw new Error('the full file could have transferred before the bucket stopped')
    } finally {
      await this.killBucket()
    }
  }

  private async readStoppedTransfer(direction: TransferDirection, baseline: TransferBaseline, signal: AbortSignal): Promise<number> {
    for (;;) {
      const sample = await this.compose(['exec', '-T', 'bucket', 'sh', '-c', STOPPED_SAMPLE], undefined, { signal, timeout: PROBE_LIMIT_MS })
      signal.throwIfAborted()
      const [threads, ...sections] = sample.split(/\nqueues\n|\ntraffic\n/)
      const states = threads.trim().split('\n')
      if (!states.every((state) => /^State:\s+T\b/.test(state))) continue
      if (sections.length !== 2) throw new Error('the stopped bucket shows no transfer counters')
      const [queues, counters] = sections
      const transferred = readTraffic(counters)[direction] - baseline.traffic[direction]
      if (transferred < 0) throw new Error('the bucket traffic counters restarted during the transfer')
      // The stopped process can send only bytes already queued in TCP. The queue is read before the counters.
      return transferred + (direction === 'sent' ? readSendQueue(queues) : 0)
    }
  }

  private async killBucket(): Promise<void> {
    const id = await this.compose(['ps', '--status', 'running', '--quiet', 'bucket'], undefined, { timeout: PROBE_LIMIT_MS })
    if (id.trim() !== '') await this.compose(['kill', 'bucket'], undefined, { timeout: PROBE_LIMIT_MS })
    this.hasSuspendedBucket = false
  }
}
