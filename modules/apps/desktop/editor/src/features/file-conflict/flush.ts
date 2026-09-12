/**
 * Flushing the window's unwritten work, when the application asks to quit. A
 * tab saves once the typing stops, so the last seconds of work are in a buffer
 * here and nowhere else, and the application asks for them over a stream this
 * page listens on for as long as it is drawn. Text that cannot be written
 * raises a conflict, and the window stays until every conflict is settled.
 */
import { shallowRef, type Ref } from 'vue'
import { following } from '@numen/ui'

/** How a flush came out: everything written, or a person still being asked. */
export type FlushResult = 'written' | 'asking'

/** What the flush needs of the core. */
export interface FlushDeps {
  quitting(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  flushed(token: string, result?: FlushResult): Promise<unknown>
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
 * The conflict a file a tab holds stands in, and nothing while it stands in
 * neither. A screen tells its files apart in its own words; these are the two
 * this folder has something to do about.
 */
export type FileConflict = 'gone' | 'stale' | null

/** Which conflict a screen's word names, and nothing for every other word. */
export const conflictIn = (state: string): FileConflict =>
  state === 'gone' || state === 'stale' ? state : null

/**
 * Answering the application when it asks the window to write out what it holds.
 *
 * Whatever holds unwritten work adds itself, and every one of them is written
 * before the answer goes back. An error is not a reason to keep the window
 * open: the person asked for it to go. A conflict is: the text is still here,
 * and nothing but the person decides where it goes.
 */
export function useFileFlush(core: FlushDeps, wait: (ms: number) => Promise<unknown> = sleep) {
  const handlers = new Set<FlushHandler>()
  let open = true
  const listening = new AbortController()

  /** The token the application is being answered under, once it has asked. */
  let under: string | null = null
  /** The last thing said under that token, which is not worth saying twice. */
  let told: FlushResult | null = null
  /** The writes owed at the ask, while they are still in the air. */
  let writing: Promise<unknown> | null = null
  /** Every conflict a person has to settle, drawn or put off. */
  const outstanding = new Set<Conflict>()
  /** The notes a person put off. They stand and are not drawn. */
  const put = new Set<string>()
  /** The conflicts to draw, which is everything standing bar what was put off. */
  const conflicts = shallowRef([]) as Ref<readonly ConflictPrompt[]>

  /** Something the flush waits for, until what this answers with is called. */
  const addHandler = (one: FlushHandler) => {
    handlers.add(one)
    return () => handlers.delete(one)
  }

  /**
   * Text a person has to settle, standing until what this answers with is
   * called. It is told to the application as soon as it is raised: whatever
   * holds the text is waiting on a person and finishes when they say so.
   */
  const raise = (one: Conflict) => {
    outstanding.add(one)
    void say()
    return () => {
      if (!outstanding.delete(one)) return
      put.delete(one.note)
      void say()
    }
  }

  async function answer(token: string) {
    under = token
    told = null
    // Every asking is put to the person whole. What was put off last time is
    // drawn again, because this is a fresh reason to answer it.
    put.clear()
    const written = Promise.allSettled([...handlers].map((one) => one()))
    writing = written
    void written.then(() => {
      if (writing !== written) return
      writing = null
      void say()
    })
    await say()
  }

  /** What is standing, drawn, and told to the application. */
  async function say() {
    const token = under
    if (token === null) return
    const all = [...outstanding]
    for (const note of [...put]) {
      if (!all.some((one) => one.note === note)) put.delete(note)
    }
    conflicts.value = all.filter((one) => !put.has(one.note)).map(prompt)
    // A write still in the air is not something a person answers, and it is
    // not an answer either.
    if (all.length === 0 && writing !== null) return
    const result: FlushResult = all.length > 0 ? 'asking' : 'written'
    if (result === told) return
    told = result
    await core.flushed(token, result)
  }

  /** One conflict with the three ways out of it. */
  const prompt = (one: Conflict): ConflictPrompt => ({
    note: one.note,
    keep: one.keep,
    take: one.take,
    later: () => {
      put.add(one.note)
      conflicts.value = conflicts.value.filter((each) => each.note !== one.note)
    },
  })

  const follows = following({
    open: () => open,
    lost: () => {},
    wait,
    // The stream went with whatever was standing still standing, and under a
    // token nothing answers to any more. It is asked for again.
    reset: () => {
      under = null
      told = null
      writing = null
    },
  })

  /** Listen for the quit, for as long as the window is drawn. */
  const start = () =>
    follows(
      () => core.quitting(listening.signal),
      async (said) => {
        if (said.flush) await answer(said.token)
      },
    )

  return {
    addHandler,
    raise,
    conflicts,
    start,
    close: () => {
      open = false
      listening.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
