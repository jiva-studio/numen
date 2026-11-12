/**
 * Flushes the window's unsaved work when the application requests quit. A tab
 * saves after typing stops, so the latest edits exist only in its buffer; the
 * application requests them over a stream this page subscribes to while rendered.
 * Text that cannot be written raises a conflict, and the window stays open
 * until every conflict is resolved. A handler that fails leaves the flush
 * unwritten too, so a quit never reports text written that no handler confirmed.
 */
import { shallowRef, type Ref } from 'vue'
import { createFollower } from '@numen/ui'

/** Outcome of a flush: everything written, or a user decision pending. */
type FlushResult = 'written' | 'unsaved'

/** What the flush needs of the core. */
export interface FlushDeps {
  watchQuit(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  reportFlush(token: string, result?: FlushResult): Promise<void>
}

/**
 * Somewhere unwritten work is held, which the flush calls and waits for. It
 * answers the conflicts of the text it could not write, and nothing when all
 * of it is written.
 */
export type FlushHandler = () => Promise<void | readonly Conflict[]>

type HandlerResults = PromiseSettledResult<void | readonly Conflict[]>[]

/** Text that is not written, and the two ways of settling it. */
export interface Conflict {
  readonly note: string
  readonly keep: () => Promise<void>
  readonly take: () => Promise<void>
}

/** A conflict as presented to the user. */
export interface ConflictPrompt {
  readonly note: string
  readonly keep: () => Promise<void>
  readonly take: () => Promise<void>
  readonly later: () => void
}

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

function createFlushReporter(
  core: FlushDeps,
  registry: ReturnType<typeof createConflictRegistry>,
  handlers: Set<FlushHandler>,
) {
  let underVal: string | null = null
  let lastResult: FlushResult | null = null
  let writing: Promise<HandlerResults> | null = null
  let failed = false
  let dropHeld: (() => void)[] = []

  async function report() {
    const token = underVal
    if (token === null) return
    const all = registry.refresh()
    if (all.length === 0 && !failed && writing !== null) return
    const result: FlushResult = all.length > 0 || failed ? 'unsaved' : 'written'
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

  /** Settles a conflict the handlers raised, and calls the handlers again to see what is left. */
  const retrying = (one: Conflict): Conflict => ({
    note: one.note,
    keep: async () => {
      await one.keep()
      startWrites()
    },
    take: async () => {
      await one.take()
      startWrites()
    },
  })

  /** Replaces what the last run of the handlers left unwritten with what this one did. */
  function adopt(results: HandlerResults) {
    for (const drop of dropHeld) drop()
    dropHeld = []
    failed = results.some((one) => one.status === 'rejected')
    for (const one of results) {
      if (one.status !== 'fulfilled' || one.value === undefined) continue
      for (const conflict of one.value) dropHeld.push(registry.raise(retrying(conflict)))
    }
  }

  function startWrites() {
    const pendingWrites = Promise.allSettled(
      [...handlers].map((one) => new Promise<void | readonly Conflict[]>((done) => done(one()))),
    )
    writing = pendingWrites
    pendingWrites
      .then((results) => {
        if (writing !== pendingWrites) return
        writing = null
        adopt(results)
        reportInBackground()
      })
      .catch(forgetResult)
  }

  async function answer(token: string) {
    underVal = token
    lastResult = null
    registry.clearPut()
    startWrites()
    await report()
  }

  const reset = () => {
    underVal = null
    lastResult = null
    writing = null
    failed = false
  }

  return { answer, report, reportInBackground, reset }
}

export function createFileFlush(core: FlushDeps, wait: (ms: number) => Promise<void> = sleep) {
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

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))
