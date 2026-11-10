import { PDF_DOCS } from './documents'
import type { VaultNote } from './notes'
import { DECK_PATH, RECORDING_PATH, YOUTUBE_LINK_PATH } from './tabs'

export const ROOT = ''

/** One row of the file tree: a folder or a file. */
export interface Entry {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
  readonly isNote: boolean
  readonly type: 'note' | 'link' | 'deck'
}

/** Whether a row is a note file, whose selection the linked tabs follow. */
export const isNoteRow = (entry: Entry | undefined): boolean =>
  entry !== undefined && !entry.isFolder && entry.type === 'note' && entry.isNote

export const getFileName = (path: string): string => path.split('/').pop() ?? path

export function getFolderOf(path: string): string {
  const slash = path.lastIndexOf('/')
  return slash < 0 ? ROOT : path.slice(0, slash)
}

/** The folders above a path, from the root down. */
export function getParentFolders(path: string): readonly string[] {
  const parts = path.split('/').slice(0, -1)
  return parts.map((_, deep) => parts.slice(0, deep + 1).join('/'))
}

const createFolderEntry = (path: string, name: string): Entry => ({
  path,
  name,
  isFolder: true,
  isNote: false,
  type: 'note',
})

const createFileEntry = (path: string, type: Entry['type'], isNote: boolean): Entry => ({
  path,
  name: getFileName(path),
  isFolder: false,
  isNote,
  type,
})

function listFolders(folders: readonly string[], folder: string): Entry[] {
  const prefix = folder === ROOT ? '' : `${folder}/`
  return folders
    .filter((one) => one.startsWith(prefix) && !one.slice(prefix.length).includes('/'))
    .map((one) => {
      const name = one.slice(prefix.length)
      return createFolderEntry(one, folder === ROOT ? name.charAt(0).toUpperCase() + name.slice(1) : name)
    })
}

/** The media and decks the vault holds beside its notes, by folder. */
const FILES_BY_FOLDER: Readonly<Record<string, readonly Entry[]>> = {
  reading: Object.keys(PDF_DOCS).map((path) => createFileEntry(path, 'note', false)),
  lectures: [
    createFileEntry(RECORDING_PATH, 'note', false),
    createFileEntry(YOUTUBE_LINK_PATH, 'link', true),
  ],
  decks: [createFileEntry(DECK_PATH, 'deck', true)],
}

/** What a folder of the vault holds: the folders in it, then its notes, then its other files. */
export function listEntries(
  notes: readonly VaultNote[],
  folders: readonly string[],
  folder: string,
): readonly Entry[] {
  const shown = notes.filter((note) => note.folder === folder && !note.isHiddenFromFolderList)
  return [
    ...listFolders(folders, folder),
    ...shown.map((note) => createFileEntry(note.path, 'note', true)),
    ...(FILES_BY_FOLDER[folder] ?? []),
  ]
}
