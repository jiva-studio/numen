/**
 * What currently open, told to the application, and the icon each tab carries.
 */
import { computed, watch, type Component } from 'vue'
import { findTabKindIcon } from '@/entities/tab'
import type { OpenTabs } from '@/entities/tab'
import type { VaultPort } from '@/app/ports/vault'
import type { OpenTab, useWindowTabs } from '@/entities/tab'

export interface OpenTabsDeps {
  core: Pick<VaultPort, 'writeOpenTabs'>
  /** The window tabs manager. */
  windowTabs: ReturnType<typeof useWindowTabs>
}

/** What a tab points at, where what it holds is kept in the vault. */
const getTabSource = (tab: OpenTab<'document' | 'recording' | 'book'> | undefined) => ({
  ...(tab?.document ? { document: tab.document } : {}),
  ...(tab?.recording ? { recording: tab.recording } : {}),
  ...(tab?.book ? { book: tab.book } : {}),
})

export function useOpenTabs({ core, windowTabs }: OpenTabsDeps) {
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
    front: getActiveTabId(),
    tabs: windowTabs.tabs.value.map(({ id, title }) => {
      const one = windowTabs.getTab(id)
      const openTab = one?.kind.getOpenTab?.(one.state) as
        OpenTab<'document' | 'recording' | 'book'> | undefined
      return {
        id,
        kind: one?.kind.kind ?? '',
        title,
        path: openTab?.path ?? '',
        ...getTabSource(openTab),
      }
    }),
  })

  const open = computed<OpenTabs>(() => getOpenTabs())
  watch(
    open,
    (now) => {
      // A list that did not reach the core is written again by the next tab
      // opened, closed or moved, and the window draws from its own state
      // meanwhile. There is nothing here for a person to do.
      void core.writeOpenTabs(now).catch(() => {})
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
