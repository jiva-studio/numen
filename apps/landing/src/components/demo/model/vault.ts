import { reactive, ref } from 'vue'
import { listEntries, getFolderOf, type Entry } from '../lib/entries'
import { getNeighbourhood, type Neighbourhood, type Relation } from '../lib/neighbourhood'
import {
  ENTROPY_PATH,
  INITIAL_EDGE_LABELS,
  INITIAL_FOLDERS,
  INITIAL_NOTES,
  type VaultNote,
} from '../lib/notes'

const NEW_NOTE_BODY =
  'Write Markdown here — headings, lists, math, and [[wikilinks]] format live as you move the caret.\n'

/** The notes, folders and links of the demo vault, held in memory. */
export function createVault() {
  const notes = reactive<Record<string, VaultNote>>(
    Object.fromEntries(
      INITIAL_NOTES.map((one) => [one.path, { ...one, parents: [...one.parents], jumps: [...one.jumps] }]),
    ),
  )
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

  const getNeighbours = (path: string): Neighbourhood => getNeighbourhood(notes, labels, path)

  function getUntitledTitle(folder: string): string {
    for (let count = 1; count < 1000; count += 1) {
      const title = count === 1 ? 'Untitled note' : `Untitled note ${count}`
      if (!notes[folder ? `${folder}/${title}.md` : `${title}.md`]) return title
    }
    return 'Untitled note'
  }

  function addNote(note: Omit<VaultNote, 'path'>): string {
    const path = note.folder ? `${note.folder}/${note.title}.md` : `${note.title}.md`
    notes[path] = { ...note, path }
    return path
  }

  const createUntitled = (folder: string): string =>
    addNote({ title: getUntitledTitle(folder), folder, parents: [], jumps: [], body: NEW_NOTE_BODY })

  function createFolder(folder: string): string {
    const taken = new Set(list(folder).map((one) => one.name))
    let name = 'New folder'
    for (let count = 2; taken.has(name); count++) name = `New folder ${count}`
    const path = folder ? `${folder}/${name}` : name
    folders.value = [...folders.value, path]
    return path
  }

  /** A note made as a parent, a child or a jump of another, and the path it is filed under. */
  function createRelated(title: string, fromPath: string, relation: Relation): string {
    const from = getNoteOrEntropy(fromPath)
    const folder = from.folder || 'physics'
    const cleanTitle = title.trim() || getUntitledTitle(folder)
    const path = addNote({
      title: cleanTitle,
      folder,
      parents: relation === 'child' ? [from.path] : [],
      jumps: relation === 'jump' ? [from.path] : [],
      body: `Created as a **${relation}** of [[${from.title}]].\n\n${NEW_NOTE_BODY}`,
    })
    if (relation === 'parent') from.parents = [...from.parents, path]
    return path
  }

  function rekeyLabels(from: string, to: string) {
    for (const [key, label] of Object.entries(labels)) {
      const [source, target] = key.split('->')
      if (source !== from && target !== from) continue
      delete labels[key]
      labels[`${source === from ? to : source}->${target === from ? to : target}`] = label
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
    for (const other of Object.values(notes)) {
      other.parents = other.parents.map((one) => (one === from ? to : one))
      other.jumps = other.jumps.map((one) => (one === from ? to : one))
    }
    rekeyLabels(from, to)
    for (const listener of renameListeners) listener(from, to)
  }

  function moveFolder(from: string, to: string) {
    const prefix = `${from}/`
    folders.value = folders.value.map((one) => {
      if (one === from) return to
      return one.startsWith(prefix) ? `${to}/${one.slice(prefix.length)}` : one
    })
    for (const path of Object.keys(notes)) {
      if (path.startsWith(prefix)) renameNote(path, `${to}/${path.slice(prefix.length)}`)
    }
  }

  const move = (from: string, to: string) => (notes[from] ? renameNote(from, to) : moveFolder(from, to))

  /** Takes a note out of the vault and out of every link; false where it is the last note. */
  function removeNote(path: string): boolean {
    if (!notes[path] || Object.keys(notes).length <= 1) return false
    delete notes[path]
    for (const other of Object.values(notes)) {
      other.parents = other.parents.filter((one) => one !== path)
      other.jumps = other.jumps.filter((one) => one !== path)
    }
    return true
  }

  return {
    notes,
    labels,
    read: { getNote, getNoteOrEntropy, list, getEntryAt, getNeighbours },
    edit: { createUntitled, createFolder, createRelated, move, removeNote },
    onRename: (listener: (from: string, to: string) => void) => renameListeners.push(listener),
  }
}

export type Vault = ReturnType<typeof createVault>
