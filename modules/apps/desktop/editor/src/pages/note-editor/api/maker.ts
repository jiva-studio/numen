/**
 * Making a note from the picture, and joining two that are already on it.
 *
 * A note is made in a relation of another one, and the link that puts it there is
 * written into it as it is made. What a relation means for what gets written is
 * decided here: the plex reports the shape of a gesture and nothing else.
 */
import type { PlexRelation } from '@numen/ui'
import { formatErrorMessage } from '@numen/wire'
import type { Link, NewNote, PairLink, Role } from '@/entities/note'
import type { CreateResult } from '@/entities/file'
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
  /**
   * A relationship removed from one note.
   */
  removeLink(path: string, to: string, role?: Role): Promise<ErrorCode | null>
  /** The links between two notes made exactly the ones given. */
  replaceLinks(first: string, second: string, links: readonly PairLink[]): Promise<ErrorCode | null>
}

/** The relations a link can put a note in. Each is also the role the link carries. */
type CreatableRelation = 'parent' | 'child' | 'jump'

/** Relations a person may make a note in: exactly those a link can write. */
export const CREATABLE: readonly CreatableRelation[] = ['parent', 'child', 'jump']

/** The role a link carries to put a note in that relation, and nothing for a sibling. */
const getRole = (relation: PlexRelation): CreatableRelation | undefined =>
  CREATABLE.find((one) => one === relation)

/**
 * What a new note writes about the note it was made from. The new one takes the
 * relation the gesture named, so from where it stands the other one takes the
 * opposite relation.
 */
const opposites: Partial<Record<PlexRelation, CreatableRelation>> = {
  parent: 'child',
  child: 'parent',
  jump: 'jump',
}

/**
 * What a new note writes to sit in that relation of another one. A note in no relation
 * writes nothing, and a relation nothing faces is not a relation to make one in.
 */
const resolveRelationLinks = (
  from: string,
  relation: PlexRelation | null,
): readonly Link[] | null => {
  if (!relation) return []
  const role = opposites[relation]
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

export function createNoteWriter(core: NoteMaker, writeMessage: MessageWriter) {
  /**
   * One note asked for. A name the vault has already filed is handed back as
   * `occupied` for the caller to answer for.
   */
  async function createNote(
    title: string,
    folder: string,
    links: readonly Link[],
  ): Promise<NoteRef | ErrorCode | null> {
    try {
      const made = await core.create({ title, folder, links })
      if (!made.ok) return made.error
      return { path: made.value.path, title }
    } catch (error) {
      writeMessage(formatErrorMessage(error), 'error')
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
      const made = await createNote(createNumberedName(taken), folder, links)
      if (made === 'occupied') continue
      return reportNoteResult(made)
    }
    writeMessage(exhausted, 'error')
    return null
  }

  /**
   * A note under the name a person gave it: in a relation of another note, filed
   * beside it, or on its own at the top of the vault.
   */
  async function createWithTitle(
    title: string,
    from: string,
    relation: PlexRelation | null,
  ): Promise<NoteRef | null> {
    const links = resolveRelationLinks(from, relation)
    if (!links) return null
    return reportNoteResult(await createNote(title, from ? getParentFolder(from) : '', links))
  }

  /** Create a note in a relation of another one, filed in the folder that one is in. */
  async function createInRelation(from: string, relation: PlexRelation): Promise<NoteRef | null> {
    const links = resolveRelationLinks(from, relation)
    if (!links) return null
    return createUntitled(getParentFolder(from), links)
  }

  /** What a note that was asked for came to, said to the person where it failed. */
  const reportNoteResult = (result: NoteRef | ErrorCode | null): NoteRef | null => {
    if (result === null) return null
    if (typeof result === 'string') {
      writeMessage(words[result], 'error')
      return null
    }
    writeMessage('')
    return result
  }

  /**
   * Join two notes that are both there. The link is written in the one the
   * gesture came from, and says where the other sits.
   */
  async function join(
    from: string,
    to: string,
    relation: PlexRelation,
    label?: string,
  ): Promise<boolean> {
    const role = getRole(relation)
    if (!role) return false
    try {
      const joinError = await core.join(from, {
        to,
        role,
        ...(label !== undefined ? { label } : {}),
      })
      if (joinError !== null) {
        writeMessage(words[joinError], 'error')
        return false
      }
      writeMessage('')
      return true
    } catch (error) {
      writeMessage(formatErrorMessage(error), 'error')
      return false
    }
  }

  /**
   * Remove a relationship from one note.
   */
  async function removeLink(from: string, to: string, relation?: PlexRelation): Promise<boolean> {
    const role = relation ? getRole(relation) : undefined
    try {
      const removeError = await core.removeLink(from, to, role)
      if (removeError !== null) {
        writeMessage(words[removeError], 'error')
        return false
      }
      writeMessage('')
      return true
    } catch (error) {
      writeMessage(formatErrorMessage(error), 'error')
      return false
    }
  }

  /**
   * Make the links between two notes exactly the ones given. A relation that is
   * not a role a link can carry refuses the whole set, and a refusal from the
   * core is said to the person.
   */
  async function replaceLinks(
    first: string,
    second: string,
    links: readonly { from: string; to: string; relation: PlexRelation; label: string }[],
  ): Promise<boolean> {
    const pairs: PairLink[] = []
    for (const { relation, ...link } of links) {
      const role = getRole(relation)
      if (!role) return false
      pairs.push({ ...link, role })
    }
    try {
      const replaceError = await core.replaceLinks(first, second, pairs)
      if (replaceError !== null) {
        writeMessage(words[replaceError], 'error')
        return false
      }
      writeMessage('')
      return true
    } catch (error) {
      writeMessage(formatErrorMessage(error), 'error')
      return false
    }
  }

  return {
    createInRelation,
    createUntitled,
    createWithTitle,
    join,
    removeLink,
    replaceLinks,
  }
}

export type NoteCreator = ReturnType<typeof createNoteWriter>

/** The name the note asked for after that many taken ones is filed under. */
const createNumberedName = (count: number): string =>
  count === 1 ? UNTITLED : `${UNTITLED} ${count}`

/** The folder a note is in, so one made from it is filed beside it. */
const getParentFolder = (path: string): string => {
  const cut = path.lastIndexOf('/')
  return cut < 0 ? '' : path.slice(0, cut)
}
