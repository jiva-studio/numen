/**
 * Making a note from the picture, and joining two that are already on it.
 *
 * A note is made in a seat of another one, and the link that puts it there is
 * written into it as it is made. What a seat means for what gets written is
 * decided here: the plex reports the shape of a gesture and nothing else.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import { formatErrorMessage } from '@numen/wire'
import type { Link, NewNote, Role } from '@/entities/note'
import type { CreateResult } from '@/shared/file'
import type { ErrorCode } from '@/shared/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import { ERRORS } from '@/shared/words'

/** What making a note asks of the vault. */
export interface NoteMaker {
  /** A note made, named after the title it is given and joined as it is written. */
  create(note: NewNote): Promise<CreateResult>
  /**
   * A relationship written into one note. The note at the other end is left
   * alone: a link is one end's account of a relationship.
   */
  join(path: string, link: Link): Promise<ErrorCode | null>
}

/**
 * The role a link carries to seat a note where the gesture put it. A seat and
 * the role that seats a note in it are one relationship named twice, so they
 * carry the same word; a seat there is no role of that name for is written
 * nowhere, and a sibling is one.
 */
const seatRoles: Partial<Record<PlexRelatedSeat, Role>> = {
  parent: 'parent',
  child: 'child',
  jump: 'jump',
}

/** Seats a person may make a note in: exactly those a link can write. */
export const CREATABLE = Object.keys(seatRoles) as readonly PlexRelatedSeat[]

/**
 * What a new note writes about the note it was made from. The new one takes the
 * seat the gesture named, so from where it stands the other one takes the
 * opposite seat.
 */
const facing: Partial<Record<PlexRelatedSeat, PlexRelatedSeat>> = {
  parent: 'child',
  child: 'parent',
  jump: 'jump',
}

/**
 * What a new note writes to sit in that seat of another one. A note in no seat
 * writes nothing, and a seat nothing faces is not a seat to make one in.
 */
const resolveSeatLinks = (from: string, seat: PlexRelatedSeat | null): readonly Link[] | null => {
  if (!seat) return []
  const opposite = facing[seat]
  const role = opposite && seatRoles[opposite]
  return role ? [{ to: from, role }] : null
}

/** What a note is called before the person has called it anything. */
export const UNTITLED = 'Untitled note'

/** How many names are asked for before the vault refusing them all is said. */
const names = 100

/**
 * What a person is told when a note could not be made or joined. A name that
 * is taken is a name to choose again, and nothing has been renamed.
 */
const words: Record<ErrorCode, string> = {
  ...ERRORS,
  occupied: 'a note of that name is filed there already',
}

/** What a person is told when the vault took none of the names it was offered. */
const exhausted = `every name from ${UNTITLED} onwards is taken`

/** A note that now exists: where it is filed, and what it is called. */
export interface NoteRef {
  readonly path: string
  readonly title: string
}

export function noteCreator(core: NoteMaker, said: MessageWriter) {
  /**
   * One note asked for. A name the vault has already filed is handed back as
   * `occupied` for the caller to answer for.
   */
  async function creates(
    title: string,
    folder: string,
    links: readonly Link[],
  ): Promise<NoteRef | ErrorCode | null> {
    try {
      const made = await core.create({ title, folder, links })
      if (made.error) return made.error
      return { path: made.path, title }
    } catch (error) {
      said(formatErrorMessage(error), 'error')
      return null
    }
  }

  /**
   * A note under the first name the vault has free.
   *
   * The name is asked for again for as long as the vault answers that it is
   * taken: whether a name is free is the filesystem's to answer at the moment
   * the file is made, so it is asked one name at a time.
   */
  async function createUntitled(folder: string, links: readonly Link[]): Promise<NoteRef | null> {
    for (let taken = 1; taken <= names; taken++) {
      const made = await creates(nameAt(taken), folder, links)
      if (made === 'occupied') continue
      return reportNoteResult(made)
    }
    said(exhausted, 'error')
    return null
  }

  /**
   * A note under the name a person gave it: in a seat of another note, filed
   * beside it, or on its own at the top of the vault.
   */
  async function createWithTitle(
    title: string,
    from: string,
    seat: PlexRelatedSeat | null,
  ): Promise<NoteRef | null> {
    const links = resolveSeatLinks(from, seat)
    if (!links) return null
    return reportNoteResult(await creates(title, from ? folderOf(from) : '', links))
  }

  /** Create a note in a seat of another one, filed in the folder that one is in. */
  async function createInSeat(from: string, seat: PlexRelatedSeat): Promise<NoteRef | null> {
    const links = resolveSeatLinks(from, seat)
    if (!links) return null
    return createUntitled(folderOf(from), links)
  }

  /** What a note that was asked for came to, said to the person where it failed. */
  const reportNoteResult = (made: NoteRef | ErrorCode | null): NoteRef | null => {
    if (made === null) return null
    if (typeof made === 'string') {
      said(words[made], 'error')
      return null
    }
    said('')
    return made
  }

  /**
   * Join two notes that are both there. The link is written in the one the
   * gesture came from, and says where the other sits.
   */
  async function join(from: string, to: string, seat: PlexRelatedSeat): Promise<boolean> {
    const role = seatRoles[seat]
    if (!role) return false
    try {
      const joinError = await core.join(from, { to, role })
      if (joinError !== null) {
        said(words[joinError], 'error')
        return false
      }
      said('')
      return true
    } catch (error) {
      said(formatErrorMessage(error), 'error')
      return false
    }
  }

  return {
    createNote: createWithTitle,
    createInSeat,
    createUntitled,
    createWithTitle,
    make: createInSeat,
    named: createUntitled,
    calls: createWithTitle,
    join,
  }
}

export type NoteCreator = ReturnType<typeof noteCreator>

/** The name the note asked for after that many taken ones is filed under. */
const nameAt = (taken: number): string => (taken === 1 ? UNTITLED : `${UNTITLED} ${taken}`)

/** The folder a note is in, so one made from it is filed beside it. */
const folderOf = (path: string): string => {
  const cut = path.lastIndexOf('/')
  return cut < 0 ? '' : path.slice(0, cut)
}
