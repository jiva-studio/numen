/**
 * What is currently open, reported to the application, and the icon each tab carries.
 */
import { computed, ref, type Component, type Ref } from 'vue'
import { getPanes } from '@numen/workbench'
import { findTabKindIcon } from '../lib/icons'
import type { SavedTabsApi } from '../api/saved-tabs'
import type { OpenTabs, SavedTab, Tab } from '../lib/tab'
import { useTabPersistence } from './useTabPersistence'
import type { WindowTabs } from './windowTabs'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { JsonValue } from '@/shared/api/json'

export interface OpenTabsDeps {
  windowTabs: WindowTabs
  isWriting?: Readonly<Ref<boolean>>
  writeMessage: MessageWriter
  writeOpenTabs: SavedTabsApi['writeOpenTabs']
}

/** The state a kind saved, as JSON, and empty where it saved none. */
export const formatSavedState = (saved: JsonValue | undefined): string =>
  saved === undefined ? '' : JSON.stringify(saved)

function describeTab(windowTabs: WindowTabs, id: string, title: string): Tab {
  const one = windowTabs.tabs.getTab(id)
  if (!one) return { id, kind: '', title, path: '', state: '' }
  const openTab = one.kind.address?.getOpenTab?.(one.state)
  return {
    id,
    kind: one.kind.kind,
    title,
    path: openTab?.path ?? '',
    state: formatSavedState(one.kind.address?.getState?.(one.state)),
    ...(openTab?.place ? { place: openTab.place } : {}),
  }
}

function findActiveTabId(windowTabs: WindowTabs): string {
  const isTakingText = (id: string): boolean =>
    windowTabs.tabs.getTab(id)?.kind.events?.receiveText !== undefined
  const isTool = (id: string): boolean => windowTabs.tabs.getTab(id)?.kind.display?.isTool === true
  const frontTab = windowTabs.handle.getFrontTab()
  if (frontTab && !isTakingText(frontTab.id)) {
    if (!isTool(frontTab.id)) return frontTab.id
    const shown = new Set(getPanes(windowTabs.panes.layout.value.root).map((one) => one.active))
    const beside = [...windowTabs.tabs.tabs.value]
      .reverse()
      .find((one) => shown.has(one.id) && !isTool(one.id) && !isTakingText(one.id))
    return beside?.id ?? frontTab.id
  }
  const nextTabs = [...windowTabs.tabs.tabs.value].reverse().find((one) => !isTakingText(one.id))
  return nextTabs?.id ?? frontTab?.id ?? ''
}

export function useOpenTabs({
  windowTabs,
  isWriting = ref(true),
  writeMessage,
  writeOpenTabs,
}: OpenTabsDeps) {
  const reportError = (error: unknown) =>
    writeMessage(`The open tabs were not saved: ${formatError(error)}`, 'error')

  /**
   * The tab in front. Where that is a tool, the tab of another pane that shows and was shown last,
   * because the core keeps one active tab per window and a tool is only what the person worked
   * from. Where the tab in front takes text, the last tab that takes no text.
   */
  const getActiveTabId = (): string => findActiveTabId(windowTabs)

  const savedLayout = computed(() => JSON.stringify(windowTabs.panes.layout.value))
  const getOpenTabs = (): OpenTabs => ({
    activeId: getActiveTabId(),
    layout: savedLayout.value,
    tabs: windowTabs.tabs.tabs.value.map(({ id, title }) => describeTab(windowTabs, id, title)),
  })

  const open = computed<OpenTabs>(() => getOpenTabs())
  const { flush, pause, resume } = useTabPersistence(open, isWriting, writeOpenTabs, reportError)

  /** Returns the declared icon of a kind, or its built-in icon. Tabs of other windows use it. */
  const getKindIcon = (kind: string): Component | null =>
    windowTabs.kinds.findKind((one) => one.kind === kind)?.display?.icon ?? findTabKindIcon(kind)

  /** The icon the tab's kind declares, or the one the window keeps for that kind. */
  const getTabIcon = (id: string): Component | null => {
    const kind = windowTabs.tabs.getTab(id)?.kind
    // Stryker disable next-line StringLiteral: no kind and the empty kind both map to no icon
    return kind?.display?.icon ?? findTabKindIcon(kind?.kind ?? '')
  }

  return {
    open,
    flush,
    pause,
    resume,
    getActiveTabId,
    getTabIcon,
    getKindIcon,
  }
}

const parseSavedState = (state: string): JsonValue | undefined => {
  /* Stryker disable BlockStatement: falling off catch returns undefined */
  try {
    return JSON.parse(state)
  } catch {
    return undefined
  }
  /* Stryker restore BlockStatement */
}

/** Converts a stored tab to the saved tab the window opens. */
export const parseSavedTab = ({ id, kind, path, title, state }: Tab): SavedTab => ({
  id,
  kind,
  path,
  ...(title ? { title } : {}),
  saved: parseSavedState(state),
})

/** The tabs the core kept, as the window opens them again. */
export const parseSavedTabs = (open: OpenTabs): readonly SavedTab[] =>
  open.tabs.filter((one) => one.id !== '' && one.kind !== '').map(parseSavedTab)
