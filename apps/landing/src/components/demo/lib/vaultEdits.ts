import type { Relation } from './neighborhood'
import { INITIAL_NOTES, type VaultNote } from './notes'

const NEW_NOTE_BODY =
  'Write Markdown here — headings, lists, math, and [[wikilinks]] format live as you move the caret.\n'

/** A note with no links and the starting text. */
export const getBlankNote = (title: string, folder: string): Omit<VaultNote, 'path'> => ({
  title,
  folder,
  parents: [],
  jumps: [],
  body: NEW_NOTE_BODY,
})

/** A note made as a parent, a child or a jump of another, in the folder of the other. */
export const getRelatedNote = (
  from: VaultNote,
  relation: Relation,
  title: string,
  folder: string,
): Omit<VaultNote, 'path'> => ({
  title,
  folder,
  parents: relation === 'child' ? [from.path] : [],
  jumps: relation === 'jump' ? [from.path] : [],
  body: `Created as a **${relation}** of [[${from.title}]].\n\n${NEW_NOTE_BODY}`,
})

/** The path a note is filed under in a folder. */
export const getNotePath = (folder: string, title: string): string =>
  folder ? `${folder}/${title}.md` : `${title}.md`

/** The notes of the demo vault as they start, each a copy that can be edited. */
export const getInitialNotes = (): Record<string, VaultNote> =>
  Object.fromEntries(
    INITIAL_NOTES.map((one) => [one.path, { ...one, parents: [...one.parents], jumps: [...one.jumps] }]),
  )

/** The first title "Untitled note", "Untitled note 2", ... that no note in the folder has. */
export function getUntitledTitle(isTaken: (path: string) => boolean, folder: string): string {
  for (let count = 1; count < 1000; count += 1) {
    const title = count === 1 ? 'Untitled note' : `Untitled note ${count}`
    if (!isTaken(getNotePath(folder, title))) return title
  }
  return 'Untitled note'
}

/** The first name "New folder", "New folder 2", ... that is not among the taken names. */
export function getNewFolderName(taken: ReadonlySet<string>): string {
  let name = 'New folder'
  for (let count = 2; taken.has(name); count++) name = `New folder ${count}`
  return name
}

/** The folders after the folder at `from` and everything in it move to `to`. */
export function getMovedFolders(folders: readonly string[], from: string, to: string): string[] {
  const prefix = `${from}/`
  return folders.map((one) => {
    if (one === from) return to
    return one.startsWith(prefix) ? `${to}/${one.slice(prefix.length)}` : one
  })
}

/** The edge labels with every key that names the note at `from` renamed to name the note at `to`. */
export function getRekeyedLabels(
  labels: Readonly<Record<string, string>>,
  from: string,
  to: string,
): Record<string, string> {
  const rekeyed: Record<string, string> = {}
  for (const [key, label] of Object.entries(labels)) {
    const [source, target] = key.split('->')
    const isNamed = source === from || target === from
    rekeyed[isNamed ? `${source === from ? to : source}->${target === from ? to : target}` : key] = label
  }
  return rekeyed
}
