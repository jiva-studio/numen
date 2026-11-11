/**
 * What is currently open, reported to the application, and the icon each tab carries.
 */
import { computed, ref, watch, type Component, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { findTabKindIcon } from '../lib/icons'
import { writeOpenTabs as writeOpenTabsToCore } from '../api/saved-tabs'
import type { OpenTabs, SavedTab, Tab } from '../lib/tab'
import type { WindowTabs } from './windowTabs'
import { formatError, type Caught } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { JsonValue } from '@/shared/api/json'

export interface OpenTabsDeps {
  windowTabs: WindowTabs
  isWriting?: Readonly<Ref<boolean>>
  writeMessage: MessageWriter
  writeOpenTabs?: (open: OpenTabs) => Promise<void>
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

export function useOpenTabs({
  windowTabs,
  isWriting = ref(true),
  writeMessage,
  writeOpenTabs = writeOpenTabsToCore,
}: OpenTabsDeps) {
  const reportError = (error: Caught) =>
    writeMessage(`The open tabs were not saved: ${formatError(error)}`, 'error')

  /** Whether a tab is of a kind that takes text put to it: a conversation, with no file of its own. */
  const isTakingText = (id: string): boolean =>
    windowTabs.tabs.getTab(id)?.kind.events?.receiveText !== undefined

  /** The tab in front, and the last one that takes no text where the one in front does. */
  const getActiveTabId = (): string => {
    const frontTab = windowTabs.groups.handle.front()
    if (frontTab && !isTakingText(frontTab.id)) return frontTab.id
    const nextTabs = [...windowTabs.tabs.tabs.value].reverse().find((one) => !isTakingText(one.id))
    return nextTabs?.id ?? frontTab?.id ?? ''
  }

  const getOpenTabs = (): OpenTabs => ({
    activeId: getActiveTabId(),
    tabs: windowTabs.tabs.tabs.value.map(({ id, title }) => describeTab(windowTabs, id, title)),
  })

  const open = computed<OpenTabs>(() => getOpenTabs())
  let pendingWrite: OpenTabs | null = null
  let sending: Promise<void> | null = null
  const sendPending = async (): Promise<void> => {
    while (pendingWrite) {
      const current = pendingWrite
      pendingWrite = null
      try {
        await writeOpenTabs(current)
      } catch (error) {
        // The next tab opened, closed or moved writes the list again.
        reportError(error)
      }
    }
    sending = null
  }
  const flushOpenTabs = (next: OpenTabs): Promise<void> => {
    pendingWrite = next
    sending ??= sendPending()
    return sending
  }
  /** Whether the writes are held back, because the vault the tabs belong to is being left. */
  const isPaused = ref(false)
  watch(
    [open, isWriting, isPaused],
    ([now, isOn, isHeld]) => {
      if (!isOn || isHeld) return
      runInBackground(flushOpenTabs(now), reportError)
    },
    { immediate: true },
  )

  /**
   * Writes the tabs as they are now, waits until every write has reached the core, and holds
   * later writes back. A write made after the core shows another vault would be kept for that vault.
   */
  const pause = async (): Promise<void> => {
    if (isWriting.value) await flushOpenTabs(open.value)
    isPaused.value = true
  }

  /** Lets the tabs be written again, where the vault was not left. */
  const resume = (): void => {
    isPaused.value = false
  }

  /** Returns the declared icon of a kind, or its built-in icon. Tabs of other windows use it. */
  const getKindIcon = (kind: string): Component | null =>
    windowTabs.groups.findKind((one) => one.kind === kind)?.display?.icon ?? findTabKindIcon(kind)

  /** The icon the tab's kind declares, or the one the window keeps for that kind. */
  const getTabIcon = (id: string): Component | null => {
    const kind = windowTabs.tabs.getTab(id)?.kind
    // Stryker disable next-line StringLiteral: no kind and the empty kind both map to no icon
    return kind?.display?.icon ?? findTabKindIcon(kind?.kind ?? '')
  }

  return {
    open,
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
