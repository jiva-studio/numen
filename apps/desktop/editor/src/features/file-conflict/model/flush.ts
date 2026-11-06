/**
 * Flushes the window's unsaved work when the application requests quit. A tab
 * saves after typing stops, so the latest edits exist only in its buffer; the
 * application requests them over a stream this page subscribes to while rendered.
 * Text that cannot be written raises a conflict, and the window stays open
 * until every conflict is resolved.
 */
import { shallowRef, type Ref } from 'vue'
import { createFollower } from '@numen/ui'

/** Outcome of a flush: everything written, or a user decision pending. */
export type FlushResult = 'written' | 'asking'

/** What the flush needs of the core. */
export interface FlushDeps {
  watchQuit(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  reportFlush(token: string, result?: FlushResult): Promise<unknown>
}

/** Somewhere unwritten work is held, which the flush calls and waits for. */
export type FlushHandler = () => Promise<unknown>

/** Text a file moved out from under, and the two ways of settling it. */
export interface Conflict {
  /** The identity the note holding the text opened under. */
  readonly note: string
  /** Write the user's text over the file's content. */
  readonly keep: () => Promise<unknown>
  /** Take what the file holds, and let the typing go. */
  readonly take: () => Promise<unknown>
}

/** A conflict as presented to the user. */
export interface ConflictPrompt {
  readonly note: string
  readonly keep: () => Promise<unknown>
  readonly take: () => Promise<unknown>
  /** Put it off. It stays standing, undrawn, and the window stays. */
  readonly later: () => void
}

/**
 * Tracks unresolved conflicts. `raise` registers one and returns a function
 * that removes it; a postponed conflict stays outstanding but is not prompted.
 */
function createConflictRegistry(onChanged: () => void) {
  const outstanding = new Set<Conflict>()
  const postponed = new Set<string>()
  const conflicts = shallowRef([]) as Ref<readonly ConflictPrompt[]>

  const prompt = (one: Conflict): ConflictPrompt => ({
    note: one.note,
    keep: one.keep,
    take: one.take,
    later: () => {
      postponed.add(one.note)
      conflicts.value = conflicts.value.filter((each) => each.note !== one.note)
    },
  })

  const refresh = () => {
    const all = [...outstanding]
    conflicts.value = all.filter((one) => !postponed.has(one.note)).map(prompt)
    return all
  }

  const raise = (one: Conflict) => {
    outstanding.add(one)
    onChanged()
    return () => {
      if (!outstanding.delete(one)) return
      postponed.delete(one.note)
      onChanged()
    }
  }

  return {
    conflicts,
    raise,
    refresh,
    clearPut: () => postponed.clear(),
  }
}

/**
 * Answers the application's quit request: every registered handler writes its
 * unsaved work, and all writes finish before the response is sent. An error
 * does not keep the window open, since the user requested the close. A conflict
 * does: the text is still present and only the user decides where it is
 * written.
 */
function createFlushReporter(
  core: FlushDeps,
  registry: ReturnType<typeof createConflictRegistry>,
  handlers: Set<FlushHandler>,
) {
  let underVal: string | null = null
  let lastResult: FlushResult | null = null
  let writing: Promise<unknown> | null = null

  async function report() {
    const token = underVal
    if (token === null) return
    const all = registry.refresh()
    if (all.length === 0 && writing !== null) return
    const result: FlushResult = all.length > 0 ? 'asking' : 'written'
    if (result === lastResult) return
    lastResult = result
    await core.reportFlush(token, result)
  }

  /** A report that failed was not received, so the next one is sent again. */
  function forgetResult() {
    lastResult = null
  }

  function reportInBackground() {
    report().catch(forgetResult)
  }

  async function answer(token: string) {
    underVal = token
    lastResult = null
    registry.clearPut()
    const pendingWrites = Promise.allSettled([...handlers].map((one) => one()))
    writing = pendingWrites
    pendingWrites
      .then(() => {
        if (writing !== pendingWrites) return
        writing = null
        reportInBackground()
      })
      .catch(forgetResult)
    await report()
  }

  const reset = () => {
    underVal = null
    lastResult = null
    writing = null
  }

  return { answer, report, reportInBackground, reset }
}

export function createFileFlush(core: FlushDeps, wait: (ms: number) => Promise<unknown> = sleep) {
  const handlers = new Set<FlushHandler>()
  let open = true
  const abortController = new AbortController()

  const reporterHolder: { current?: ReturnType<typeof createFlushReporter> } = {}
  // Stryker disable next-line OptionalChaining: Assigned synchronously before conflict callbacks run.
  const registry = createConflictRegistry(() => reporterHolder.current?.reportInBackground())
  const reporter = createFlushReporter(core, registry, handlers)
  reporterHolder.current = reporter

  const addHandler = (one: FlushHandler) => {
    handlers.add(one)
    return () => handlers.delete(one)
  }

  const follows = createFollower({
    isOpen: () => open,
    // The quit stream reports no connection status.
    setLost: () => undefined,
    wait,
    reset: reporter.reset,
  })

  /** Listen for the quit, for as long as the window is drawn. */
  const start = () =>
    follows(
      () => core.watchQuit(abortController.signal),
      async (message) => {
        if (message.flush) await reporter.answer(message.token)
      },
    )

  return {
    addHandler,
    raise: registry.raise,
    conflicts: registry.conflicts,
    start,
    close: () => {
      open = false
      abortController.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
