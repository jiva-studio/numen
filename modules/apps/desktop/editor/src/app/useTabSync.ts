/**
 * Synchronizes active path selection across linked tabs in the window.
 */
import type { useWindowTabs } from '@/entities/tab'
import type { useNoteEditors } from '@/app/useNoteEditors'
import type { PlexTabState } from '@plugins/view-plex-graph/frontend'
import type { FilesTabState } from '@/pages/file-manager'
import type { TabId, TabLinksState } from '@/features/tab-linking'

export interface TabSyncDeps {
  held: ReturnType<typeof useWindowTabs>
  tabLinks: TabLinksState
  editing: ReturnType<typeof useNoteEditors>
}

export function isNotePath(path: string): boolean {
  return path.endsWith('.md') || !path.includes('.')
}

interface SyncTargetContext {
  targetId: TabId
  targetTab: NonNullable<ReturnType<ReturnType<typeof useWindowTabs>['getTab']>>
  editing: ReturnType<typeof useNoteEditors>
}

function syncNoteTab(path: string, title: string, context: SyncTargetContext): void {
  if (!isNotePath(path)) return
  const { targetId, targetTab, editing } = context
  const noteId =
    (targetTab.state as { id?: string } | undefined)?.id ?? targetId.replace(/^note:/, '')
  void editing.noted.loadInTab(noteId, path, title)
}

function syncPlexTab(targetId: TabId, path: string, held: ReturnType<typeof useWindowTabs>): void {
  if (!isNotePath(path)) return
  const plexState = held.getTabStateIn<PlexTabState>(targetId, 'plex')
  if (plexState && plexState.view.here.value !== path) {
    void plexState.view.go(path)
  }
}

function syncFilesTab(targetId: TabId, path: string, held: ReturnType<typeof useWindowTabs>): void {
  const filesState = held.getTabStateIn<FilesTabState>(targetId, 'files')
  if (filesState) {
    void filesState.list.revealPath(path)
  }
}

interface SyncDestination {
  targetId: TabId
  path: string
  title: string
}

function syncToTab(
  destination: SyncDestination,
  held: ReturnType<typeof useWindowTabs>,
  editing: ReturnType<typeof useNoteEditors>,
): void {
  const { targetId, path, title } = destination
  const targetTab = held.getTab(targetId)
  if (!targetTab) return

  switch (targetTab.kind.kind) {
    case 'note':
      syncNoteTab(path, title, { targetId, targetTab, editing })
      break
    case 'plex':
      syncPlexTab(targetId, path, held)
      break
    case 'files':
      syncFilesTab(targetId, path, held)
      break
  }
}

export function useTabSync({ held, tabLinks, editing }: TabSyncDeps) {
  let isSyncing = false

  const syncPathFromTab = (sourceTabId: TabId, path: string, title = ''): boolean => {
    if (!sourceTabId || !path || isSyncing) return false
    const targets = tabLinks.getLinkedTargets(sourceTabId)
    if (targets.length === 0) return false

    isSyncing = true
    try {
      for (const targetId of targets) {
        syncToTab({ targetId, path, title }, held, editing)
      }
      return true
    } finally {
      isSyncing = false
    }
  }

  return { syncPathFromTab }
}
