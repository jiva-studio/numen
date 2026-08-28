/**
 * The stencils the window has open, and what one stencil tab holds.
 *
 * A stencil is saved the way a note is, and the string the store is dirty
 * against is its fields and its faces written out. A field renamed here is
 * renamed in every card the vault knows it cuts, which is the write's doing.
 */
import type { Half, PlexShowing } from '@numen/ui'
import type { Cards, Refused, Went } from '../core'
import type { Store } from '../doing'
import { editing, type Editing } from '../note/editing'
import { markOf } from '../note/tab'
import type { Says } from '../telling'
import type { Host, Kind } from '../windowing'
import { REFUSED } from '../words'
import type { Putting } from '../putting'
import { STENCIL } from '../workspace'
import StencilTab from './StencilTab.vue'
import {
  faceAdded,
  faceCarried,
  faceGone,
  faceNamed,
  faceWritten,
  facesOf,
  fieldAdded,
  fieldCarried,
  fieldGone,
  marksOf,
  sameMarks,
  sameSheet,
  sheetBodyOf,
  sheetIn,
  sheetOf,
  type Marks,
  type Sheet,
} from './model'
import { WORDS as words } from './words'

/** What the vault said about one file the last time it was read or written. */
interface Told {
  /**
   * What is wrong, against the face it was read against. A problem stands on
   * the face it came in on, so a face carried elsewhere takes its mark with it.
   */
  readonly marks: Marks
  readonly refusal: Refused | null
  /** The file the stencil last came out of, for a rename to present. */
  readonly at: string
}

const NOTHING: Told = {
  marks: { at: new Map(), fields: new Map(), whole: [] },
  refusal: null,
  at: '',
}

/** What one stencil tab holds. */
export interface Held {
  /** The file this tab opened on, which is the identity it keeps. */
  readonly id: string
  /** The stencil as the window draws it: the state it is in, and what it stands at. */
  shown(): Editing
  /** The fields and the faces, as the editor draws them. */
  sheet(): Sheet
  /** What is wrong with the file, against the face or the field it stands on. */
  marks(): Marks
  /** What the whole file was refused for, in words a person reads. */
  saying(): string
  addsField(name: string): void
  namesField(field: string, name: string): void
  removesField(field: string): void
  movesField(field: string, at: string | null): void
  addsFace(name: string): void
  namesFace(id: string, name: string): void
  removesFace(id: string): void
  movesFace(id: string, at: string | null): void
  writes(id: string, half: Half, text: string): void
  /** The person keeps what they have written, over whatever the file holds. */
  keep(): void
  /** The person takes what the file holds. */
  take(): void
  /** The tab is closing, and what is unwritten goes to the file first. */
  shuts(id: string): void
}

export function stencilling(
  cards: Cards,
  host: Host,
  puts: Putting,
  says: Says = () => {},
) {
  /** What the vault last said about each file, under the path it is filed at. */
  const told = new Map<string, Told>()
  /** What each file is called, as the vault last read it. */
  const titles = new Map<string, string>()

  /**
   * What is wrong with a file, as this reading has it. A reading saying what
   * the last one said leaves what is drawn against the file standing.
   */
  const marking = (path: string, read: Marks): Marks => {
    const held = told.get(path)?.marks
    return held && sameMarks(held, read) ? held : read
  }

  const store = editing({
    read: async (path) => {
      const answer = await cards.readStencil(path)
      const sheet = answer.stencil ? sheetOf(answer.stencil) : null
      told.set(path, {
        marks: marking(
          path,
          marksOf(answer.stencil?.problems ?? [], [], sheet?.faces.map((face) => face.id) ?? []),
        ),
        refusal: answer.refusal,
        at: answer.at,
      })
      if (answer.stencil) titles.set(path, answer.stencil.title)
      if (answer.refusal !== null) return { body: '', refusal: answer.refusal }
      return { body: sheet ? sheetBodyOf(sheet) : '', refusal: null, at: answer.at }
    },
    write: async (path, body, seen) => {
      const sheet = sheetIn(body)
      const answer = await cards.writeStencil(
        path,
        sheet.fields,
        facesOf(sheet),
        seen?.at ?? null,
      )
      const said = told.get(path) ?? NOTHING
      told.set(path, {
        marks: said.marks,
        refusal: answer.refusal,
        // Nothing was written where the write was refused or overtaken, so the
        // file the tab last stood on is the file it still stands on.
        at: answer.changed || answer.refusal !== null ? said.at : answer.at,
      })
      return { body: '', refusal: answer.refusal, at: answer.at, changed: answer.changed }
    },
  })

  /** The last string a stencil was read out of, and what it came to. */
  const parsed = new Map<string, { body: string; sheet: Sheet }>()

  const sheetAt = (id: string): Sheet => {
    const body = store.shown(id).body
    const held = parsed.get(id)
    if (held && held.body === body) return held.sheet
    // A file read again carries fresh identities for the same faces, so the
    // string it comes back as differs from the string that went out. A stencil
    // reading as the one on screen leaves that one standing, and the field a
    // person is typing into is not drawn again.
    const read = sheetIn(body)
    const sheet = held && sameSheet(held.sheet, read) ? held.sheet : read
    parsed.set(id, { body, sheet })
    return sheet
  }

  /** A stencil as it now stands, written back into the store. */
  const turns = (id: string, sheet: Sheet): void => {
    const body = sheetBodyOf(sheet)
    parsed.set(id, { body, sheet })
    store.typed(id, body)
  }

  /**
   * A field under another name, which the vault writes wherever that name
   * stands: in the fields of this stencil, in the placeholders of its faces,
   * and as a heading in every card this stencil cuts. The tab then reads the
   * file the way it reads any file the vault has changed under it.
   */
  const renames = async (id: string, field: string, name: string): Promise<void> => {
    if (!name || name === field) return
    const path = store.where(id)
    const answer = await cards.renameField(path, field, name, (told.get(path) ?? NOTHING).at || null)
    if (answer.refusal !== null) return says(REFUSED[answer.refusal], 'refusal')
    if (answer.cards > 0) says(words.renamed(answer.cards, answer.decks.length))
    // A deck the rename did not reach keeps the old heading, and nothing else
    // would tell the person which.
    if (answer.notWritten.length > 0) {
      says(words.notWritten(answer.notWritten.map((one) => one.path)), 'refusal')
    }
    store.changed([path])
  }

  /** What one file was refused for, in words a person reads. */
  const sayingOf = (path: string): string => {
    const said = told.get(path) ?? NOTHING
    if (said.refusal === 'notAStencil') return words.notAStencil
    return said.refusal === null ? '' : words.refused
  }

  const held = (id: string): Held => ({
    id,
    shown: () => store.shown(id),
    sheet: () => sheetAt(id),
    marks: () => (told.get(store.where(id)) ?? NOTHING).marks,
    saying: () => sayingOf(store.where(id)),
    addsField: (name) => turns(id, fieldAdded(sheetAt(id), name)),
    namesField: (field, name) => void renames(id, field, name),
    removesField: (field) => turns(id, fieldGone(sheetAt(id), field)),
    movesField: (field, at) => turns(id, fieldCarried(sheetAt(id), field, at)),
    addsFace: (name) => turns(id, faceAdded(sheetAt(id), name)),
    namesFace: (face, name) => turns(id, faceNamed(sheetAt(id), face, name)),
    removesFace: (face) => turns(id, faceGone(sheetAt(id), face)),
    movesFace: (face, at) => turns(id, faceCarried(sheetAt(id), face, at)),
    writes: (face, half, text) => turns(id, faceWritten(sheetAt(id), face, half, text)),
    keep: () => store.keep(id),
    take: () => store.take(id),
    shuts: (tab) => {
      void store.shut(id).then((gone) => {
        if (!gone) return
        parsed.delete(id)
        host.closes(tab)
      })
    },
  })

  /** What a stencil tab is called: the title the file carries, or the file itself. */
  const called = (path: string): string => titles.get(path) || (path.split('/').pop() ?? path)

  /** The tab holding a stencil lets go of it, wherever the window draws it. */
  const shuts = (id: string): void => {
    const tab = host.each<Held>(STENCIL).find((one) => one.held.id === id)
    tab?.held.shuts(tab.id)
  }

  /** The stencils, as a command reaches the ones the window has open. */
  const kept: Store = {
    has: (id) => store.all().includes(id),
    where: (id) => store.where(id),
    called: (id) => called(store.where(id)),
    asking: (id) => store.overtaken(id) !== null,
    settles: (id) => store.settles(id),
    shuts,
    holding: (path) => store.all().find((id) => store.where(id) === path) ?? null,
  }

  /** A stencil tab as the window keeps it, filed under the path it opened at. */
  const kind: Kind<Held> = {
    kind: STENCIL,
    opens: (path) => {
      store.open(path)
      return held(path)
    },
    called: (one) => called(store.where(one.id)),
    marked: (one) => markOf(store.shown(one.id).state),
    draws: StencilTab,
    identity: (path) => path,
    shuts: (one, id) => {
      one.shuts(id)
      return false
    },
    gone: () => {},
  }

  /** A stencil put in front of the person, in a tab of its own. */
  const shows = (path: string, title = '', showing: PlexShowing = 'here'): void => {
    if (title) titles.set(path, title)
    void (showing === 'beside' ? host.beside(STENCIL, path) : host.opens(STENCIL, path))
  }

  // The editor of a stencil, which is its fields and its faces. A face stands
  // on no line of prose, so a stencil asked for at a place inside it opens
  // whole.
  puts.holds('stencil', shows)

  const changed = (paths: readonly string[], renamed: readonly Went[] = []): void => {
    store.changed(paths, renamed)
  }

  return {
    kind,
    held,
    changed,
    called,
    kept,
    all: store.all,
    shown: store.shown,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
