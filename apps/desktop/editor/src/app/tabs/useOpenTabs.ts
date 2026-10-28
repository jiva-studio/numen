/**
 * What currently open, told to the application, and the icon each tab carries.
 */
import { computed, ref, watch, type Component, type Ref } from 'vue'
import { findTabKindIcon } from '@/entities/tab'
import type { OpenTabs } from '@/entities/tab'
import type { VaultPort } from '@/app/ports/vault'
import type { SavedTab, useWindowTabs } from '@/entities/tab'

export interface OpenTabsDeps {
  core: Pick<VaultPort, 'writeOpenTabs'>
  /** The window tabs manager. */
  windowTabs: ReturnType<typeof useWindowTabs>
  /**
   * Whether this window writes its tabs. It is false until the tabs the core kept are open again,
   * and stays false in a window a tab was detached into.
   */
  isWriting?: Readonly<Ref<boolean>>
}

/** The state a kind saved, as JSON, and empty where it saved none. */
const formatSavedState = (saved: unknown): string =>
  saved === undefined || saved === null ? '' : JSON.stringify(saved)

export function useOpenTabs({ core, windowTabs, isWriting = ref(true) }: OpenTabsDeps) {
  /** Whether a tab is of a kind that takes text put to it: a conversation, with no file of its own. */
  const isTakingText = (id: string): boolean =>
    windowTabs.getTab(id)?.kind.receiveText !== undefined

  /** The tab in front, and the last one that takes no text where the one in front does. */
  const getActiveTabId = (): string => {
    const frontTab = windowTabs.handle.front()
    if (frontTab && !isTakingText(frontTab.id)) return frontTab.id
    const nextTabs = [...windowTabs.tabs.value].reverse().find((one) => !isTakingText(one.id))
    return nextTabs?.id ?? frontTab?.id ?? ''
  }

  const getOpenTabs = (): OpenTabs => ({
    activeId: getActiveTabId(),
    tabs: windowTabs.tabs.value.map(({ id, title }) => {
      const one = windowTabs.getTab(id)
      return {
        id,
        kind: one?.kind.kind ?? '',
        title,
        path: one?.kind.getOpenTab?.(one.state).path ?? '',
        state: formatSavedState(one && one.kind.getState?.(one.state)),
      }
    }),
  })

  const open = computed<OpenTabs>(() => getOpenTabs())
  let pendingWrite: OpenTabs | null = null
  let isSending = false
  const flushOpenTabs = async (next: OpenTabs): Promise<void> => {
    pendingWrite = next
    if (isSending) return
    isSending = true
    while (pendingWrite) {
      const current = pendingWrite
      pendingWrite = null
      try {
        await core.writeOpenTabs(current)
      } catch {
        // A list that did not reach the core is written again by the next tab
        // opened, closed or moved, and the window draws from its own state
        // meanwhile. There is nothing here for a person to do.
      }
    }
    isSending = false
  }
  watch(
    [open, isWriting],
    ([now, isOn]) => {
      if (!isOn) return
      void flushOpenTabs(now)
    },
    { immediate: true },
  )

  /** The icon the tab's kind declares, or the one the window keeps for that kind. */
  const getTabIcon = (id: string): Component | null => {
    const kind = windowTabs.getTab(id)?.kind
    // Stryker disable next-line StringLiteral: no kind and the empty kind both map to no icon
    return kind?.icon ?? findTabKindIcon(kind?.kind ?? '')
  }

  return {
    open,
    getActiveTabId,
    getTabIcon,
  }
}

/** The state a kind saved, parsed from its JSON, and nothing where it is empty or not JSON. */
const parseSavedState = (state: string): unknown => {
  if (!state) return undefined
  try {
    return JSON.parse(state) as unknown
  } catch {
    return undefined
  }
}

/** The tabs the core kept, as the window opens them again. */
export const parseSavedTabs = (open: OpenTabs): readonly SavedTab[] =>
  open.tabs
    .filter((one) => one.id !== '' && one.kind !== '')
    .map(({ id, kind, path, title, state }) => ({
      id,
      kind,
      path,
      ...(title ? { title } : {}),
      saved: parseSavedState(state),
    }))
