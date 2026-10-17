/** The plex tab kind of a window. */
import { computed, type ShallowRef } from 'vue'
import { usePlexView, type PlexTabState } from '@plugins/view-plex-graph/frontend'
import { CREATABLE } from '@/entities/note'
import type { MessageWriter } from '@/shared/notices/messages'
import { searchCore } from '../vault/search'
import type { WindowKindsDeps } from './deps'
import { createPlexKind as buildPlexKind } from './plexKind'

export interface PlexKindDeps extends Pick<
  WindowKindsDeps,
  | 'core'
  | 'tabOpeners'
  | 'held'
  | 'editing'
  | 'settings'
  | 'window'
  | 'getTarget'
  | 'runCommand'
  | 'tabLinks'
  | 'tabSync'
> {
  dragged: ShallowRef<readonly string[]>
  writeMessage: MessageWriter
  askAgent: (text: string) => void
}

function findPlexTabId(held: PlexKindDeps['held'], state?: PlexTabState): string | null {
  if (state) {
    const match = held.handle.each<PlexTabState>('plex').find((tab) => tab.state === state)
    if (match) return match.id
  }
  const front = held.handle.front()
  if (front?.kind === 'plex') return front.id
  return held.handle.last<PlexTabState>('plex')?.id ?? null
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
  tabSync,
}: PlexKindDeps) {
  return buildPlexKind(held.handle, () => usePlexView(core), {
    editor: editing.making,
    isReady: computed(() => !window.failure.value),
    isNesting: settings.nesting.isNesting,
    headings: settings.nesting.headings,
    openNote: (path, options) => {
      const title = options?.title ?? ''
      const tabId = findPlexTabId(held, options?.state)
      if (tabId && tabSync?.syncPathFromTab(tabId, path, title)) {
        return
      }
      void tabOpeners.openFile(path, title, { where: options?.showing, line: options?.line })
    },
    onSelectPath: (path, title, state) => {
      const tabId = findPlexTabId(held, state)
      if (tabId) {
        tabSync?.syncPathFromTab(tabId, path, title ?? '')
      }
    },
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
