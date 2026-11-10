import { reactive, ref } from 'vue'
import { listEntries, getFolderOf, type Entry } from '../lib/entries'
import { getNeighborhood, type Neighborhood, type Relation } from '../lib/neighborhood'
import { ENTROPY_PATH, INITIAL_EDGE_LABELS, INITIAL_FOLDERS, type VaultNote } from '../lib/notes'
import {
  getBlankNote,
  getInitialNotes,
  getMovedFolders,
  getNewFolderName,
  getNotePath,
  getRekeyedLabels,
  getRelatedNote,
  getUntitledTitle,
} from '../lib/vaultEdits'

/** The notes, folders and links of the demo vault, held in memory. */
export function createVault() {
  const notes = reactive<Record<string, VaultNote>>(getInitialNotes())
  const folders = ref<string[]>([...INITIAL_FOLDERS])
  const labels = reactive<Record<string, string>>({ ...INITIAL_EDGE_LABELS })
  const renameListeners: ((from: string, to: string) => void)[] = []

  const getNote = (path: string): VaultNote | undefined => notes[path]

  function getNoteOrEntropy(path: string): VaultNote {
    const note = notes[path] ?? notes[ENTROPY_PATH]
    if (!note) throw new Error(`the demo vault has no note at ${ENTROPY_PATH}`)
    return note
  }

  const list = (folder: string): readonly Entry[] =>
    listEntries(Object.values(notes), folders.value, folder)

  const getEntryAt = (path: string): Entry | undefined =>
    list(getFolderOf(path)).find((one) => one.path === path)

  const getNeighbors = (path: string): Neighborhood => getNeighborhood(notes, labels, path)

  const getTitleForNew = (folder: string): string => getUntitledTitle((path) => path in notes, folder)

  function addNote(note: Omit<VaultNote, 'path'>): string {
    const path = getNotePath(note.folder, note.title)
    notes[path] = { ...note, path }
    return path
  }

  const createUntitled = (folder: string): string =>
    addNote(getBlankNote(getTitleForNew(folder), folder))

  function createFolder(folder: string): string {
    const name = getNewFolderName(new Set(list(folder).map((one) => one.name)))
    const path = folder ? `${folder}/${name}` : name
    folders.value = [...folders.value, path]
    return path
  }

  /** A note made as a parent, a child or a jump of another, and the path it is filed under. */
  function createRelated(title: string, fromPath: string, relation: Relation): string {
    const from = getNoteOrEntropy(fromPath)
    const folder = from.folder || 'physics'
    const path = addNote(getRelatedNote(from, relation, title.trim() || getTitleForNew(folder), folder))
    if (relation === 'parent') from.parents = [...from.parents, path]
    return path
  }

  /** Applies a change to the links of every note. */
  function relinkAll(change: (paths: string[]) => string[]) {
    for (const other of Object.values(notes)) {
      other.parents = change(other.parents)
      other.jumps = change(other.jumps)
    }
  }

  /** The note at `from` filed under `to`, with every link and label that named it. */
  function renameNote(from: string, to: string) {
    const note = notes[from]
    if (!note) return
    const parts = to.split('/')
    const filename = parts.pop() ?? to
    delete notes[from]
    notes[to] = { ...note, path: to, title: filename.replace(/\.md$/, ''), folder: parts.join('/') }
    relinkAll((paths) => paths.map((one) => (one === from ? to : one)))
    const rekeyed = getRekeyedLabels(labels, from, to)
    for (const key of Object.keys(labels)) delete labels[key]
    Object.assign(labels, rekeyed)
    for (const listener of renameListeners) listener(from, to)
  }

  function moveFolder(from: string, to: string) {
    const prefix = `${from}/`
    folders.value = getMovedFolders(folders.value, from, to)
    for (const path of Object.keys(notes)) {
      if (path.startsWith(prefix)) renameNote(path, `${to}/${path.slice(prefix.length)}`)
    }
  }

  const move = (from: string, to: string) => (notes[from] ? renameNote(from, to) : moveFolder(from, to))

  /** Takes a note out of the vault and out of every link; false where it is the last note. */
  function removeNote(path: string): boolean {
    if (!notes[path] || Object.keys(notes).length <= 1) return false
    delete notes[path]
    relinkAll((paths) => paths.filter((one) => one !== path))
    return true
  }

  return {
    notes,
    labels,
    read: { getNote, getNoteOrEntropy, list, getEntryAt, getNeighbors },
    edit: { createUntitled, createFolder, createRelated, move, removeNote },
    onRename: (listener: (from: string, to: string) => void) => renameListeners.push(listener),
  }
}

export type Vault = ReturnType<typeof createVault>
