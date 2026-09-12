/** The plex tab kind of a window. */
import { computed, type ShallowRef } from 'vue'
import { plexKind, usePlexView } from '@/pages/plex-graph'
import { CREATABLE } from '@/pages/note-editor'
import type { MessageWriter } from '@/shared/notices/messages'
import type { WindowKindsDeps } from './deps'

export interface PlexKindDeps
  extends Pick<
    WindowKindsDeps,
    'core' | 'puts' | 'held' | 'editing' | 'settings' | 'window' | 'where' | 'carries'
  > {
  dragged: ShallowRef<readonly string[]>
  told: MessageWriter
  asks: (text: string) => void
}

export function createPlexKind({
  core,
  puts,
  held,
  editing,
  settings,
  window,
  where,
  carries,
  dragged,
  told,
  asks,
}: PlexKindDeps) {
  return plexKind(held.handle, () => usePlexView(core), {
    makes: editing.making,
    ready: computed(() => !window.failure.value),
    hangs: settings.hungParts.hangs,
    parts: settings.hungParts.parts,
    opens: (path, title, showing, line) => void puts.opens(path, title, showing, line),
    inside: (paths) => core.headings(paths),
    asks,
    runs: (id, path, title) => carries(id, { ...where(), path, title }),
    opening: window.opening,
    first: () => window.first(),
    dragged,
    says: (text) => told(text, 'error'),
    writes: async () => (await editing.making.createUntitled('', []))?.path ?? '',
    creatable: CREATABLE,
  })
}
