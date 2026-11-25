import { computed, reactive, type Component, type ComputedRef } from 'vue'
import { Download } from '@lucide/vue'
import { getFileName } from '@/shared/lib/paths'
import { DOWNLOAD_KIND, formatDownloadFailure } from '../lib/downloadFailure'
import type { AnyTabKind, TabKind } from '../lib/kinds'
import { WORDS } from '../words'
import type { WindowTabs } from './windowTabs'

/** What a download tab holds: the file, its title, the failure that ended the download, and what retries it. */
export interface DownloadState {
  readonly path: string
  readonly view: ComputedRef<{ readonly title: string; readonly failure: string }>
  readonly retry: () => void
}

/** The tabs that stand for files while their bytes are brought onto this device. */
export interface DownloadTabs {
  /** Opens or focuses the tab of the file; false where a download of the file already runs. */
  start: (path: string, title: string) => Promise<boolean>
  /** Shows the failure of the download in the tab of the file. */
  fail: (path: string, error: unknown) => void
  /** Whether the tab of the file is open. */
  isOpen: (path: string) => boolean
  /** Ends the download of the file and closes its tab. */
  finish: (path: string) => void
}

interface Entry {
  title: string
  failure: string
}

/**
 * The tab kind of a file being downloaded and the operations on its tabs. A tab of the kind has one
 * per path; closing it does not stop the download. `retry` opens the file again.
 */
export function createDownloadTabs(
  pane: Component,
  windowTabs: WindowTabs,
  retry: (path: string, title: string) => void,
): { readonly kind: AnyTabKind; readonly tabs: DownloadTabs } {
  const entries = reactive(new Map<string, Entry>())
  const getTabIds = (path: string) =>
    windowTabs.handle
      .listTabs<DownloadState>(DOWNLOAD_KIND)
      .filter((one) => one.state.path === path)
      .map((one) => one.id)

  const kind: TabKind<DownloadState> = {
    kind: DOWNLOAD_KIND,
    open: (path) => {
      const view = computed(() => {
        const entry = entries.get(path)
        return {
          title: entry?.title || getFileName(path),
          failure: entry ? entry.failure : WORDS.downloadFailed,
        }
      })
      return { path, view, retry: () => retry(path, entries.get(path)?.title ?? '') }
    },
    pane,
    getTitle: (state) => state.view.value.title,
    display: { icon: Download },
    events: {
      onClose: (state) => {
        if (entries.get(state.path)?.failure) entries.delete(state.path)
        return true
      },
    },
    address: {
      identity: (path) => path,
      getOpenTab: (state) => ({ path: state.path }),
    },
  }

  const tabs: DownloadTabs = {
    start: async (path, title) => {
      const isRunning = entries.get(path)?.failure === ''
      entries.set(path, { title, failure: '' })
      await windowTabs.tabs.openTabOfKind(DOWNLOAD_KIND, path)
      return !isRunning
    },
    fail: (path, error) => {
      const entry = entries.get(path)
      if (!entry) return
      if (getTabIds(path).length === 0) entries.delete(path)
      else entry.failure = formatDownloadFailure(error)
    },
    isOpen: (path) => getTabIds(path).length > 0,
    finish: (path) => {
      entries.delete(path)
      for (const id of getTabIds(path)) windowTabs.handle.closeTab(id)
    },
  }
  return { kind, tabs }
}
