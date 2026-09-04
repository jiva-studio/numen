/**
 * Writing out what the window still owes, when the window goes.
 *
 * A tab saves once the typing stops, so the last seconds of work are in a
 * buffer here and nowhere else. The application asks for them over a stream
 * this page listens on for as long as it is drawn, and waits for the answer.
 *
 * Text that cannot be written raises a question, and the window stays until
 * every question is answered.
 */
import { ref, type Ref } from 'vue'
import { following } from '@numen/ui'

/** What this page has left when it answers. */
export type Owed = 'written' | 'asking'

/** What the quit needs of the core. */
export interface Going {
  quitting(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  flushed(token: string, owed?: Owed): Promise<unknown>
}

/** Somewhere unwritten work is held, which the quit waits for. */
export type Owing = () => Promise<unknown>

/** Text that could not be written, and the two ways out of it. */
export interface Question {
  /** The identity the note holding the text opened under. */
  readonly note: string
  /** Write what the person has, over what the file holds. */
  readonly keep: () => Promise<unknown>
  /** Take what the file holds, and let the typing go. */
  readonly take: () => Promise<unknown>
}

/** A question as the window draws it. */
export interface DrawnQuestion {
  readonly note: string
  readonly keep: () => Promise<unknown>
  readonly take: () => Promise<unknown>
  /** Put it off. It stays standing, undrawn, and the window stays. */
  readonly later: () => void
}

/**
 * Answering the application when it asks for what the window owes.
 *
 * Whatever holds unwritten work adds itself, and every one of them is written
 * before the answer goes back. A refusal is not a reason to keep the window
 * open: the person asked for it to go. A question is: the text is still here,
 * and nothing but the person decides where it goes.
 */
export function leaving(core: Going, wait: (ms: number) => Promise<unknown> = sleep) {
  const owing = new Set<Owing>()
  let open = true
  const listening = new AbortController()

  /** The token the application is being answered under, once it has asked. */
  let under: string | null = null
  /** The last thing said under that token, which is not worth saying twice. */
  let told: Owed | null = null
  /** The writes owed at the ask, while they are still in the air. */
  let writing: Promise<unknown> | null = null
  /** Every question a person has to answer, drawn or put off. */
  const outstanding = new Set<Question>()
  /** The notes a person put off. They stand and are not drawn. */
  const put = new Set<string>()
  /** The questions to draw, which is everything standing bar what was put off. */
  const questions = ref([]) as Ref<readonly DrawnQuestion[]>

  /** Something the quit waits for, until what this answers with is called. */
  const holds = (one: Owing) => {
    owing.add(one)
    return () => owing.delete(one)
  }

  /**
   * Text a person has to answer for, standing until what this answers with is
   * called. It is told to the application as soon as it is raised: whatever
   * holds the text is waiting on a person and finishes when they say so.
   */
  const raise = (one: Question) => {
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
    const owed = Promise.allSettled([...owing].map((one) => one()))
    writing = owed
    void owed.then(() => {
      if (writing !== owed) return
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
    questions.value = all.filter((one) => !put.has(one.note)).map(drawn)
    // A write still in the air is not something a person answers, and it is
    // not an answer either.
    if (all.length === 0 && writing !== null) return
    const owed: Owed = all.length > 0 ? 'asking' : 'written'
    if (owed === told) return
    told = owed
    await core.flushed(token, owed)
  }

  /** One question with the three ways out of it. */
  const drawn = (one: Question): DrawnQuestion => ({
    note: one.note,
    keep: one.keep,
    take: one.take,
    later: () => {
      put.add(one.note)
      questions.value = questions.value.filter((drawing) => drawing.note !== one.note)
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
    holds,
    raise,
    questions,
    start,
    close: () => {
      open = false
      listening.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
