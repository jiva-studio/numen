/**
 * The notes the window has open, and what turns the state model.
 *
 * `tab.ts` decides; this carries out what it decides — reads, writes and the
 * interval — and holds the answers where the template can draw them.
 */
import { ref, type Ref } from 'vue'
import {
  opening,
  stateOf,
  tabAfter,
  waiting,
  type Effect,
  type Event,
  type Move,
  type Refusal,
  type Seen,
  type State,
  type Tab,
} from './tab'
import type { Answered, RefusalReason } from '../core'

/** One open note as the window draws it. */
export interface OpenNote {
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
  unreachable: 'the vault could not be reached, so this note was not written',
}

/** What a person is told and answers with when their tab was overtaken. */
export interface ConflictWords {
  readonly says: string
  readonly keep: string
  readonly take: string
}

const overtaken: ConflictWords = {
  says: 'this note changed on disk, and saving stopped',
  keep: 'keep mine',
  take: "take the file's",
}

/** What the window hands the store of open notes, beside the vault itself. */
export interface EditingOptions {
  /** How long the typing settles for, and how long a note may go unwritten. */
  limits?: typeof waiting
  /** What hears that a note on screen was replaced by what its file holds. */
  replaced?(path: string): void
  /** When it is now. A step of the undo is stamped with it. */
  now?(): number
}

export function editing(core: Notes, how: EditingOptions = {}) {
  const limits = how.limits ?? waiting
  const replaced = how.replaced ?? (() => {})
  const now = how.now ?? (() => Date.now())
  /** Every open note, under an identity its caller mints and this never reads into. */
  const tabs = ref(new Map<string, Tab>())
  /** The interval each tab is waiting on, so arming again replaces it. */
  const timers = new Map<string, ReturnType<typeof setTimeout>>()
  /** The bodies the editors are showing, which Vue writes into as a person types. */
  const bodies = ref(new Map<string, string>())

  /** A note opened under an identity, on the file it opens at. */
  const open = (id: string, path: string = id): void => {
    if (tabs.value.has(id)) return
    carry(id, opening(path))
  }

  /**
   * A note the window is no longer showing.
   *
   * A tab that cannot be written is held once and says why. Asked a second time
   * it goes: the person has been told, and insisting is theirs to do.
   */
  const shut = (id: string): Promise<boolean> =>
    new Promise((done) => {
      if (!tabs.value.has(id)) return done(true)
      if (told.has(id)) {
        forget(id)
        return done(true)
      }
      closing.set(id, done)
      turn(id, { kind: 'closing' })
    })

  /** The tabs that were held once, and go the next time they are asked. */
  const told = new Set<string>()

  /** Who is waiting for a tab to finish going. */
  const closing = new Map<string, (gone: boolean) => void>()

  /** Who is waiting for a note to have nothing more on its way to the file. */
  const settling = new Map<string, () => void>()

  /**
   * A note whose file is about to be renamed or removed. Its interval goes and
   * what it owes reaches the path it still stands at, and it answers once
   * nothing more of it is on its way there.
   */
  const settles = (id: string): Promise<void> =>
    new Promise((done) => {
      if (!tabs.value.has(id)) return done()
      settling.set(id, done)
      turn(id, { kind: 'settling' })
    })

  /**
   * The file an open note stands at now, under the identity it opened under. A
   * note that moved is followed by the tab that opened it.
   */
  const where = (id: string): string => tabs.value.get(id)?.path ?? id

  /** Whether the window has this note open at all. */
  const has = (id: string): boolean => tabs.value.has(id)

  /**
   * The file this tab last read or wrote, which a caller writing to the same
   * note beside the tab presents. Empty where no file has been read.
   */
  const at = (id: string): string => tabs.value.get(id)?.at ?? ''

  /** The person typed. */
  const typed = (id: string, body: string): void => {
    bodies.value.set(id, body)
    turn(id, { kind: 'typed', body, at: now() })
  }

  /** The vault changed. Every open note hears it and decides for itself. */
  const changed = (paths: readonly string[], renamed: readonly Move[] = []): void => {
    for (const id of [...tabs.value.keys()]) turn(id, { kind: 'changed', paths, renamed })
  }

  /** A save asked for now. */
  const save = (id: string): void => turn(id, { kind: 'saving' })

  /** The person keeps what they have written. */
  const keep = (id: string): void => turn(id, { kind: 'keeping' })

  /** The person takes what the file holds. */
  const take = (id: string): void => turn(id, { kind: 'taking' })

  const shown = (id: string): OpenNote => {
    const tab = tabs.value.get(id)
    return {
      path: tab?.path ?? id,
      body: bodies.value.get(id) ?? '',
      state: tab ? stateOf(tab) : 'loading',
      refusal: tab?.refused ?? null,
    }
  }

  /** Everything open, for a caller that draws them all. */
  const all = (): readonly string[] => [...tabs.value.keys()]

  const sayingOf = (id: string): string => {
    const refusal = tabs.value.get(id)?.refused
    return refusal ? words[refusal] : ''
  }

  /** What an overtaken note puts to the person, for the window to draw. */
  const overtakenOf = (id: string): ConflictWords | null => {
    const tab = tabs.value.get(id)
    return tab && stateOf(tab) === 'overtaken' ? overtaken : null
  }

  function turn(id: string, event: Event): void {
    const tab = tabs.value.get(id)
    if (!tab) return
    carry(id, tabAfter(tab, event, limits))
  }

  function carry(id: string, next: ReturnType<typeof tabAfter>): void {
    tabs.value.set(id, next.tab)
    for (const effect of next.effects) act(id, effect)

    // Held with nothing on its way to the file: the tab stays, and whoever
    // asked for it to close hears that it did not.
    const held = next.effects.some((effect) => effect.kind === 'hold')
    const writing = next.effects.some((effect) => effect.kind === 'write')
    if (held && !writing) closing.get(id)?.(false)

    // A note settling is done the moment nothing of it is on its way.
    if (!next.tab.flight) settled(id)
  }

  function settled(id: string): void {
    settling.get(id)?.()
    settling.delete(id)
  }

  function act(id: string, effect: Effect): void {
    switch (effect.kind) {
      case 'read':
        void read(id, effect.path, effect.generation)
        return
      case 'write':
        void write(id, effect.path, effect.body, effect.seen)
        return
      case 'arm':
        arm(id, effect.after)
        return
      case 'disarm':
        clearTimeout(timers.get(id))
        timers.delete(id)
        return
      case 'replace':
        bodies.value.set(id, effect.body)
        replaced(where(id))
        return
      case 'hold':
        return
      case 'say':
        // The tab has said why for as long as it has been open. Held once here,
        // so a person who meant it can ask again.
        told.add(id)
        return
      case 'close':
        forget(id)
        return
    }
  }

  function arm(id: string, after: number): void {
    clearTimeout(timers.get(id))
    timers.set(
      id,
      setTimeout(() => {
        timers.delete(id)
        turn(id, { kind: 'fired' })
      }, after),
    )
  }

  async function read(id: string, path: string, generation: number): Promise<void> {
    let answered: Answered & { at?: string }
    try {
      answered = await core.read(path)
    } catch {
      // The core did not answer. What is on screen is still here, and asking
      // again is what finds out whether the vault came back.
      turn(id, { kind: 'read', generation, answer: { kind: 'refused', refusal: 'unreachable' } })
      return
    }
    turn(id, {
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

  async function write(id: string, path: string, body: string, seen: Seen | null): Promise<void> {
    let answered: Answered & { at?: string; changed?: boolean }
    try {
      answered = await core.write(path, body, seen)
    } catch {
      turn(id, { kind: 'written', answer: { kind: 'refused', refusal: 'unreachable' } })
      return
    }
    turn(id, {
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
    if (closing.has(id)) turn(id, { kind: 'closing' })
  }

  function forget(id: string): void {
    clearTimeout(timers.get(id))
    timers.delete(id)
    tabs.value.delete(id)
    bodies.value.delete(id)
    closing.get(id)?.(true)
    closing.delete(id)
    settled(id)
    told.delete(id)
  }

  /** Every tab writes what it owes, for a window that is going. */
  const flush = async (): Promise<void> => {
    await Promise.all(all().map(shut))
  }

  return {
    open,
    shut,
    settles,
    where,
    has,
    at,
    typed,
    changed,
    save,
    keep,
    take,
    shown,
    all,
    saying: sayingOf,
    overtaken: overtakenOf,
    flush,
    tabs: tabs as Ref<Map<string, Tab>>,
  }
}

/**
 * A refusal in the words the tab model uses.
 *
 * A write never answers `missing`, because a save creates the file it does not
 * find, and a read answers it as its own kind. `occupied` is a note being made
 * and `unnameable` a note being named, neither of which is something a tab does.
 * A deck is refused by what it is and by the size it is read up to, and the tab
 * that holds one says which in its own words.
 */
const refusalOf = (from: RefusalReason): Refusal => {
  if (from === 'deckTooLarge') return 'tooLarge'
  if (from === 'notAStencil' || from === 'notADeck' || from === 'notAPreset') return 'notANote'
  return from === 'missing' || from === 'occupied' || from === 'unnameable' ? 'unreadable' : from
}
