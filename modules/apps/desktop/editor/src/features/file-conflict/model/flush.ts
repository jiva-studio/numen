/**
 * Flushing the window's unwritten work, when the application asks to quit. A
 * tab saves once the typing stops, so the last seconds of work are in a buffer
 * here and nowhere else, and the application asks for them over a stream this
 * page listens on for as long as it is drawn. Text that cannot be written
 * raises a conflict, and the window stays until every conflict is settled.
 */
import { shallowRef, type Ref } from 'vue'
import { createFollower } from '@numen/ui'

/** How a flush came out: everything written, or a person still being asked. */
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
  /** Write what the person has, over what the file holds. */
  readonly keep: () => Promise<unknown>
  /** Take what the file holds, and let the typing go. */
  readonly take: () => Promise<unknown>
}

/** A conflict as the window puts it to the person. */
export interface ConflictPrompt {
  readonly note: string
  readonly keep: () => Promise<unknown>
  readonly take: () => Promise<unknown>
  /** Put it off. It stays standing, undrawn, and the window stays. */
  readonly later: () => void
}

/**
 * Answering the application when it asks the window to write out what it holds.
 *
 * Whatever holds unwritten work adds itself, and every one of them is written
 * before the answer goes back. An error is not a reason to keep the window
 * open: the person asked for it to go. A conflict is: the text is still here,
 * and nothing but the person decides where it goes.
 */
function createConflictRegistry(onChanged: () => void) {
  const outstanding = new Set<Conflict>()
  const put = new Set<string>()
  const conflicts = shallowRef([]) as Ref<readonly ConflictPrompt[]>

  const prompt = (one: Conflict): ConflictPrompt => ({
    note: one.note,
    keep: one.keep,
    take: one.take,
    later: () => {
      put.add(one.note)
      conflicts.value = conflicts.value.filter((each) => each.note !== one.note)
    },
  })

  const refresh = () => {
    const all = [...outstanding]
    conflicts.value = all.filter((one) => !put.has(one.note)).map(prompt)
    return all
  }

  const raise = (one: Conflict) => {
    outstanding.add(one)
    onChanged()
    return () => {
      if (!outstanding.delete(one)) return
      put.delete(one.note)
      onChanged()
    }
  }

  return {
    conflicts,
    raise,
    refresh,
    clearPut: () => put.clear(),
  }
}

/**
 * Answering the application when it asks the window to write out what it holds.
 *
 * Whatever holds unwritten work adds itself, and every one of them is written
 * before the answer goes back. An error is not a reason to keep the window
 * open: the person asked for it to go. A conflict is: the text is still here,
 * and nothing but the person decides where it goes.
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

  async function answer(token: string) {
    under = token
    lastResult = null
    registry.clearPut()
    const pendingWrites = Promise.allSettled([...handlers].map((one) => one()))
    writing = pendingWrites
    void pendingWrites.then(() => {
      if (writing !== pendingWrites) return
      writing = null
      void report()
    })
    await report()
  }

  const reset = () => {
    under = null
    lastResult = null
    writing = null
  }

  return { answer, report, reset }
}

export function useFileFlush(core: FlushDeps, wait: (ms: number) => Promise<unknown> = sleep) {
  const handlers = new Set<FlushHandler>()
  let open = true
  const listening = new AbortController()

  const reporterHolder: { current?: ReturnType<typeof createFlushReporter> } = {}
  // Stryker disable next-line OptionalChaining: Assigned synchronously before conflict callbacks run.
  const registry = createConflictRegistry(() => void reporterHolder.current?.report())
  const reporter = createFlushReporter(core, registry, handlers)
  reporterHolder.current = reporter

  const addHandler = (one: FlushHandler) => {
    handlers.add(one)
    return () => handlers.delete(one)
  }

  const follows = createFollower({
    isOpen: () => open,
    setLost: () => {},
    wait,
    reset: reporter.reset,
  })

  /** Listen for the quit, for as long as the window is drawn. */
  const start = () =>
    follows(
      () => core.watchQuit(listening.signal),
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
      listening.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
