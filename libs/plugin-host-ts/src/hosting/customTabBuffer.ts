/**
 * The state of a plugin's note tab bound to the window's store of open notes.
 *
 * The store writes after typing settles, against the file the tab last read. A file that changed on
 * disk under unsaved text pauses the saving until the person keeps their text or takes the file's;
 * a file that changed with nothing unsaved is shown at once, and a renamed file is followed.
 */
import { computed, onScopeDispose, shallowReactive, watch } from 'vue'
import { extractFilename, stripMarkdownExtension } from './pluginView'
import type { NoteBuffer, PluginHostDeps } from './types'

/** The tab kind whose tabs hold a note of the vault. */
export const NOTE_KIND = 'note'

/** Why the saving of a note stopped: its file changed on disk, or is no longer there. */
export type FileConflict = 'stale' | 'gone' | null

const getFileConflict = (state: string): FileConflict =>
  state === 'stale' || state === 'gone' ? state : null

const getDefaultName = (path: string): string => stripMarkdownExtension(extractFilename(path))

/** The fields of a buffered note tab, which a tab's own actions cannot replace. */
export const BUFFERED_FIELDS = ['fileConflict', 'keepMine', 'takeFile', 'wikilinks', 'followLink'] as const

export interface BufferedNoteOptions {
  readonly pluginId: string
  readonly title?: string
  readonly actions: Record<string, unknown>
  readonly deps?: PluginHostDeps | undefined
}

/** One note a wikilink can name, as the editor offers it. */
export interface LinkChoice {
  readonly title: string
  readonly type: 'note'
}

/**
 * Opens the note at `targetPath` in the store under the tab id, and closes it there when the tab's
 * scope ends. The store writes what is unsaved before it lets the note go.
 */
export function createBufferedNoteState(
  targetPath: string,
  id: string,
  buffer: NoteBuffer,
  options: BufferedNoteOptions,
) {
  buffer.open(id, targetPath)
  onScopeDispose(() => {
    buffer.close(id).catch(() => undefined)
  })

  const note = computed(() => buffer.getOpenNote(id))

  const state = shallowReactive({
    at: targetPath,
    path: targetPath,
    pluginId: options.pluginId,
    title: options.title || getDefaultName(targetPath),
    note: computed(() => ({ path: note.value.path, body: note.value.body })),
    body: computed(() => note.value.body),
    isLoading: computed(() => note.value.state === 'loading'),
    errorMessage: computed(() => buffer.getErrorMessage(id)),
    wikilinks: {
      search: async (query: string): Promise<readonly LinkChoice[]> =>
        ((await options.deps?.getNotes?.()?.findNotes(query, 20)) ?? []).map((one) => ({
          title: one.title,
          type: 'note',
        })),
      headings: async (noteTitle: string): Promise<readonly string[]> => {
        const notes = options.deps?.getNotes?.()
        if (!notes) return []
        const found = noteTitle
          ? (await notes.findNotes(noteTitle, 5)).find((one) => one.title.toLowerCase() === noteTitle.toLowerCase())?.path
          : state.path
        if (!found) return []
        return ((await notes.getHeadings([found])).get(found) ?? []).map((one) => one.text)
      },
    },
    followLink: (address: string): void => {
      const resolve = options.deps?.resolveLink
      const workspace = options.deps?.getWorkspace?.()
      if (!resolve || !workspace) return
      resolve(state.path, address)
        .then((path) => {
          if (path) workspace.openFile(path, { placement: 'split' })
        })
        .catch(() => undefined)
    },
    fileConflict: computed(() => getFileConflict(note.value.state)),
    keepMine: () => buffer.keep(id),
    takeFile: () => buffer.take(id),
    updateBody: (body: string) => buffer.setBody(id, body),
    save: () => buffer.save(id),
    navigate: (nextPath: string, nextTitle?: string) => {
      if (!nextPath || nextPath === state.path) return
      buffer
        .flushPending(id)
        .catch(() => undefined)
        .then(() => buffer.reload(id, nextPath))
        .catch(() => undefined)
      setPath(nextPath, nextTitle)
    },
    ...options.actions,
  })

  function setPath(nextPath: string, nextTitle?: string): void {
    state.at = nextPath
    state.path = nextPath
    state.title = nextTitle || getDefaultName(nextPath)
  }

  watch(
    () => note.value.path,
    (moved) => {
      if (moved && moved !== state.path) setPath(moved)
    },
  )

  return state
}
