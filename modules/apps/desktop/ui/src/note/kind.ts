/**
 * What the window knows about the notes it has open.
 *
 * The notes are read and written by one store for the whole window: what is
 * unsaved is answered to the quit as one question, and a change to the vault
 * reaches all of them at once. What one tab of one note holds is made here
 * from that store, and everything a tab decides for itself is decided here.
 */
import { nextTick, ref, watch } from 'vue'
import type { Change } from '../drawing'
import type { drawn } from '../drawn'
import type { Editing, editing } from '../editing'
import { markOf } from '../tab'
import type { Kind } from '../windowing'
import { WORDS as words } from '../words'
import { NOTE } from '../workspace'
import NoteTab from './NoteTab.vue'

/** The notes of the whole window, read and written by one store. */
type Notes = ReturnType<typeof editing>
/** What is being typed into each note now, as the editor draws it. */
type Drawings = ReturnType<typeof drawn>

/** What a note is asked to be called, as the vault last said it. */
export interface Called {
  neighbourhood(path: string): Promise<{ focus?: { title?: string } | undefined }>
}

/** What the editor of a note answers once it is drawn. */
export interface Drawn {
  focus(): boolean
  measure(): void
  reveal(line: number): boolean
}

/** What a note tab asks of the window it is drawn in. */
export interface Noting {
  /** The tab of a note that has written what it owed, going now. */
  closes(path: string): void
  /** A note made to fill a tab that was told to hold one; where it is filed. */
  makes(): Promise<string>
}

/** What one note tab holds: its text, and the answers a person gives it. */
export interface Held {
  readonly path: string
  /** The note as the window draws it: the body, and the state it is in. */
  shown(): Editing
  /** What could not be read or written, in words a person reads. */
  saying(): string
  /** What arrived from elsewhere, for the editor to take into what is typed. */
  change(): Change | null
  typed(body: string): void
  save(): void
  /** The person keeps what they have written, over whatever the file holds. */
  keep(): void
  /** The person takes what the file holds. */
  take(): void
  /** The editor of this note, as it is drawn and as it goes. */
  drew(editor: unknown): void
  measure(): void
  /** The tab is closing, and what is unwritten goes to the file first. */
  shuts(): void
}

/**
 * The line a note is opened on where none was asked for: the note itself, and
 * no line in particular.
 */
const ITSELF = -1

export function noting(vault: Called, notes: Notes, drawings: Drawings, deps: Noting) {
  /** What each note is called, as the vault last said it. */
  const titles = ref<ReadonlyMap<string, string>>(new Map())

  const calls = (path: string, name: string): void => {
    titles.value = new Map(titles.value).set(path, name)
  }

  const forgets = (path: string): void => {
    const rest = new Map(titles.value)
    rest.delete(path)
    titles.value = rest
  }

  /**
   * A note is called what the vault calls it. A heading written into a note is
   * that note's title, so a tab is asked what it is called again once what was
   * typed into it has landed.
   *
   * A vault that cannot answer leaves the tab under the name it had.
   */
  const asks = async (path: string): Promise<void> => {
    try {
      const said = (await vault.neighbourhood(path)).focus?.title
      if (said) calls(path, said)
    } catch {
      return
    }
  }

  watch(
    () => notes.all().filter((path) => notes.shown(path).state === 'clean'),
    (settled, before) => {
      for (const path of settled) {
        if (!before?.includes(path)) void asks(path)
      }
    },
    { deep: true },
  )

  /**
   * The notes owed their keyboard, and the line each was asked to open on,
   * until there is an editor to hand it to.
   */
  const owed = new Map<string, number>()

  /** The editor of each open note, for as long as its tab is drawn. */
  const editors = new Map<string, Drawn>()

  /**
   * A note opened takes the keyboard once it is on screen, on the line it was
   * asked for when it was asked for one.
   *
   * A tab already showing has an editor now; a tab that has to be drawn first
   * says so when it appears, and an editor says so when it is built.
   */
  const enters = (path: string) => {
    const line = owed.get(path)
    const editor = editors.get(path)
    if (line === undefined || !editor) return
    // An editor is registered as it is drawn, a moment before it exists to take
    // anything. The note is owed its keyboard until one has.
    if (line >= 0 ? editor.reveal(line) : editor.focus()) owed.delete(path)
  }

  /**
   * A note opened. It is owed its keyboard from here until an editor has taken
   * it, on the line it was told to stand on before it was drawn.
   */
  const opens = (path: string, line = ITSELF) => {
    notes.open(path)
    owed.set(path, line)
    void nextTick(() => enters(path))
    return held(path)
  }

  /**
   * The line an open note is to stand on, asked for after it was opened. A
   * note asked for again takes the keyboard again, wherever it already stands.
   */
  const entersAt = (path: string, line = ITSELF) => {
    owed.set(path, line)
    void nextTick(() => enters(path))
  }

  /** What one tab of a note holds. */
  const held = (path: string): Held => ({
    path,
    shown: () => notes.shown(path),
    saying: () => notes.saying(path),
    change: () => drawings.shown(path),
    typed: (body: string) => notes.typed(path, body),
    save: () => notes.save(path),
    keep: () => notes.keep(path),
    take: () => notes.take(path),
    /** The editor of this note, as it is drawn and as it goes. */
    drew: (editor: unknown) => {
      if (!editor) {
        editors.delete(path)
        return
      }
      editors.set(path, editor as Drawn)
      void nextTick(() => enters(path))
    },
    measure: () => {
      editors.get(path)?.measure()
      enters(path)
    },
    /**
     * The tab is closing. What is unwritten goes to the file first, so the tab
     * stands until the note says it is done and the window closes it then.
     */
    shuts: () => {
      owed.delete(path)
      drawings.shut(path)
      void notes.shut(path).then((gone) => {
        if (!gone) return
        forgets(path)
        deps.closes(path)
      })
    },
  })

  /** What the tab of a note is called, and the word it carries. */
  const called = (path: string): string => titles.value.get(path) ?? path
  const marked = (path: string): string | undefined => markOf(notes.shown(path).state)

  /**
   * A note tab as the window keeps it. A note is its own tab, filed under the
   * path it is written at.
   */
  const kind: Kind<Held> = {
    kind: NOTE,
    opens: (path) => opens(path),
    called: (held) => called(held.path),
    marked: (held) => marked(held.path),
    draws: NoteTab,
    identity: (path) => path,
    makes: () => deps.makes(),
    shown: (held) => held.measure(),
    shuts: (held) => {
      held.shuts()
      return false
    },
    // What an open note owes at the quit is written by the quit, which the
    // window waits for.
    gone: () => {},
    offers: words.newNote,
  }

  return { kind, titles, calls, forgets, opens, entersAt, held, called, marked }
}
