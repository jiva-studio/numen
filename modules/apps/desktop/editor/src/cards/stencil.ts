/**
 * The stencils the window has open, and what one stencil tab holds.
 *
 * A stencil is saved the way a note is, and the string the store is dirty
 * against is its fields and its faces written out. A field renamed here is
 * renamed in every card the vault knows it cuts, which is the write's doing.
 */
import { computed, type ComputedRef } from 'vue'
import type { Half, PlexShowing } from '@numen/ui'
import type { Move, RefusalReason } from '../core'
import type { Cards, Problem } from './vault'
import type { Store } from '../command/handlers'
import { openNotes, type OpenNote } from '../note/notes'
import { markOf } from '../note/tab'
import type { MessageWriter } from '../notices/messages'
import type { Kind, WindowHandle } from '../tabs/windowing'
import { REFUSED } from '../words'
import type { FileOpeners } from '../tabs/openers'
import { STENCIL } from '../tabs/workspace'
import StencilTab from './StencilTab.vue'
import {
  faceAdded,
  faceDropped,
  faceGone,
  faceNamed,
  faceWritten,
  facesOf,
  fieldAdded,
  fieldDropped,
  fieldGone,
  sameSheet,
  sheetBodyOf,
  sheetIn,
  sheetOf,
  type Sheet,
} from './body'
import { marksOf, sameMarks, type Marks } from './marks'
import { WORDS as words } from './words'

/** What the vault said about one file the last time it was read or written. */
interface VaultAnswer {
  /**
   * What is wrong with the file, in the order the faces were read in. Which
   * face each stands on is decided against the stencil the editor is drawing,
   * so a face the window is holding through a re-read keeps its mark.
   */
  readonly problems: readonly Problem[]
  /** What the last read of the file was refused for. */
  readonly reading: RefusalReason | null
  /** What the last write of it was refused for. */
  readonly writing: RefusalReason | null
  /** The file the stencil last came out of, for a rename to present. */
  readonly at: string
}

const NOTHING: VaultAnswer = { problems: [], reading: null, writing: null, at: '' }

/** What one stencil tab holds. */
export interface StencilTabState {
  /** The identity this stencil opened under, which its tab keeps wherever it goes. */
  readonly id: string
  /** The stencil as the window draws it: the state it is in, and what it stands at. */
  readonly shown: ComputedRef<OpenNote>
  /** The fields and the faces, as the editor draws them. */
  readonly sheet: ComputedRef<Sheet>
  /** What is wrong with the file, against the face or the field it stands on. */
  readonly marks: ComputedRef<Marks>
  /** What the whole file was refused for, in words a person reads. */
  readonly saying: ComputedRef<string>
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
  handle: WindowHandle,
  puts: FileOpeners,
  says: MessageWriter = () => {},
) {
  /** What the vault last said about each file, under the path it is filed at. */
  const told = new Map<string, VaultAnswer>()
  /** What each file is called, as the vault last read it. */
  const titles = new Map<string, string>()

  const store = openNotes({
    read: async (path) => {
      const answer = await cards.readStencil(path)
      const sheet = answer.stencil ? sheetOf(answer.stencil) : null
      told.set(path, {
        problems: answer.stencil?.problems ?? [],
        reading: answer.refusal,
        writing: null,
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
        { preamble: sheet.preamble, faces: facesOf(sheet), tail: sheet.tail },
        seen?.at ?? null,
      )
      const said = told.get(path) ?? NOTHING
      told.set(path, {
        problems: said.problems,
        reading: said.reading,
        writing: answer.refusal,
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

  /** The problems one tab was last marked from, and the marks that came of it. */
  const marked = new Map<string, { problems: readonly Problem[]; marks: Marks }>()

  /**
   * What is wrong with a file, against the face the editor is drawing. A
   * problem carries where it stood in the file it was read from, so it is put
   * against a face once, when the reading it came in on is the newest one: a
   * face dragged elsewhere takes its mark with it from there.
   */
  const marksAt = (id: string): Marks => {
    const problems = (told.get(store.where(id)) ?? NOTHING).problems
    const held = marked.get(id)
    if (held && held.problems === problems) return held.marks
    const read = marksOf(
      problems,
      [],
      sheetAt(id).faces.map((face) => face.id),
    )
    // Marks saying what the last ones said leave what is drawn against the
    // file standing.
    const marks = held && sameMarks(held.marks, read) ? held.marks : read
    marked.set(id, { problems, marks })
    return marks
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
    // The file moved past the stencil this tab read, and nothing was renamed
    // anywhere. What it now holds is what the tab reads next.
    if (answer.changed) {
      says(words.notRenamed, 'refusal')
      return store.changed([path])
    }
    if (answer.cards > 0) says(words.renamed(answer.cards, answer.decks.length))
    // A deck the rename did not reach keeps the old heading, and nothing else
    // would tell the person which.
    if (answer.notWritten.length > 0) {
      says(words.notWritten(answer.notWritten.map((one) => one.path)), 'refusal')
    }
    store.changed([path])
  }

  /**
   * What one tab was refused for, in words a person reads. A vault that
   * answered nothing at all left the tab refused and said no word of its own.
   */
  const sayingOf = (id: string): string => {
    if (store.shown(id).refusal === null) return ''
    const said = told.get(store.where(id)) ?? NOTHING
    if (said.reading !== null) {
      return said.reading === 'notAStencil' ? words.notAStencil : words.refused
    }
    if (said.writing !== null) {
      return said.writing === 'notAStencil' ? words.notAStencil : words.notSaved
    }
    return words.unreachable
  }

  const held = (id: string): StencilTabState => ({
    id,
    shown: computed(() => store.shown(id)),
    sheet: computed(() => sheetAt(id)),
    marks: computed(() => marksAt(id)),
    saying: computed(() => sayingOf(id)),
    addsField: (name) => turns(id, fieldAdded(sheetAt(id), name)),
    namesField: (field, name) => void renames(id, field, name),
    removesField: (field) => turns(id, fieldGone(sheetAt(id), field)),
    movesField: (field, at) => turns(id, fieldDropped(sheetAt(id), field, at)),
    addsFace: (name) => turns(id, faceAdded(sheetAt(id), name)),
    namesFace: (face, name) => turns(id, faceNamed(sheetAt(id), face, name)),
    removesFace: (face) => turns(id, faceGone(sheetAt(id), face)),
    movesFace: (face, at) => turns(id, faceDropped(sheetAt(id), face, at)),
    writes: (face, half, text) => turns(id, faceWritten(sheetAt(id), face, half, text)),
    keep: () => store.keep(id),
    take: () => store.take(id),
    shuts: (tab) => {
      const path = store.where(id)
      void store.shut(id).then((gone) => {
        if (!gone) return
        parsed.delete(id)
        marked.delete(id)
        forgets(path)
        handle.closes(tab)
      })
    },
  })

  /**
   * What the vault said about a file no tab of this window stands at any
   * longer. A second tab standing there keeps it.
   */
  const forgets = (path: string): void => {
    if (store.all().some((one) => store.where(one) === path)) return
    told.delete(path)
    titles.delete(path)
  }

  /** What a stencil tab is called: the title the file carries, or the file itself. */
  const called = (path: string): string => titles.get(path) || (path.split('/').pop() ?? path)

  /** The tab holding a stencil lets go of it, wherever the window draws it. */
  const shuts = (id: string): void => {
    const tab = handle.each<StencilTabState>(STENCIL).find((one) => one.state.id === id)
    tab?.state.shuts(tab.id)
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

  /**
   * Every open stencil under the file it stands at now, against the identity it
   * opened under. A stencil that moved is looked up here to reach the tab
   * already holding it.
   */
  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(store.all().map((one) => [store.where(one), one])),
  )

  /**
   * The identity minted for a stencil asked for by name, until its tab opens
   * under it. A stencil is asked for and shown in two steps, and both name the
   * same tab.
   */
  const minting = new Map<string, string>()
  const minted = new Map<string, string>()

  /** The identity of the tab standing at a file, minted where none stands there. */
  const mints = (path: string): string => {
    const open = tabbed.value.get(path) ?? minting.get(path)
    if (open) return open
    const one = crypto.randomUUID()
    minting.set(path, one)
    minted.set(one, path)
    return one
  }

  /**
   * A stencil tab as the window keeps it, filed under the identity it opened
   * under, so the same file asked for twice is the tab it has wherever the file
   * has been renamed to since.
   */
  const kind: Kind<StencilTabState> = {
    kind: STENCIL,
    opens: (id) => {
      const path = minted.get(id) ?? id
      store.open(id, path)
      minting.delete(path)
      minted.delete(id)
      return held(id)
    },
    called: (one) => called(store.where(one.id)),
    marked: (one) => markOf(one.shown.value.state),
    draws: StencilTab,
    identity: (id) => id,
    shuts: (one, id) => {
      one.shuts(id)
      return false
    },
    gone: () => {},
  }

  /** A stencil put in front of the person, in a tab of its own. */
  const shows = (path: string, title = '', showing: PlexShowing = 'here'): void => {
    const id = mints(path)
    if (title) titles.set(path, title)
    void (showing === 'beside' ? handle.beside(STENCIL, id) : handle.opens(STENCIL, id))
  }

  // The editor of a stencil, which is its fields and its faces. A face stands
  // on no line of prose, so a stencil asked for at a place inside it opens
  // whole.
  puts.holds('stencil', shows)

  const changed = (paths: readonly string[], renamed: readonly Move[] = []): void => {
    // What the vault said about a file is filed under that file, so a file
    // that moved takes it along.
    for (const went of renamed) {
      const said = told.get(went.from)
      if (said) told.set(went.to, said)
      told.delete(went.from)
      const title = titles.get(went.from)
      if (title !== undefined) titles.set(went.to, title)
      titles.delete(went.from)
    }
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
