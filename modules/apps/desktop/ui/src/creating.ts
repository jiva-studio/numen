/**
 * Making a note from the picture, and joining two that are already on it.
 *
 * A note is made in a seat of another one, and the link that puts it there is
 * written into it as it is made. What a seat means for what gets written is
 * decided here: the plex reports the shape of a gesture and nothing else.
 */
import { ref } from 'vue'
import type { PlexRelatedSeat } from '@numen/ui'
import type { Core, NewLink, Refused } from './showing'

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

/** What a note is called before the person has called it anything. */
export const UNTITLED = 'Untitled note'

/** How many names are asked for before the vault refusing them all is said. */
const names = 100

/** What a person is told when a note could not be made or joined. */
const words: Record<Refused, string> = {
  missing: 'that note is not in the vault',
  notANote: 'that file is not a note',
  notText: 'that file is not text',
  tooLarge: 'that note is longer than this writes',
  bodyRefused: 'that text cannot be written into a note',
  unreadable: 'the frontmatter of that note cannot be read',
  occupied: `every name from ${UNTITLED} onwards is taken`,
}

/** A note that now exists: where it is filed, and what it is called. */
export interface Made {
  readonly path: string
  readonly title: string
}

export function creating(core: Core) {
  /** What could not be done, in words a person reads. */
  const said = ref('')

  /**
   * A note under the first name the vault has free.
   *
   * The name is asked for again for as long as the vault answers that it is
   * taken: whether a name is free is the filesystem's to answer at the moment
   * the file is made, so it is asked one name at a time.
   */
  async function named(folder: string, links: readonly NewLink[]): Promise<Made | null> {
    for (let taken = 1; taken <= names; taken++) {
      const title = nameAt(taken)
      try {
        const made = await core.create({ title, folder, links })
        if (made.refusal === 'occupied') continue
        if (made.refusal !== null) {
          said.value = words[made.refusal]
          return null
        }
        said.value = ''
        return { path: made.path, title }
      } catch (error) {
        said.value = String(error)
        return null
      }
    }
    said.value = words.occupied
    return null
  }

  /** Make a note in a seat of another one, filed in the folder that one is in. */
  async function make(from: string, seat: PlexRelatedSeat): Promise<Made | null> {
    const opposite = facing[seat]
    if (!opposite) return null
    return named(folderOf(from), [{ to: from, seat: opposite }])
  }

  /** A note standing on its own, filed at the top of the vault. */
  const start = (): Promise<Made | null> => named('', [])

  /**
   * Join two notes that are both there. The link is written in the one the
   * gesture came from, and says where the other sits.
   */
  async function join(from: string, to: string, seat: PlexRelatedSeat): Promise<boolean> {
    try {
      const refusal = await core.join(from, { to, seat })
      if (refusal !== null) {
        said.value = words[refusal]
        return false
      }
      said.value = ''
      return true
    } catch (error) {
      said.value = String(error)
      return false
    }
  }

  return { make, start, join, said }
}

/** The name the note asked for after that many taken ones is filed under. */
const nameAt = (taken: number): string => (taken === 1 ? UNTITLED : `${UNTITLED} ${taken}`)

/** The folder a note is in, so one made from it is filed beside it. */
const folderOf = (path: string): string => {
  const cut = path.lastIndexOf('/')
  return cut < 0 ? '' : path.slice(0, cut)
}
