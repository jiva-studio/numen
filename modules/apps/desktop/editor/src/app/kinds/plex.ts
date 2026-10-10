/** The plex tab kind of a window. */
import { computed, type ShallowRef } from 'vue'
import { createPlexKind as buildPlexKind, usePlexView } from '@/pages/plex-graph'
import { CREATABLE } from '@/pages/note-editor'
import type { MessageWriter } from '@/shared/notices/messages'
import { searchCore } from '../vault/search'
import type { WindowKindsDeps } from './deps'

export interface PlexKindDeps extends Pick<
  WindowKindsDeps,
  'core' | 'tabOpeners' | 'held' | 'editing' | 'settings' | 'window' | 'getTarget' | 'runCommand'
> {
  dragged: ShallowRef<readonly string[]>
  writeMessage: MessageWriter
  askAgent: (text: string) => void
}

export function createPlexKind({
  core,
  tabOpeners,
  held,
  editing,
  settings,
  window,
  getTarget,
  runCommand,
  dragged,
  writeMessage,
  askAgent,
}: PlexKindDeps) {
  return buildPlexKind(held.handle, () => usePlexView(core), {
    editor: editing.making,
    isReady: computed(() => !window.failure.value),
    isHanging: settings.hungParts.isHanging,
    parts: settings.hungParts.parts,
    openNote: (path, title, showing, line) => void tabOpeners.openFile(path, title, showing, line),
    readHeadings: (paths) => core.headings(paths),
    askAgent,
    runCommand: (id, path, title) => runCommand(id, { ...getTarget(), path, title }),
    openingPath: window.opening,
    readOpeningPath: () => window.readInitialNote(),
    dragged,
    showMessage: (text) => writeMessage(text, 'error'),
    createUntitledNote: async () => (await editing.making.createUntitled('', []))?.path ?? '',
    creatable: CREATABLE,
    searchNames: async (query, limit = 20) => {
      const results = await searchCore.names(query, limit)
      return results.map((r) => ({ path: r.path, title: r.title }))
    },
  })
}
