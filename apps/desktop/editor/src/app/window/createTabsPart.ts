/**
 * The tabs of the window: how they open, close, link and move between windows.
 */
import { formatError, type Caught } from '@/shared/api/errors'
import { watch, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { getTabPaneRect } from '@numen/workbench'
import { getFileKinds } from '@/entities/file'
import {
  createWindowTabs,
  fileOpeners,
  type MovedTab,
  RestoreFailedPane,
  UnavailablePane,
  useOpenTabs,
} from '@/entities/tab'
import {
  createTabLinkSession,
  createTabReports,
  getWindowReport,
  highlightWindow,
  useTabLinks,
  writeTabLinks,
} from '@/features/tab-linking'
import { acceptTabMove, createMovedTabOpener, createTabMover } from '@/features/tab-move'
import { createCommandNotes } from '../tabs/commandDeps'
import type { TabsCore, WindowCore } from './windowCore'
import type { WindowServices } from './createWindowServices'
import { isSecondaryWindow } from './createLifecyclePart'
import type { SettingsStore } from '@/entities/settings'
import { focusWindow } from '@/entities/window'

interface TabsCoreDeps {
  readonly writeMessage: WindowCore['writeMessage']
  readonly rest: SettingsStore
  readonly isPluginStarting: (pluginId: string) => boolean
}

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export function createTabsCore(deps: TabsCoreDeps): TabsCore {
  const { writeMessage, rest } = deps
  const windowTabs = createWindowTabs(
    { unavailable: UnavailablePane, restoreFailed: RestoreFailedPane },
    {
      isPluginStarting: deps.isPluginStarting,
      enablePlugin: async (pluginId) => {
        await rest.write.writeSetting([pluginId, 'enabled'], true)
      },
      openPlugin: (pluginId) =>
        globalThis.dispatchEvent(new CustomEvent('numen:open-settings', { detail: { pluginId } })),
    },
  )
  const windowId = crypto.randomUUID()
  const reports = createTabReports({
    windowId,
    getReport: () => getWindowReport(windowTabs, !isSecondaryWindow()),
    wait,
    onError: (error) =>
      writeMessage(`The tab links were not written: ${formatError(error)}`, 'error'),
  })
  // The core ignores a link to a tab it has no report of, so a link or unlink
  // waits for the pending tab reports.
  const tabLinks = useTabLinks({
    write: (change) => reports.waitForReports().then(() => writeTabLinks(windowId, change)),
    onError: (error) => writeMessage(`The tabs were not linked: ${formatError(error)}`, 'error'),
  })
  const tabOpeners = fileOpeners({
    fileKinds: getFileKinds,
    reconcileTab: windowTabs.panes.reconcileTab,
  })
  return { windowTabs, windowId, reports, tabLinks, tabOpeners }
}

export function createTabCloser(base: WindowCore) {
  return (id: string, hold?: () => void): boolean => {
    if (!base.tabs.windowTabs.panes.releaseTab(id)) {
      hold?.()
      return false
    }
    if (!hold) base.tabs.windowTabs.panes.finishClose(id)
    return true
  }
}

export function openPlacedTabs(base: WindowCore) {
  return runInBackground(
    base.tabs.windowTabs.restored.openPlacedTabs((kind, error) =>
      base.writeMessage(`The ${kind} tab did not open: ${formatError(error)}`, 'error'),
    ),
    (error) => base.writeMessage(`The tabs did not open: ${formatError(error)}`, 'error'),
  )
}

interface WindowTabLinksDeps {
  readonly tabSync: WindowServices['tabSync']
  readonly closeTab: (id: string) => void
  readonly isHighlighted: Ref<boolean>
}

export function createWindowTabLinks(base: WindowCore, deps: WindowTabLinksDeps) {
  const isMain = !isSecondaryWindow()
  const getReport = () => getWindowReport(base.tabs.windowTabs, isMain)
  const onError = (error: Caught) =>
    base.writeMessage(`The tab links were not written: ${formatError(error)}`, 'error')
  const session = createTabLinkSession({
    acceptTabMove,
    windowId: base.tabs.windowId,
    tabLinks: base.tabs.tabLinks,
    applySync: deps.tabSync.applySync,
    applyGroupPaths: deps.tabSync.applyGroupPaths,
    openMovedTab: (moved) => runInBackground(openMovedTab(moved), onError),
    setHighlighted: (isOn) => {
      deps.isHighlighted.value = isOn
    },
    reports: base.tabs.reports,
    wait,
    onError,
  })
  const openMovedTab = createMovedTabOpener({
    windowTabs: base.tabs.windowTabs,
    closeTab: deps.closeTab,
    acceptMove: session.acceptMove,
    focusWindow: () => focusWindow(),
    wait,
    onError: (error) =>
      base.writeMessage(`The moved tab did not open: ${formatError(error)}`, 'error'),
  })
  watch(() => JSON.stringify(getReport()), session.reportTabs)
  // Hiding the page cancels the watch in the core; a page restored from the
  // back-forward cache watches again.
  globalThis.addEventListener('pagehide', session.close)
  globalThis.addEventListener('pageshow', (event) => {
    if (!event.persisted) return
    runInBackground(session.reopen(), (error) =>
      base.writeMessage(`The tab links stopped: ${formatError(error)}`, 'error'),
    )
  })
  return {
    start: session.start,
    close: session.close,
    openMovedTab: (moved: MovedTab) => runInBackground(openMovedTab(moved), onError),
  }
}

export function createWindowTabMoves(
  base: WindowCore,
  editing: WindowServices['editing'],
  closeTab: (id: string) => boolean,
) {
  const notes = createCommandNotes(editing.reached, base.tabs.windowTabs)
  const mover = createTabMover({
    windowId: base.tabs.windowId,
    windowTabs: base.tabs.windowTabs,
    notes,
    closeTab,
    writeMessage: base.writeMessage,
    getPaneRect: (id) => getTabPaneRect(base.tabs.windowTabs.panes.layout.value, id),
  })
  /** The mover writes its own errors; a failed highlight request is ignored. */
  const ignoreError = () => undefined
  type Point = { readonly x: number; readonly y: number }
  // The core does not order requests, so each highlight waits for the previous one; an off
  // request then cannot arrive before the on request it ends.
  let lastHighlight: Promise<void> = Promise.resolve()
  return {
    windowId: base.tabs.windowId,
    isTabAsking: (id: string) => notes.asking(id),
    highlightWindow: (windowId: string, isOn: boolean) => {
      lastHighlight = lastHighlight.then(() => highlightWindow(windowId, isOn)).catch(ignoreError)
    },
    detachTab: (id: string, point: Point) =>
      runInBackground(mover.detachTab(id, point), ignoreError),
    detachTabInPlace: (id: string) => runInBackground(mover.detachTabInPlace(id), ignoreError),
    moveTab: (id: string, windowId: string) =>
      runInBackground(mover.moveTab(id, windowId), ignoreError),
  }
}

export const createWindowOpenTabs = (base: WindowCore, isWriting: Ref<boolean>) =>
  useOpenTabs({
    windowTabs: base.tabs.windowTabs,
    isWriting,
    reportError: (error) =>
      base.writeMessage(`The open tabs were not saved: ${formatError(error)}`, 'error'),
  })
