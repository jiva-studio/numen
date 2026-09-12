/** The plex tab kind of a window. */
import { computed, type ShallowRef } from 'vue'
import { plexKind, usePlexView } from '@/pages/plex-graph'
import { CREATABLE } from '@/pages/note-editor'
import type { MessageWriter } from '@/shared/notices/messages'
import type { WindowKindsDeps } from './deps'

export interface PlexKindDeps
  extends Pick<
    WindowKindsDeps,
    'core' | 'tabOpeners' | 'held' | 'editing' | 'settings' | 'window' | 'where' | 'runCommand'
  > {
  dragged: ShallowRef<readonly string[]>
  told: MessageWriter
  askAgent: (text: string) => void
}

export function createPlexKind({
  core,
  tabOpeners,
  held,
  editing,
  settings,
  window,
  where,
  runCommand,
  dragged,
  told,
  askAgent,
}: PlexKindDeps) {
  return plexKind(held.handle, () => usePlexView(core), {
    editor: editing.making,
    ready: computed(() => !window.failure.value),
    hangs: settings.hungParts.hangs,
    parts: settings.hungParts.parts,
    openNote: (path, title, showing, line) => void tabOpeners.openFile(path, title, showing, line),
    inside: (paths) => core.headings(paths),
    askAgent,
    runCommand: (id, path, title) => runCommand(id, { ...where(), path, title }),
    opening: window.opening,
    first: () => window.first(),
    dragged,
    showMessage: (text) => told(text, 'error'),
    createUntitledNote: async () => (await editing.making.createUntitled('', []))?.path ?? '',
    creatable: CREATABLE,
  })
}
