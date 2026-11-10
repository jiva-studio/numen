/**
 * The tabs of the window: how they open, close, link and move between windows.
 */
import { formatError } from '@/shared/api/errors'
import { watch, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { getTabPaneRect } from '@numen/workbench'
import { getFileKinds } from '@/entities/file'
import {
  acceptTabMove,
  cancelTabLinksWatch,
  createWindowTabs,
  detachTab,
  fileOpeners,
  highlightWindow,
  moveTab,
  type MovedTab,
  RestoreFailedPane,
  UnavailablePane,
  useOpenTabs,
  watchTabLinks,
  writeTabLinks,
} from '@/entities/tab'
import { useTabLinks } from '@/features/tab-linking'
import { createCommandNotes } from '../commands/commandDeps'
import { createTabCommands } from '../tabs/tabCommands'
import { createTabLinkSession, getWindowReport } from '../tabs/tabLinkSession'
import { createMovedTabOpener, createTabMover } from '../tabs/tabMove'
import type { WindowCore, WindowServices } from './createDisplayPart'
import { isSecondaryWindow } from './createLifecyclePart'
import type { useSettings } from './useSettings'
import { focusWindow } from '@/entities/window'

/** The calls of the core that the tab links, the tab moves and the tab reports make. */
const core = {
  acceptTabMove,
  cancelTabLinksWatch,
  detachTab,
  highlightWindow,
  moveTab,
  watchTabLinks,
  writeTabLinks,
}

interface TabsCoreDeps {
  readonly writeMessage: WindowCore['writeMessage']
  readonly settingsHolder: { current?: ReturnType<typeof useSettings> }
  readonly pluginStarts: { current?: (pluginId: string) => boolean }
}

export function createTabsCore(deps: TabsCoreDeps) {
  const { writeMessage, settingsHolder, pluginStarts } = deps
  /* Stryker disable OptionalChaining,ObjectLiteral,StringLiteral,Regex: window core wiring delegates to mocked collaborators in unit tests */
  const windowTabs = createWindowTabs(
    { unavailable: UnavailablePane, restoreFailed: RestoreFailedPane },
    {
      isPluginStarting: (pluginId) => pluginStarts.current?.(pluginId) ?? true,
      enablePlugin: async (pluginId) => {
        await settingsHolder.current?.rest.writeSetting([pluginId, 'enabled'], true)
      },
      openPlugin: (pluginId) =>
        globalThis.dispatchEvent(new CustomEvent('numen:open-settings', { detail: { pluginId } })),
    },
  )
  const windowId = crypto.randomUUID()
  // The core ignores a link to a tab it has no report of, so a link or unlink
  // waits for the pending tab reports. The tab links session sets the wait.
  const tabReports = { wait: (): Promise<void> => Promise.resolve() }
  const tabLinks = useTabLinks({
    write: (change) => tabReports.wait().then(() => core.writeTabLinks(windowId, change)),
    onError: (error) => writeMessage(`The tabs were not linked: ${formatError(error)}`, 'error'),
  })
  const tabOpeners = fileOpeners({ fileKinds: getFileKinds, reconcileTab: windowTabs.reconcileTab })
  /* Stryker restore OptionalChaining,ObjectLiteral,StringLiteral,Regex */
  return { windowTabs, windowId, tabReports, tabLinks, tabOpeners }
}

export function createTabCloser(base: WindowCore) {
  return (id: string, hold?: () => void): boolean => {
    if (!base.windowTabs.releaseTab(id)) {
      hold?.()
      return false
    }
    if (!hold) base.windowTabs.finishClose(id)
    return true
  }
}

export function openPlacedTabs(base: WindowCore) {
  return runInBackground(
    base.windowTabs.openPlacedTabs((kind, error) =>
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
  const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
  const getReport = () => getWindowReport(base.windowTabs, isMain)
  const onError = (error: unknown) =>
    base.writeMessage(`The tab links were not written: ${formatError(error)}`, 'error')
  const session = createTabLinkSession({
    core,
    windowId: base.windowId,
    tabLinks: base.tabLinks,
    applySync: deps.tabSync.applySync,
    applyGroupPaths: deps.tabSync.applyGroupPaths,
    openMovedTab: (moved) => runInBackground(openMovedTab(moved), onError),
    setHighlighted: (isOn) => {
      deps.isHighlighted.value = isOn
    },
    getReport,
    wait,
    onError,
  })
  const openMovedTab = createMovedTabOpener({
    windowTabs: base.windowTabs,
    closeTab: deps.closeTab,
    acceptMove: session.acceptMove,
    focusWindow: () => focusWindow(),
    wait,
    onError: (error) =>
      base.writeMessage(`The moved tab did not open: ${formatError(error)}`, 'error'),
  })
  base.tabReports.wait = session.waitForReports
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
    ...session,
    openMovedTab: (moved: MovedTab) => runInBackground(openMovedTab(moved), onError),
  }
}

export function createWindowTabMoves(
  base: WindowCore,
  editing: WindowServices['editing'],
  closeTab: (id: string) => boolean,
) {
  const notes = createCommandNotes(editing.reached, base.windowTabs)
  const mover = createTabMover({
    core,
    windowId: base.windowId,
    windowTabs: base.windowTabs,
    notes,
    closeTab,
    writeMessage: base.writeMessage,
    getPaneRect: (id) => getTabPaneRect(base.windowTabs.layout.value, id),
  })
  /** The mover writes its own errors; a failed highlight request is ignored. */
  const ignoreError = () => undefined
  type Point = { readonly x: number; readonly y: number }
  // The core does not order requests, so each highlight waits for the previous one; an off
  // request then cannot arrive before the on request it ends.
  let lastHighlight: Promise<void> = Promise.resolve()
  return {
    windowId: base.windowId,
    isTabAsking: (id: string) => notes.asking(id),
    highlightWindow: (windowId: string, isOn: boolean) => {
      lastHighlight = lastHighlight
        .then(() => core.highlightWindow(windowId, isOn))
        .catch(ignoreError)
    },
    detachTab: (id: string, point: Point) =>
      runInBackground(mover.detachTab(id, point), ignoreError),
    detachTabInPlace: (id: string) => runInBackground(mover.detachTabInPlace(id), ignoreError),
    moveTab: (id: string, windowId: string) =>
      runInBackground(mover.moveTab(id, windowId), ignoreError),
  }
}

export function createWindowTabCommands(
  base: WindowCore,
  moves: ReturnType<typeof createWindowTabMoves>,
) {
  const { windowTabs, tabLinks } = base
  const tabCommands = createTabCommands({
    tabLinks,
    getTabs: () => windowTabs.tabs.value,
    getKind: (id) => windowTabs.getTab(id)?.kind.kind,
    isLinkable: (id) => windowTabs.getTab(id)?.kind.navigate !== undefined,
    windowId: moves.windowId,
    isAsking: moves.isTabAsking,
    detach: moves.detachTabInPlace,
    move: moves.moveTab,
  })
  tabCommands.register()
  return tabCommands
}

export const useWindowOpenTabs = (base: WindowCore, isWriting: Ref<boolean>) =>
  useOpenTabs({
    windowTabs: base.windowTabs,
    isWriting,
    reportError: (error) =>
      base.writeMessage(`The open tabs were not saved: ${formatError(error)}`, 'error'),
  })
