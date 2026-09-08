/**
 * What the window knows about the notes it has open.
 *
 * The notes are read and written by one store for the whole window: what is
 * unsaved is answered to the quit as one question, and a change to the vault
 * reaches all of them at once. What one tab of one note holds is made here from
 * that store.
 */
import { computed, type ComputedRef } from 'vue'
import { pointsAtNote, type PlexShowing } from '@numen/ui'
import type { Store } from '../command/deps'
import type { Kind, WindowHandle } from '../tabs/windowing'
import { NOTE } from '../tabs/workspace'
import type { Change } from './drawing'
import type { noteChanges } from './changes'
import type { OpenNote, openNotes } from './notes'
import { noteKeyboard, ITSELF } from './keyboard'
import { noteTitles, type NoteTitlesDeps } from './titles'
import NoteTab from './NoteTab.vue'
import { markOf } from './tab'
import type { FileOpeners } from '../tabs/openers'

/** The notes of the whole window, read and written by one store. */
type Notes = ReturnType<typeof openNotes>
/** What is being typed into each note now, as the editor draws it. */
type NoteChanges = ReturnType<typeof noteChanges>

export type { NoteTitlesDeps }

/** What the notes of a window ask of the vault, beside what names them. */
export interface NoteTabDeps extends NoteTitlesDeps {
  /**
   * Where each of those addresses lands, by the address it was asked about.
   * They are written in the note at `from`, and one that reaches nothing is
   * absent.
   */
  resolve(from: string, written: readonly string[]): Promise<ReadonlyMap<string, string>>
}

/** What one note tab holds: its text, and the answers a person gives it. */
export interface NoteTabState {
  /** The identity this note opened under, which its tab keeps wherever it goes. */
  readonly id: string
  /** The note as the window draws it: the body, and the state it is in. */
  readonly shown: ComputedRef<OpenNote>
  /** What could not be read or written, in words a person reads. */
  readonly saying: ComputedRef<string>
  /** What arrived from elsewhere, for the editor to take into what is typed. */
  readonly change: ComputedRef<Change | null>
  typed(body: string): void
  save(): void
  /** The person keeps what they have written, over whatever the file holds. */
  keep(): void
  /** The person takes what the file holds. */
  take(): void
  /** The editor of this note, as it is drawn and as it goes. */
  drew(editor: unknown): void
  measure(): void
  /**
   * A link in the prose followed. One naming a note opens it beside this one;
   * an address no note answers to opens nothing.
   */
  follows(address: string): void
  /** The tab is closing, and what is unwritten goes to the file first. */
  shuts(id: string): void
}

export function noting(
  vault: NoteTabDeps,
  notes: Notes,
  changes: NoteChanges,
  handle: WindowHandle,
  puts: FileOpeners,
) {
  const names = noteTitles(vault, notes)
  const keyboard = noteKeyboard()

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
    const open = tabbed.value.get(path) ?? minting.get(path)
    if (open) return open
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
    void (showing === 'beside' ? handle.beside(NOTE, id) : handle.opens(NOTE, id))
    keyboard.owes(id)
  }

  /** A note given the keyboard on a line, in whichever tab holds it. */
  const entersAt = (path: string, line?: number) => keyboard.owes(opened(path), line)

  // The editor of a note, which is where its prose is read and written. A line
  // is one of the lines of that prose, and the keyboard stands on it. A link
  // note is a note: what it points at is drawn above the prose, in the tab the
  // prose is in.
  for (const kind of ['note'] as const) {
    puts.holds(kind, (path, title, showing, line) => {
      shows(path, title, showing)
      if (line !== undefined) entersAt(path, line)
    })
  }

  /** The tab holding a note lets go of it, wherever the window draws it. */
  const shuts = (id: string) => {
    const tab = handle.each<NoteTabState>(NOTE).find((one) => one.state.id === id)
    tab?.state.shuts(tab.id)
  }

  /**
   * What one tab of a note holds. What is being drawn over a note is filed by
   * the file it is being drawn on, which is where the note stands now.
   */
  const held = (id: string): NoteTabState => {
    return {
      id,
      shown: computed(() => notes.shown(id)),
      saying: computed(() => notes.saying(id)),
      change: computed(() => changes.shown(notes.where(id))),
      typed: (body: string) => notes.typed(id, body),
      save: () => notes.save(id),
      keep: () => notes.keep(id),
      take: () => notes.take(id),
      drew: (editor: unknown) => keyboard.drew(id, editor),
      measure: () => keyboard.measure(id),
      follows: (address: string) => {
        if (!pointsAtNote(address)) return
        const from = notes.where(id)
        void vault.resolve(from, [address]).then((landed) => {
          const path = landed.get(address)
          if (path) void puts.opens(path, '', 'beside')
        })
      },
      /** The tab stands until the note says the write is done, and goes then. */
      shuts: (tab: string) => {
        keyboard.drops(id)
        changes.shut(notes.where(id))
        void notes.shut(id).then((gone) => {
          if (!gone) return
          names.forgets(id)
          handle.closes(tab)
        })
      },
    }
  }

  /**
   * A note tab as the window keeps it. A note is its own tab, filed under the
   * identity it opened under.
   */
  /** The file this note stands at now, and nothing while the store has let it go. */
  const standsAt = (state: NoteTabState): string =>
    notes.has(state.id) ? notes.where(state.id) : ''

  const kind: Kind<NoteTabState> = {
    kind: NOTE,
    opens: (id) => opens(id),
    called: (state) => names.called(state.id),
    marked: (state) => markOf(state.shown.value.state),
    draws: NoteTab,
    identity: (id) => id,
    shown: (state) => state.measure(),
    at: (state) => {
      const path = standsAt(state)
      return { path, title: path ? names.called(state.id) : '' }
    },
    attends: (state) => ({ path: standsAt(state) }),
    shuts: (state, id) => {
      state.shuts(id)
      return false
    },
    // What an open note owes at the quit is written by the quit, which the
    // window waits for.
    gone: () => {},
  }

  /** The notes, as a command reaches the ones the window has open. */
  const kept: Store = {
    has: (id) => notes.has(id),
    where: (id) => notes.where(id),
    called: (id) => names.called(id),
    asking: (id) => notes.overtaken(id) !== null,
    settles: (id) => notes.settles(id),
    shuts,
    holding: (path) => tabbed.value.get(path) ?? null,
  }

  return {
    kind,
    kept,
    held,
    opens,
    titles: names.titles,
    /** Every open note's editor takes its measurements again. */
    measures: keyboard.measures,
    calls: (path: string, title: string) => names.calls(mints(path), title),
    called: (path: string) => names.called(opened(path)),
    entersAt,
    shuts,
    /**
     * What the application is doing, as it last said. A tab whose note is named
     * there is being fetched for, and asks for its transcript again — which is
     * how a copy that has just landed becomes what plays.
     */
  }
}
