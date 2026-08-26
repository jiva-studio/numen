/**
 * Making a note from the picture, and joining two that are already on it.
 *
 * A note is made in a seat of another one, and the link that puts it there is
 * written into it as it is made. What a seat means for what gets written is
 * decided here: the plex reports the shape of a gesture and nothing else.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import type { Core, NewLink, Refused, Role } from '../core'
import type { Says } from '../telling'
import { REFUSED } from '../words'

/**
 * Seats a person may make a note in. A sibling is another child of a shared
 * parent, so no link writes one.
 */
export const CREATABLE = ['parent', 'child', 'jump'] as const

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
 * The role a link carries to seat a note where the gesture put it. A sibling is
 * another child of a shared parent, so no link writes one.
 */
const carries: Partial<Record<PlexRelatedSeat, Role>> = {
  parent: 'parent',
  child: 'child',
  jump: 'jump',
}

/**
 * What a new note writes to sit in that seat of another one. A note in no seat
 * writes nothing, and a seat nothing faces is not a seat to make one in.
 */
const seatedOn = (from: string, seat: PlexRelatedSeat | null): readonly NewLink[] | null => {
  if (!seat) return []
  const opposite = facing[seat]
  const role = opposite && carries[opposite]
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
const words: Record<Refused, string> = {
  ...REFUSED,
  occupied: 'a note of that name is filed there already',
}

/** What a person is told when the vault took none of the names it was offered. */
const exhausted = `every name from ${UNTITLED} onwards is taken`

/** A note that now exists: where it is filed, and what it is called. */
export interface Made {
  readonly path: string
  readonly title: string
}

export function creating(core: Core, said: Says) {
  /**
   * One note asked for. A name the vault has already filed is handed back as
   * `occupied` for the caller to answer for.
   */
  async function creates(
    title: string,
    folder: string,
    links: readonly NewLink[],
  ): Promise<Made | Refused | null> {
    try {
      const made = await core.create({ title, folder, links })
      if (made.refusal !== null) return made.refusal
      return { path: made.path, title }
    } catch (error) {
      said(String(error), 'refusal')
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
  async function named(folder: string, links: readonly NewLink[]): Promise<Made | null> {
    for (let taken = 1; taken <= names; taken++) {
      const made = await creates(nameAt(taken), folder, links)
      if (made === 'occupied') continue
      return answered(made)
    }
    said(exhausted, 'refusal')
    return null
  }

  /**
   * A note under the name a person gave it: in a seat of another note, filed
   * beside it, or on its own at the top of the vault.
   */
  async function calls(
    title: string,
    from: string,
    seat: PlexRelatedSeat | null,
  ): Promise<Made | null> {
    const links = seatedOn(from, seat)
    if (!links) return null
    return answered(await creates(title, from ? folderOf(from) : '', links))
  }

  /** Make a note in a seat of another one, filed in the folder that one is in. */
  async function make(from: string, seat: PlexRelatedSeat): Promise<Made | null> {
    const links = seatedOn(from, seat)
    if (!links) return null
    return named(folderOf(from), links)
  }

  /** A note standing on its own, filed at the top of the vault. */
  const start = (): Promise<Made | null> => named('', [])

  /** What a note that was asked for came to, said to the person where it failed. */
  const answered = (made: Made | Refused | null): Made | null => {
    if (made === null) return null
    if (typeof made === 'string') {
      said(words[made], 'refusal')
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
    const role = carries[seat]
    if (!role) return false
    try {
      const refusal = await core.join(from, { to, role })
      if (refusal !== null) {
        said(words[refusal], 'refusal')
        return false
      }
      said('')
      return true
    } catch (error) {
      said(String(error), 'refusal')
      return false
    }
  }

  return { make, named, calls, start, join }
}

/** The name the note asked for after that many taken ones is filed under. */
const nameAt = (taken: number): string => (taken === 1 ? UNTITLED : `${UNTITLED} ${taken}`)

/** The folder a note is in, so one made from it is filed beside it. */
const folderOf = (path: string): string => {
  const cut = path.lastIndexOf('/')
  return cut < 0 ? '' : path.slice(0, cut)
}
