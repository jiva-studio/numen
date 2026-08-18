/**
 * The notes the window has open, and what turns the state model.
 *
 * `tab.ts` decides; this carries out what it decides — reads, writes and the
 * interval — and holds the answers where the template can draw them.
 */
import { ref, type Ref } from 'vue'
import { opening, stateOf, tabAfter, waiting, type Effect, type Event, type Refusal, type Seen, type State, type Tab } from './tab'
import type { Answered, Refused } from './showing'

/** One open note as the window draws it. */
export interface Editing {
  readonly path: string
  readonly body: string
  readonly state: State
  readonly refusal: Refusal | null
}

/**
 * The core as a tab reads and writes through it. A read answers with the
 * fingerprint it read at, and a write presents the one the tab last saw and
 * answers whether the file carries another.
 */
export interface Notes {
  read(path: string): Promise<Answered & { at?: string }>
  write(
    path: string,
    body: string,
    seen: Seen | null,
  ): Promise<Answered & { at?: string; changed?: boolean }>
}

/** What a person is shown for each refusal. */
const words: Record<Refusal, string> = {
  notANote: 'this file is not a note',
  notText: 'this file is not text',
  tooLarge: 'this note is longer than the editor holds',
  bodyRefused: 'a note begins below its frontmatter, and this text begins with one',
  unreadable: 'the frontmatter of this note cannot be read',
}

/** What a person is told and answers with when their tab was overtaken. */
export interface Overtaken {
  readonly says: string
  readonly keep: string
  readonly take: string
}

const overtaken: Overtaken = {
  says: 'this note changed on disk, and saving stopped',
  keep: 'keep mine',
  take: "take the file's",
}

export function editing(core: Notes, limits = waiting) {
  const tabs = ref(new Map<string, Tab>())
  /** The interval each tab is waiting on, so arming again replaces it. */
  const timers = new Map<string, ReturnType<typeof setTimeout>>()
  /** The bodies the editors are showing, which Vue writes into as a person types. */
  const bodies = ref(new Map<string, string>())

  const open = (path: string): void => {
    if (tabs.value.has(path)) return
    carry(path, opening(path))
  }

  /** A note the window is no longer showing. A close writes what is owed. */
  const shut = (path: string): Promise<void> =>
    new Promise((done) => {
      if (!tabs.value.has(path)) return done()
      closing.set(path, done)
      turn(path, { kind: 'closing' })
    })

  /** Who is waiting for a tab to finish going. */
  const closing = new Map<string, () => void>()
  /** The last refusal a person was told about, which outlives its tab. */
  const said = ref('')

  /** The person typed. */
  const typed = (path: string, body: string): void => {
    bodies.value.set(path, body)
    turn(path, { kind: 'typed', body, at: Date.now() })
  }

  /** The vault changed. Every open note hears it and decides for itself. */
  const changed = (paths: readonly string[]): void => {
    for (const path of [...tabs.value.keys()]) turn(path, { kind: 'changed', paths })
  }

  /** A save asked for now. */
  const save = (path: string): void => turn(path, { kind: 'saving' })

  /** The person keeps what they have written. */
  const keep = (path: string): void => turn(path, { kind: 'keeping' })

  /** The person takes what the file holds. */
  const take = (path: string): void => turn(path, { kind: 'taking' })

  const shown = (path: string): Editing => {
    const tab = tabs.value.get(path)
    return {
      path,
      body: bodies.value.get(path) ?? '',
      state: tab ? stateOf(tab) : 'loading',
      refusal: tab?.refused ?? null,
    }
  }

  /** Everything open, for a caller that draws them all. */
  const all = (): readonly string[] => [...tabs.value.keys()]

  const sayingOf = (path: string): string => {
    const refusal = tabs.value.get(path)?.refused
    return refusal ? words[refusal] : ''
  }

  /** What an overtaken note puts to the person, for the window to draw. */
  const overtakenOf = (path: string): Overtaken | null => {
    const tab = tabs.value.get(path)
    return tab && stateOf(tab) === 'overtaken' ? overtaken : null
  }

  function turn(path: string, event: Event): void {
    const tab = tabs.value.get(path)
    if (!tab) return
    carry(path, tabAfter(tab, event, limits))
  }

  function carry(path: string, next: ReturnType<typeof tabAfter>): void {
    tabs.value.set(path, next.tab)
    tabs.value = new Map(tabs.value)
    for (const effect of next.effects) act(path, effect)
  }

  function act(path: string, effect: Effect): void {
    switch (effect.kind) {
      case 'read':
        void read(path, effect.generation)
        return
      case 'write':
        void write(path, effect.body, effect.seen)
        return
      case 'arm':
        arm(path, effect.after)
        return
      case 'replace':
        bodies.value.set(path, effect.body)
        bodies.value = new Map(bodies.value)
        return
      case 'hold':
        return
      case 'say':
        // Said where it outlives the tab: the words come from the model and the
        // tab is about to go with them.
        said.value = words[effect.refusal]
        return
      case 'close':
        forget(path)
        return
    }
  }

  function arm(path: string, after: number): void {
    clearTimeout(timers.get(path))
    timers.set(
      path,
      setTimeout(() => {
        timers.delete(path)
        turn(path, { kind: 'fired' })
      }, after),
    )
  }

  async function read(path: string, generation: number): Promise<void> {
    let answered: Answered & { at?: string }
    try {
      answered = await core.read(path)
    } catch {
      answered = { body: '', refusal: 'unreadable' }
    }
    turn(path, {
      kind: 'read',
      generation,
      answer:
        answered.refusal === null
          ? { kind: 'body', body: answered.body, at: answered.at ?? '' }
          : answered.refusal === 'missing'
            ? { kind: 'missing' }
            : { kind: 'refused', refusal: refusalOf(answered.refusal) },
    })
  }

  async function write(path: string, body: string, seen: Seen | null): Promise<void> {
    let answered: Answered & { at?: string; changed?: boolean }
    try {
      answered = await core.write(path, body, seen)
    } catch {
      answered = { body: '', refusal: 'unreadable' }
    }
    turn(path, {
      kind: 'written',
      // A file that moved past what the tab read is put to the person.
      answer: answered.changed
        ? { kind: 'changed' }
        : answered.refusal === null
          ? { kind: 'ok', at: answered.at ?? '' }
          : { kind: 'refused', refusal: refusalOf(answered.refusal) },
    })
    // A tab held for its write is asked about again, so the model decides what
    // the answer means for a close it already agreed to.
    if (closing.has(path)) turn(path, { kind: 'closing' })
  }

  function forget(path: string): void {
    clearTimeout(timers.get(path))
    timers.delete(path)
    tabs.value.delete(path)
    tabs.value = new Map(tabs.value)
    bodies.value.delete(path)
    bodies.value = new Map(bodies.value)
    closing.get(path)?.()
    closing.delete(path)
  }

  /** Every tab writes what it owes, for a window that is going. */
  const flush = async (): Promise<void> => {
    await Promise.all(all().map(shut))
  }

  return {
    open,
    shut,
    typed,
    changed,
    save,
    keep,
    take,
    shown,
    all,
    saying: sayingOf,
    overtaken: overtakenOf,
    said,
    flush,
    tabs: tabs as Ref<Map<string, Tab>>,
  }
}

/**
 * A refusal in the words the tab model uses.
 *
 * A write never answers `missing`, because a save creates the file it does not
 * find, and a read answers it as its own kind. `occupied` is a note being made,
 * which is not something a tab does.
 */
const refusalOf = (from: Refused): Refusal =>
  from === 'missing' || from === 'occupied' ? 'unreadable' : from
