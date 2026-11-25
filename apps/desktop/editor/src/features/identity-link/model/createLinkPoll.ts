/**
 * The reads of one link at a fixed interval. Each read carries its number: an answer older than one already
 * applied is dropped, and one that arrives after a reset or a stop is dropped too. Reads that keep failing
 * stop the interval, and a link that waits for its channel through too many reads stops it as well; both are reported
 * once as stalled, so the dialog does not wait on a core that does not answer.
 */
import type { Result } from '@numen/result'
import type { LinkInfo } from '@/entities/identity'

const MAX_READ_FAILURES = 6

export interface LinkPollOptions {
  read: () => Promise<Result<LinkInfo, string>>
  isEnded: (info: LinkInfo) => boolean
  onInfo: (info: LinkInfo) => void
  isWaiting: (info: LinkInfo) => boolean
  stallAfter: number
  onStalled: () => void
  intervalMs: number
  run: (task: Promise<void>) => void
}

export function createLinkPoll(options: LinkPollOptions) {
  let timer: ReturnType<typeof setInterval> | undefined
  let epoch = 0
  let lastRead = 0
  let lastApplied = 0
  let failures = 0
  let waits = 0
  let isStopped = false

  const stop = (): void => {
    if (timer !== undefined) clearInterval(timer)
    timer = undefined
  }

  const fail = (): void => {
    failures += 1
    if (failures !== MAX_READ_FAILURES) return
    stop()
    options.onStalled()
  }

  const readOnce = async (): Promise<void> => {
    lastRead += 1
    const number = lastRead
    const started = epoch
    const answer = await options.read().catch(() => undefined)
    if (isStopped || started !== epoch) return
    if (answer === undefined || !answer.ok) {
      fail()
      return
    }
    if (number < lastApplied) return
    lastApplied = number
    failures = 0
    options.onInfo(answer.value)
    waits = options.isWaiting(answer.value) ? waits + 1 : 0
    if (waits >= options.stallAfter) options.onStalled()
    if (options.isEnded(answer.value)) stop()
  }

  return {
    readOnce,
    stop,
    reset(): void {
      stop()
      epoch += 1
      failures = 0
      waits = 0
    },
    start(): void {
      stop()
      timer = setInterval(() => options.run(readOnce()), options.intervalMs)
    },
    dispose(): void {
      stop()
      isStopped = true
    },
  }
}
