/**
 * The state of a plugin's note tab bound to the window's store of open notes.
 *
 * The store writes after typing settles, against the file the tab last read. A file that changed on
 * disk under unsaved text pauses the saving until the person keeps their text or takes the file's;
 * a file that changed with nothing unsaved is shown at once, and a renamed file is followed.
 */
import { computed, onScopeDispose, shallowReactive, watch } from 'vue'
import type { EditorHandle } from '@numen/plugin-sdk'
import { extractFilename, stripMarkdownExtension } from './pane/pluginView'
import type { NoteBuffer, PluginHostDeps } from './types'

/** The tab kind whose tabs hold a note of the vault. */
export const NOTE_KIND = 'note'

const getDefaultName = (path: string): string => stripMarkdownExtension(extractFilename(path))

/** Why the saving of a note stopped: its file changed on disk, or is no longer there. */
export type FileConflict = 'stale' | 'gone' | null

const getFileConflict = (state: string): FileConflict =>
  state === 'stale' || state === 'gone' ? state : null

export interface BufferedNoteOptions {
  readonly pluginId: string
  readonly title?: string
  readonly deps?: PluginHostDeps | undefined
}

/** One note a wikilink can name, as the editor offers it. */
interface LinkChoice {
  readonly title: string
  readonly type: 'note'
}

/** The note a buffered tab opens: its path, the id it has in the store, and the store. */
export interface BufferedNoteTarget {
  readonly targetPath: string
  readonly id: string
  readonly buffer: NoteBuffer
}

function createWikilinks(deps: PluginHostDeps | undefined, getPath: () => string) {
  return {
    search: async (query: string): Promise<readonly LinkChoice[]> =>
      ((await deps?.getNotes?.()?.findNotes(query, 20)) ?? []).map((one) => ({
        title: one.title,
        type: 'note' as const,
      })),
    headings: async (noteTitle: string): Promise<readonly string[]> => {
      const notes = deps?.getNotes?.()
      if (!notes) return []
      const found = noteTitle
        ? (await notes.findNotes(noteTitle, 5)).find(
            (one) => one.title.toLowerCase() === noteTitle.toLowerCase(),
          )?.path
        : getPath()
      if (!found) return []
      return ((await notes.getHeadings([found])).get(found) ?? []).map((one) => one.text)
    },
  }
}

function createLinkFollower(deps: PluginHostDeps | undefined, getPath: () => string) {
  return (address: string): void => {
    const resolve = deps?.resolveLink
    const workspace = deps?.getWorkspace?.()
    if (!resolve || !workspace) return
    resolve(getPath(), address)
      .then((path) => {
        if (path) workspace.files.open(path, { placement: 'split' })
      })
      .catch(() => undefined)
  }
}

/**
 * Opens the note at `targetPath` in the store under the tab id, and closes it there when the tab's
 * scope ends. The store writes what is unsaved before it lets the note go.
 */
export function createBufferedNoteState(
  { targetPath, id, buffer }: BufferedNoteTarget,
  options: BufferedNoteOptions,
) {
  buffer.open(id, targetPath)
  onScopeDispose(() => {
    buffer.close(id).catch(() => undefined)
  })

  const note = computed(() => buffer.getOpenNote(id))
  let editor: EditorHandle | null = null

  const state = shallowReactive({
    at: targetPath,
    path: targetPath,
    pluginId: options.pluginId,
    title: options.title || getDefaultName(targetPath),
    note: computed((): { path: string; body: string } => ({
      path: note.value.path || state.path,
      body: note.value.body,
    })),
    body: computed(() => note.value.body),
    isLoading: computed(() => note.value.state === 'loading'),
    errorMessage: computed(() => buffer.getErrorMessage(id)),
    wikilinks: createWikilinks(options.deps, (): string => state.path),
    followLink: createLinkFollower(options.deps, (): string => state.path),
    fileConflict: computed(() => getFileConflict(note.value.state)),
    keepMine: () => buffer.keep(id),
    takeFile: () => buffer.take(id),
    attachEditor: (attached: EditorHandle | null): void => {
      editor = attached
    },
    measure: (): void => editor?.measure?.(),
    updateBody: (body: string) => buffer.setBody(id, body),
    save: () => buffer.save(id),
    navigate: (nextPath: string, nextTitle?: string): void => {
      if (!nextPath || nextPath === state.path) return
      buffer
        .flushPending(id)
        .catch(() => undefined)
        .then(() => buffer.reload(id, nextPath))
        .catch(() => undefined)
      setPath(nextPath, nextTitle)
    },
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
