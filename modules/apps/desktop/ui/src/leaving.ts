/**
 * Writing out what the window still owes, when the window goes.
 *
 * A tab saves once the typing stops, so at any moment the last seconds of work
 * are in a buffer here and nowhere else. The application asks for them over a
 * stream this page listens on for as long as it is drawn, and waits for the
 * answer.
 *
 * Some of that work cannot be written: the file changed under it, and only the
 * person can say what the note ends up holding. Whatever holds such text raises
 * a question, the application is told there are questions outstanding, and the
 * window stays until they are answered.
 */
import { ref, type Ref } from 'vue'

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
  /** The note the text belongs to. */
  readonly path: string
  /** Write what the person has, over what the file holds. */
  readonly keep: () => Promise<unknown>
  /** Take what the file holds, and let the typing go. */
  readonly take: () => Promise<unknown>
}

/** A question as the window draws it. */
export interface Standing {
  readonly path: string
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
  /** The paths a person put off. They stand and are not drawn. */
  const put = new Set<string>()
  /** The questions to draw, which is everything standing bar what was put off. */
  const questions = ref([]) as Ref<readonly Standing[]>

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
      put.delete(one.path)
      void say()
    }
  }

  async function answer(token: string) {
    under = token
    told = null
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
    const standing = [...outstanding]
    for (const path of [...put]) {
      if (!standing.some((one) => one.path === path)) put.delete(path)
    }
    questions.value = standing.filter((one) => !put.has(one.path)).map(drawn)
    // A write still in the air is not something a person answers, and it is
    // not an answer either.
    if (standing.length === 0 && writing !== null) return
    const owed: Owed = standing.length > 0 ? 'asking' : 'written'
    if (owed === told) return
    told = owed
    await core.flushed(token, owed)
  }

  /** One question with the three ways out of it. */
  const drawn = (one: Question): Standing => ({
    path: one.path,
    keep: one.keep,
    take: one.take,
    later: () => {
      put.add(one.path)
      questions.value = questions.value.filter((drawing) => drawing.path !== one.path)
    },
  })

  /**
   * Listen for the quit, taken up again the way following is. A window that
   * had stopped listening would look exactly like one with nothing to write.
   */
  async function start() {
    while (open) {
      try {
        for await (const said of core.quitting(listening.signal)) {
          if (!open) return
          if (said.flush) await answer(said.token)
        }
      } catch {
        if (!open) return
      }
      // The stream went with whatever was standing still standing, and under a
      // token nothing answers to any more. It is asked for again.
      under = null
      told = null
      writing = null
      await wait(1000)
    }
  }

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
