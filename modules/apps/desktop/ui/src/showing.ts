/**
 * What the window is showing, and the rules for changing it.
 *
 * Apart from the template because these are the rules that decide whether the
 * window keeps up with the vault, and a rule inside a component is a rule that
 * is only exercised by looking at the screen.
 */
import { computed, ref, shallowRef } from 'vue'
import { rateOf } from '@numen/ui'
import type { PlexRelatedSeat } from '@numen/ui'
import type { Neighbourhood } from './plex'
import { standing, type Standing } from './standing'
import type { Said } from './drawing'
import type { Went } from './tab'

/** Everything the window asks of the core, and nothing about how it is drawn. */
/**
 * One piece of work the application is doing behind the window.
 *
 * Every kind of work is one of these, which is what keeps the window from
 * growing a branch per kind: it draws the list it is given.
 */
export interface Task {
  /** What the work is called, so that the same work reported again replaces it. */
  readonly id: string
  /** The work, in the words to show, and what it is on. */
  readonly doing: string
  readonly about: string
  /** How far it has got, where there is a total to count against. */
  readonly done: number
  readonly total: number
  /** Why it stopped, when it stopped badly. */
  readonly failed: string
}

export interface Core {
  neighbourhood(path: string): Promise<Neighbourhood>
  opening(): Promise<{ path: string } | null>
  state(): Promise<{
    name: string
    ready: boolean
    failed: string
    unwatched: string
    unreachable: string
    /** Spans of text the index holds, and how many of them carry a vector. */
    chunks: bigint
    embedded: bigint
    /** What the pass now running found to do, and how much of it is done. */
    owing: bigint
    made: bigint
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
  changes(signal: AbortSignal): AsyncIterable<{
    paths: string[]
    reload: boolean
    renamed: readonly Went[]
  }>
  /** A change being made to a note's prose, reported while it is being made. */
  editing(signal: AbortSignal): AsyncIterable<Said>
  /**
   * Everything the application is doing behind the window, for as long as the
   * window listens.
   *
   * The whole list arrives whenever any of it changes, and the first arrives at
   * once. It is a stream because work can begin without the window asking for
   * it: an agent is told to read a document, and this is where the person
   * watching sees it happen.
   */
  tasks(signal: AbortSignal): AsyncIterable<readonly Task[]>
  /** The notes something else asked to be put in front of the person. */
  focus(signal: AbortSignal): AsyncIterable<{ path: string }>
  /** The prose of a note, below its frontmatter, and the file it came out of. */
  read(path: string): Promise<Answered & { at?: string }>
  /**
   * Prose into a note, keeping the frontmatter the file has when it lands.
   *
   * Seen is what a read gave this caller. Prose on disk that the caller never
   * saw comes back as changed, and nothing is written. Nothing seen writes
   * over whatever is there.
   */
  write(
    path: string,
    body: string,
    seen: { prose: string; at: string } | null,
  ): Promise<Answered & { at?: string; changed?: boolean }>
  /** A note made, named after the title it is given and joined as it is written. */
  create(note: NewNote): Promise<Made>
  /**
   * A relationship written into one note. The note at the other end is left
   * alone: a link is one end's account of a relationship.
   */
  join(path: string, link: NewLink): Promise<Refused | null>
  /**
   * The window going, for as long as the client listens. The stream opens with
   * the token this client answers under.
   */
  quitting(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  /** Everything this client owed has been written. */
  flushed(token: string, owed?: 'written' | 'asking'): Promise<void>
}

/**
 * What a read or a write came back with. A refusal carries no body, and the
 * words for one belong to whatever shows it.
 */
export interface Answered {
  body: string
  refusal: Refused | null
}

export type Refused =
  | 'missing'
  | 'notANote'
  | 'notText'
  | 'tooLarge'
  | 'bodyRefused'
  | 'unreadable'
  | 'occupied'

/** A note to make: what it is called, where it goes, and what it arrives joined to. */
export interface NewNote {
  title: string
  /** Where in the vault it goes, relative to the root. Empty is the root. */
  folder: string
  links: readonly NewLink[]
}

/**
 * One relationship as the note it is written in declares it: the note at the
 * other end, by the path it is filed under, and where that note sits seen from
 * this one.
 */
export interface NewLink {
  to: string
  seat: PlexRelatedSeat
  /** What the person calls this relationship, when they call it anything. */
  label?: string
}

/** What making a note came back with. */
export interface Made {
  /** Where the note is filed. Empty when nothing was made. */
  path: string
  refusal: Refused | null
}

/** One plex as its tab holds it: where it stands, and letting go of it. */
export interface Plexed extends Standing {
  /** The person is looking at this plex. */
  looking(): void
  /** The tab has closed, and nothing is asked for this plex again. */
  close(): void
}

/**
 * The vault as the whole window reads it, and the plexes it keeps up to date.
 *
 * One window reads the vault once: one stream of changes, one stream of edits,
 * one stream of notes asked for, and one set of counts. A plex tab holds where
 * it is standing and nothing more, and hears from here when to ask again.
 */
export function showing(
  core: Core,
  wait: (ms: number) => Promise<unknown> = sleep,
  now: () => number = () => Date.now(),
  /**
   * What else hears about a change. A change carrying no paths names nothing:
   * everything showing the vault reads again.
   */
  told: (paths: readonly string[], renamed?: readonly Went[]) => void = () => {},
  /** What hears about a change to a note while it is being made. */
  drawing: (said: Said) => void = () => {},
  /** What opens a plex on a note, for a window with none open to show it in. */
  shows: (path: string) => void = () => {},
) {
  const name = ref('')
  const indexing = ref(true)
  /** The core could not be reached: nothing else in the window is true. */
  const failure = ref('')
  /** What the window itself lost touch with, said until it has it back. */
  const lost = ref('')
  const trouble = ref('')
  const unwatched = ref('')
  /** Why an agent cannot be reached, said in the panel that would have asked it. */
  const unreachable = ref('')
  /** Whether the vault holds a note to show at all. */
  const holds = ref(false)
  /** The note the vault opens with, which is where a plex standing nowhere goes. */
  const opening = ref('')
  /**
   * How far reading the vault for meaning has got.
   *
   * Cutting finishes long before embedding does, so the pair is what says how
   * far there is to go. `reading` names what is being read, and is empty
   * between sources as well as after the last one.
   */
  const chunks = ref(0)
  const embedded = ref(0)
  /**
   * What the pass now running found to do, and how much of it is done.
   *
   * This is the work in hand. The counts above are the whole of what the vault
   * holds, and somebody who changed one note is waiting on one chunk.
   */
  const owing = ref(0)
  const made = ref(0)
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
  /**
   * Everything the application is doing behind the window.
   *
   * It arrives whole and is shown whole. A new kind of work is an entry here
   * rather than another count to read out of the state and another branch in
   * what draws it.
   */
  const tasks = ref<readonly Task[]>([])
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
   * The plexes open in the window, each in a tab of its own, in the order the
   * person was last in them. The last one is the plex in front.
   */
  const plexes = shallowRef<readonly Standing[]>([])
  /**
   * The plex the person is looking at. A note asked for from outside the window
   * is put in front of it.
   */
  const ahead = computed<Standing | null>(() => plexes.value.at(-1) ?? null)
  /** The note the person is looking at, which is what a question is about. */
  const looking = computed(() => ahead.value?.here.value ?? '')
  /**
   * The one thing the window says while it still works: what it lost touch
   * with, or what the plex in front could not show.
   */
  const warning = computed(() => lost.value || ahead.value?.trouble.value || '')

  let open = true
  /** Let go of every stream the window is listening to. */
  const listening = new AbortController()

  /**
   * A plex of its own, followed for as long as its tab is open. It opens where
   * it is told to, where the person is looking, or on the note the vault opens
   * with. Closing it is what stops it being asked for.
   */
  function plex(at = ''): Plexed {
    const view = standing(core)
    const from = at || ahead.value?.here.value || opening.value
    plexes.value = [...plexes.value, view]
    if (from) void view.go(from)

    return {
      ...view,
      /** The person is looking at this plex. */
      looking: () => {
        if (!plexes.value.includes(view)) return
        plexes.value = [...plexes.value.filter((one) => one !== view), view]
      },
      close: () => {
        view.close()
        plexes.value = plexes.value.filter((one) => one !== view)
      },
    }
  }

  /**
   * One plex asks for its picture again. A plex standing nowhere is given the
   * note the vault opens with.
   */
  async function again(view: Standing) {
    const path = view.here.value || opening.value
    if (path) await view.go(path)
  }

  /** The note the vault opens with, and whether it holds one at all. */
  async function first() {
    const note = await core.opening()
    holds.value = note !== null
    opening.value = note?.path ?? ''
    return opening.value
  }

  /** What the vault says about itself, which is not only its name. */
  async function ask() {
    const state = await core.state()
    name.value = state.name
    trouble.value = state.failed
    unwatched.value = state.unwatched
    unreachable.value = state.unreachable
    chunks.value = Number(state.chunks)
    embedded.value = Number(state.embedded)
    owing.value = Number(state.owing)
    made.value = Number(state.made)
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
    const done = learning.value ? made.value : booksRead.value
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
   *
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
            lost.value = ''
          }
        } catch (error) {
          missed++
          if (missed > 1) lost.value = String(error)
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
          if (change.paths.length === 0 && !change.reload && change.renamed.length === 0) continue
          told(change.reload ? [] : change.paths, change.renamed)
          try {
            // Asked once for the window, and only while a plex has nowhere to
            // stand.
            if (plexes.value.some((view) => !view.here.value)) await first()
          } catch {
            // The next change asks again.
          }
          await Promise.all(plexes.value.map((view) => again(view)))
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
        lost.value = String(error)
      }
      await wait(1000)
      // Taken up again, so what was said about losing it no longer holds.
      if (open) lost.value = ''
    }
  }

  /** Follows the changes being made to notes, and takes the stream up again. */
  async function draw() {
    while (open) {
      try {
        for await (const said of core.editing(listening.signal)) {
          if (!open) return
          // The stream opens by saying nothing, which is how an open one is
          // told from one that never opened.
          if (said.path) drawing(said)
        }
      } catch (error) {
        if (!open) return
        lost.value = String(error)
      }
      await wait(1000)
    }
  }

  /**
   * A note put in front of the person: the plex they are looking at travels
   * there, and a window holding no plex at all opens one on it.
   */
  async function travel(path: string) {
    if (ahead.value) await ahead.value.go(path)
    else shows(path)
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
          if (!wanted.path) continue
          await travel(wanted.path)
        }
      } catch (error) {
        if (!open) return
        lost.value = String(error)
      }
      await wait(1000)
    }
  }

  /**
   * Keeps the list of what is being done up to date while the window is open.
   *
   * Nothing is asked for on a timer. Work begins without the window: an agent is
   * told to read a document, and the list says so the moment it starts.
   *
   * Taken up again the way following is, and for the same reason.
   */
  async function attend() {
    while (open) {
      try {
        for await (const list of core.tasks(listening.signal)) {
          if (!open) return
          tasks.value = list
        }
      } catch (error) {
        if (!open) return
        lost.value = String(error)
      }
      await wait(1000)
      // Taken up again, so what was said about losing it no longer holds.
      if (open) lost.value = ''
    }
  }

  /** Waits for the scan to have stored something, then shows the first note. */
  async function start() {
    try {
      while (open) {
        const state = await ask()
        const note = await first()
        if (!open) return
        if (note) {
          indexing.value = false
          await Promise.all(plexes.value.map((view) => again(view)))
          void follow()
          void watch()
          void draw()
          void attend()
          void keepUp()
          return
        }
        // A vault that could not be read is not an empty one, and neither is
        // one still being read. Both end the waiting; only one is empty.
        if (state.failed || state.ready) {
          indexing.value = false
          void follow()
          void watch()
          void draw()
          void attend()
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
    name,
    indexing,
    failure,
    warning,
    trouble,
    unwatched,
    unreachable,
    holds,
    looking,
    travel,
    chunks,
    embedded,
    owing,
    made,
    reading,
    embedding,
    books,
    booksRead,
    tasks,
    learning,
    working,
    rate,
    plex,
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
