/**
 * A saved search service at the outer edge of the tab, scripted by a test or a story: it answers
 * GetResults with the response it holds, yields the changes it is given on the open watch, and
 * writes an UpdateSearch into the response it answers next, as the backend writes the note.
 */
import { ServiceFailure, type SavedSearchService } from '../api/service'
import type {
  GetResultsRequest,
  GetResultsResponse,
  ResultsChange,
  UpdateSearchRequest,
  UpdateSearchResponse,
} from '../api/types'

type WatchEvent =
  | { readonly kind: 'change'; readonly change: ResultsChange }
  | { readonly kind: 'end' }
  | { readonly kind: 'failure'; readonly message: string }

interface Watch {
  readonly queue: WatchEvent[]
  wake: () => void
}

interface HeldRead {
  readonly answer: GetResultsResponse
  readonly resolve: (answer: GetResultsResponse) => void
}

/** The scripted service and what it was asked. */
export interface ScriptedService extends SavedSearchService {
  readonly calls: string[]
  readonly reads: GetResultsRequest[]
  readonly readSignals: AbortSignal[]
  readonly watchSignals: AbortSignal[]
  readonly updates: UpdateSearchRequest[]
  setAnswer(answer: GetResultsResponse): void
  /** Every GetResults from now fails with the message, until setAnswer. */
  failReads(message: string): void
  /** GetResults calls wait until releaseReads. */
  holdReads(): void
  /** Answers the waiting GetResults calls, the newest first, each with the answer of its time. */
  releaseReads(): void
  /** A change on the open watch, or on the next one when none is open. */
  emit(change: ResultsChange): void
  endWatch(): void
  failWatch(message: string): void
  /** What UpdateSearch answers from now; a string fails it with that message. */
  setUpdateAnswer(answer: UpdateSearchResponse | string): void
}

const abortError = (): DOMException => new DOMException('The call was aborted.', 'AbortError')

/** A service answering `answer` until told otherwise. */
export function createScriptedService(first: GetResultsResponse): ScriptedService {
  let answer = first
  let readFailure = ''
  let isHolding = false
  let held: HeldRead[] = []
  let updateAnswer: UpdateSearchResponse | string = { errors: [] }
  let watch: Watch | null = null
  const pending: WatchEvent[] = []
  const calls: string[] = []
  const reads: GetResultsRequest[] = []
  const readSignals: AbortSignal[] = []
  const watchSignals: AbortSignal[] = []
  const updates: UpdateSearchRequest[] = []

  const push = (event: WatchEvent): void => {
    if (!watch) {
      pending.push(event)
      return
    }
    watch.queue.push(event)
    watch.wake()
  }

  const waitForRelease = (signal: AbortSignal): Promise<GetResultsResponse> =>
    new Promise((resolve, reject) => {
      held.push({ answer, resolve })
      signal.addEventListener('abort', () => reject(abortError()), { once: true })
    })

  const scripted: ScriptedService = {
    calls,
    reads,
    readSignals,
    watchSignals,
    updates,

    async getResults(request, signal) {
      calls.push('getResults')
      reads.push(request)
      readSignals.push(signal)
      const now = isHolding ? await waitForRelease(signal) : answer
      if (signal.aborted) throw abortError()
      if (readFailure !== '') throw new ServiceFailure(readFailure)
      return now
    },

    async *watchResults(_request, signal) {
      calls.push('watchResults')
      watchSignals.push(signal)
      const own: Watch = { queue: pending.splice(0), wake: () => undefined }
      watch = own
      const onAbort = (): void => own.wake()
      signal.addEventListener('abort', onAbort)
      try {
        for (;;) {
          if (signal.aborted) return
          const next = own.queue.shift()
          if (!next) {
            await new Promise<void>((resolve) => (own.wake = resolve))
            continue
          }
          if (next.kind === 'end') return
          if (next.kind === 'failure') throw new ServiceFailure(next.message)
          yield next.change
        }
      } finally {
        signal.removeEventListener('abort', onAbort)
        if (watch === own) watch = null
      }
    },

    async updateSearch(request, signal) {
      calls.push('updateSearch')
      updates.push(request)
      if (signal.aborted) throw abortError()
      if (typeof updateAnswer === 'string') throw new ServiceFailure(updateAnswer)
      if (updateAnswer.errors.length === 0) {
        answer = { ...answer, query: request.query, rerank: request.rerank, limit: request.limit }
      }
      return updateAnswer
    },

    setAnswer(next) {
      answer = next
      readFailure = ''
    },
    failReads(message) {
      readFailure = message
    },
    holdReads() {
      isHolding = true
    },
    releaseReads() {
      isHolding = false
      const waiting = held.reverse()
      held = []
      for (const one of waiting) one.resolve(one.answer)
    },
    emit(change) {
      push({ kind: 'change', change })
    },
    endWatch() {
      push({ kind: 'end' })
    },
    failWatch(message) {
      push({ kind: 'failure', message })
    },
    setUpdateAnswer(next) {
      updateAnswer = next
    },
  }
  return scripted
}
