/**
 * What the window knows about the notes it has open.
 *
 * The notes are read and written by one store for the whole window: what is
 * unsaved is answered to the quit as one question, and a change to the vault
 * reaches all of them at once. What one tab of one note holds is made here
 * from that store. What a note is called is in `naming.ts`, and the keyboard
 * it is owed is in `entering.ts`.
 */
import { computed } from 'vue'
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
  /** The identity this note opened under, which its tab keeps wherever it goes. */
  readonly id: string
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
   * Every open note under the file it stands at now, against the identity it
   * opened under. A note that moved is looked up here to reach the tab already
   * holding it.
   */
  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(notes.all().map((id) => [notes.where(id), id])),
  )

  /**
   * The identity minted for a note asked for by name, until its tab opens under
   * it. A note is named and shown in two steps, and both name the same tab.
   */
  const minting = new Map<string, string>()
  const minted = new Map<string, string>()

  /** The identity of the tab standing at a file, minted where none stands there. */
  const mints = (path: string): string => {
    const standing = tabbed.value.get(path) ?? minting.get(path)
    if (standing) return standing
    const id = crypto.randomUUID()
    minting.set(path, id)
    minted.set(id, path)
    return id
  }

  /** The identity of the tab standing at a file, and the name itself where none does. */
  const opened = (path: string): string => tabbed.value.get(path) ?? minting.get(path) ?? path

  /**
   * A note opened under the identity it was minted. It is owed its keyboard
   * from here until an editor has taken it, on the line it was told to stand on.
   */
  const opens = (id: string, line = ITSELF) => {
    const path = minted.get(id) ?? id
    notes.open(id, path)
    minting.delete(path)
    minted.delete(id)
    keyboard.owes(id, line)
    return held(id)
  }

  /**
   * A note put in front of the person, under the name it is called by and in a
   * tab of its own. It takes the keyboard, opened now or already open.
   */
  const shows = (path: string, title = '', showing: PlexShowing = 'here') => {
    const id = mints(path)
    if (title) names.calls(id, title)
    void (showing === 'beside' ? host.beside(NOTE, id) : host.opens(NOTE, id))
    keyboard.owes(id)
  }

  /** A note given the keyboard on a line, in whichever tab holds it. */
  const entersAt = (path: string, line?: number) => keyboard.owes(opened(path), line)

  /** The tab holding a note lets go of it, wherever the window draws it. */
  const shuts = (id: string) => {
    const tab = host.each<Held>(NOTE).find((one) => one.held.id === id)
    tab?.held.shuts(tab.id)
  }

  /**
   * What one tab of a note holds. What is being drawn over a note is filed by
   * the file it is being drawn on, which is where the note stands now.
   */
  const held = (id: string): Held => ({
    id,
    shown: () => notes.shown(id),
    saying: () => notes.saying(id),
    change: () => drawings.shown(notes.where(id)),
    typed: (body: string) => notes.typed(id, body),
    save: () => notes.save(id),
    keep: () => notes.keep(id),
    take: () => notes.take(id),
    drew: (editor: unknown) => keyboard.drew(id, editor),
    measure: () => keyboard.measure(id),
    /** The tab stands until the note says the write is done, and goes then. */
    shuts: (tab: string) => {
      keyboard.drops(id)
      drawings.shut(notes.where(id))
      void notes.shut(id).then((gone) => {
        if (!gone) return
        names.forgets(id)
        host.closes(tab)
      })
    },
  })

  /**
   * A note tab as the window keeps it. A note is its own tab, filed under the
   * identity it opened under.
   */
  const kind: Kind<Held> = {
    kind: NOTE,
    opens: (id) => opens(id),
    called: (held) => names.called(held.id),
    marked: (held) => markOf(notes.shown(held.id).state),
    draws: NoteTab,
    identity: (id) => id,
    makes: async () => {
      const path = await deps.makes()
      return path ? mints(path) : ''
    },
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
    calls: (path: string, title: string) => names.calls(mints(path), title),
    called: (path: string) => names.called(opened(path)),
    /** What the note that opened under an identity is called. */
    titled: (id: string) => names.called(id),
    entersAt,
    shuts,
    /**
     * The identity the note standing at a file opened under, and nothing where
     * none stands there. It is what the store answers to.
     */
    holding: (path: string): string | null => tabbed.value.get(path) ?? null,
  }
}
