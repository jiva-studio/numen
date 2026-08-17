/**
 * What the window is showing, and the rules for changing it.
 *
 * Apart from the template because these are the rules that decide whether the
 * window keeps up with the vault, and a rule inside a component is a rule that
 * is only exercised by looking at the screen.
 */
import { ref } from 'vue'
import { rateOf } from '@numen/ui'
import type { Neighbourhood } from './plex'

/** Everything the window asks of the core, and nothing about how it is drawn. */
export interface Core {
  neighbourhood(path: string): Promise<Neighbourhood>
  opening(): Promise<{ path: string } | null>
  state(): Promise<{
    name: string
    ready: boolean
    failed: string
    unwatched: string
    /** Spans of text the index holds, and how many of them carry a vector. */
    chunks: bigint
    embedded: bigint
    /** The source being read now, empty when nothing is. */
    reading: string
    /** Whether anything is going to turn the chunks into vectors. */
    embedding: boolean
    /** Books the vault holds, and how many of those cutting has got through. */
    books: bigint
    booksRead: bigint
    /** Whether vectors are being made now, which is the second of two phases. */
    learning: boolean
    /** Whether the vault is still being read at all. */
    busy: boolean
  }>
  changes(signal: AbortSignal): AsyncIterable<{ paths: string[]; reload: boolean }>
  /** The notes something else asked to be put in front of the person. */
  focus(signal: AbortSignal): AsyncIterable<{ path: string }>
}

export function showing(
  core: Core,
  wait: (ms: number) => Promise<unknown> = sleep,
  now: () => number = () => Date.now(),
) {
  const neighbourhood = ref<Neighbourhood | null>(null)
  /**
   * The note the window is showing, as it asked for it.
   *
   * Kept apart from what came back: when the note is gone the answer carries no
   * path, and taking the next question from the focus would leave the window
   * with nothing to ask about and no way back.
   */
  const here = ref('')
  const name = ref('')
  const indexing = ref(true)
  /** The core could not be reached: nothing else in the window is true. */
  const failure = ref('')
  /** Something is wrong and the window still works: it says so and carries on. */
  const notice = ref('')
  const trouble = ref('')
  const unwatched = ref('')
  /**
   * How far reading the vault for meaning has got.
   *
   * Cutting finishes long before embedding does, so the pair is what says how
   * far there is to go. `reading` names what is being read, and is empty
   * between sources as well as after the last one.
   */
  const chunks = ref(0)
  const embedded = ref(0)
  const reading = ref('')
  /**
   * Whether anything is going to embed what was cut.
   *
   * False is the ordinary state of an installation with no model: the vault is
   * searched by its words, and the count of embedded chunks stays where it is.
   */
  const embedding = ref(false)
  /**
   * Books the vault holds and how far cutting has got through them.
   *
   * Cutting opens files, so it is counted in books; embedding works on what
   * cutting produced and is counted in chunks. `learning` says which is running.
   */
  const books = ref(0)
  const booksRead = ref(0)
  const learning = ref(false)
  /**
   * Whether the vault is still being read.
   *
   * Reading a book and embedding one change no file, so the vault is the only
   * one that knows there is more to come. This is it saying so.
   */
  const working = ref(false)
  /**
   * How fast the count now shown is moving, a second.
   *
   * Measured here because a rate needs a clock, and the words it turns into are
   * made by a model that has none. It is measured over the interval of the loop
   * that keeps the counts up to date, and only there: `ask` is called from three
   * places at three cadences, and a rate is a rate over one of them.
   *
   * Each phase counts a different thing — books opened, then windows embedded —
   * so a phase is timed on its own. Carrying a figure across is not a rate at
   * all.
   */
  const rate = ref(0)
  let counted: { phase: string; done: number; at: number } | null = null

  /**
   * Which question is the current one. Two answers can be in flight — a click
   * while a change is being followed — and without this the slower one wins
   * whatever was asked last.
   */
  let asked = 0
  let open = true
  /** Let go of every stream the window is listening to. */
  const listening = new AbortController()

  async function go(path: string) {
    const mine = ++asked
    try {
      const answer = await core.neighbourhood(path)
      if (mine !== asked) return
      if (!answer.focus?.path) {
        // The vault no longer holds it. What is on screen stays, and following
        // goes on, so putting the file back brings it straight back.
        notice.value = `${path} is not in the vault`
        return
      }
      notice.value = ''
      here.value = path
      neighbourhood.value = answer
    } catch (error) {
      if (mine !== asked) return
      notice.value = String(error)
    }
  }

  /** What the vault says about itself, which is not only its name. */
  async function ask() {
    const state = await core.state()
    name.value = state.name
    trouble.value = state.failed
    unwatched.value = state.unwatched
    chunks.value = Number(state.chunks)
    embedded.value = Number(state.embedded)
    reading.value = state.reading
    embedding.value = state.embedding
    books.value = Number(state.books)
    booksRead.value = Number(state.booksRead)
    learning.value = state.learning
    working.value = state.busy
    return state
  }

  /**
   * Take the rate from this reading of the count and the one before it, in the
   * phase it belongs to.
   *
   * A phase that has only been read once has no rate: one reading is a count,
   * and two are a rate.
   */
  function time() {
    const phase = learning.value ? 'learning' : 'reading'
    const done = learning.value ? embedded.value : booksRead.value
    const at = now()
    if (counted !== null && counted.phase === phase) {
      rate.value = rateOf({ done: counted.done, rate: rate.value }, done, (at - counted.at) / 1000)
    } else {
      rate.value = 0
    }
    counted = { phase, done, at }
  }

  /**
   * Whether the vault has work in hand.
   *
   * The vault says so; the counts do not. Cutting a library begins after the
   * notes are read, so a count of nothing is what the work looks like both
   * before it starts and while it runs.
   */
  const busy = () => working.value

  /**
   * Keep the counts up to date while the vault has work in hand.
   *
   * Reading books and embedding them change no file, so following the vault says
   * nothing about either. The loop asks on its own until the vault says it is
   * done, and one failed answer is skipped: the interval comes round again.
   *
   * One loop at a time, so every rate is taken over the loop's own interval.
   */
  let keeping = false

  async function keepUp() {
    if (keeping) return
    keeping = true
    // One interval stale is not worth a message. Two in a row is a vault that
    // has stopped answering, and the counts on screen are of a moment that has
    // passed.
    let missed = 0
    try {
      for (;;) {
        await wait(2000)
        if (!open) return
        try {
          await ask()
          time()
          if (missed > 0) {
            missed = 0
            notice.value = ''
          }
        } catch (error) {
          missed++
          if (missed > 1) notice.value = String(error)
        }
        if (!busy()) return
      }
    } finally {
      keeping = false
    }
  }

  /**
   * Follow the vault.
   *
   * Every change asks for the picture again, including changes to notes that
   * are nowhere on it: a link is written at one end and shows at both, so a
   * note edited somewhere else is exactly how a new parent arrives. What is on
   * screen cannot answer whether a change reaches it.
   *
   * The stream ends when the core stops or the connection goes, and it is taken
   * up again. A window that had stopped following would look exactly like one
   * that is up to date.
   */
  async function follow() {
    while (open) {
      try {
        for await (const change of core.changes(listening.signal)) {
          if (!open) return
          if (change.paths.length === 0 && !change.reload) continue
          if (here.value) {
            await go(here.value)
          } else {
            const note = await core.opening()
            if (note) await go(note.path)
          }
          try {
            await ask()
          } catch {
            // The stream stays open. What a change means is already drawn; the
            // counts come round with the next one.
          }
          void keepUp()
        }
      } catch (error) {
        if (!open) return
        notice.value = String(error)
      }
      await wait(1000)
      // Taken up again, so what was said about losing it no longer holds.
      if (open) notice.value = ''
    }
  }

  /**
   * Travels to whatever is asked for while the window is open — an agent
   * working the vault beside the person naming the note it is talking about.
   *
   * Taken up again the way following is, and for the same reason.
   */
  async function watch() {
    while (open) {
      try {
        for await (const wanted of core.focus(listening.signal)) {
          if (!open) return
          if (wanted.path) await go(wanted.path)
        }
      } catch (error) {
        if (!open) return
        notice.value = String(error)
      }
      await wait(1000)
    }
  }

  /** Waits for the scan to have stored something, then shows the first note. */
  async function start() {
    try {
      while (open) {
        const state = await ask()
        const note = await core.opening()
        if (!open) return
        if (note) {
          indexing.value = false
          await go(note.path)
          void follow()
          void watch()
          void keepUp()
          return
        }
        // A vault that could not be read is not an empty one, and neither is
        // one still being read. Both end the waiting; only one is empty.
        if (state.failed || state.ready) {
          indexing.value = false
          void follow()
          void watch()
          void keepUp()
          return
        }
        await wait(100)
      }
    } catch (error) {
      failure.value = String(error)
      indexing.value = false
    }
  }

  return {
    neighbourhood,
    here,
    name,
    indexing,
    failure,
    notice,
    trouble,
    unwatched,
    chunks,
    embedded,
    reading,
    embedding,
    books,
    booksRead,
    learning,
    working,
    rate,
    go,
    start,
    follow,
    watch,
    close: () => {
      open = false
      listening.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
