/**
 * What the window knows about the notes it has open.
 *
 * The notes are read and written by one store for the whole window: what is
 * unsaved is answered to the quit as one question, and a change to the vault
 * reaches all of them at once. What one tab of one note holds is made here
 * from that store. What a note is called is in `naming.ts`, and the keyboard
 * it is owed is in `entering.ts`.
 */
import type { PlexShowing } from '@numen/ui'
import type { Host, Kind } from '../windowing'
import { NOTE } from '../workspace'
import type { Change } from './drawing'
import type { drawn } from './drawn'
import type { Editing, editing } from './editing'
import { entering, ITSELF } from './entering'
import { naming, type Called } from './naming'
import NoteTab from './NoteTab.vue'
import { markOf } from './tab'
import { WORDS as words } from './words'

/** The notes of the whole window, read and written by one store. */
type Notes = ReturnType<typeof editing>
/** What is being typed into each note now, as the editor draws it. */
type Drawings = ReturnType<typeof drawn>

export type { Called }

/** What the notes of a window ask of the vault they are read from. */
export interface Noting {
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
  shuts(id: string): void
}

export function noting(vault: Called, notes: Notes, drawings: Drawings, host: Host, deps: Noting) {
  const names = naming(vault, notes)
  const keyboard = entering()

  /**
   * A note opened. It is owed its keyboard from here until an editor has taken
   * it, on the line it was told to stand on before it was drawn.
   */
  const opens = (path: string, line = ITSELF) => {
    notes.open(path)
    keyboard.owes(path, line)
    return held(path)
  }

  /**
   * A note put in front of the person, under the name it is called by and in a
   * tab of its own. It takes the keyboard, opened now or already open.
   */
  const shows = (path: string, title = '', showing: PlexShowing = 'here') => {
    if (title) names.calls(path, title)
    void (showing === 'beside' ? host.beside(NOTE, path) : host.opens(NOTE, path))
    keyboard.owes(path)
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
    drew: (editor: unknown) => keyboard.drew(path, editor),
    measure: () => keyboard.measure(path),
    /** The tab stands until the note says the write is done, and goes then. */
    shuts: (id: string) => {
      keyboard.drops(path)
      drawings.shut(path)
      void notes.shut(path).then((gone) => {
        if (!gone) return
        names.forgets(path)
        host.closes(id)
      })
    },
  })

  /**
   * A note tab as the window keeps it. A note is its own tab, filed under the
   * path it is written at.
   */
  const kind: Kind<Held> = {
    kind: NOTE,
    opens: (path) => opens(path),
    called: (held) => names.called(held.path),
    marked: (held) => markOf(notes.shown(held.path).state),
    draws: NoteTab,
    identity: (path) => path,
    makes: () => deps.makes(),
    shown: (held) => held.measure(),
    shuts: (held, id) => {
      held.shuts(id)
      return false
    },
    // What an open note owes at the quit is written by the quit, which the
    // window waits for.
    gone: () => {},
    offers: words.newNote,
  }

  return {
    kind,
    held,
    opens,
    shows,
    titles: names.titles,
    calls: names.calls,
    called: names.called,
    entersAt: keyboard.owes,
  }
}
