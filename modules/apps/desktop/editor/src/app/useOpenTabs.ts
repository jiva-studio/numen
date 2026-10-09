/**
 * What the person has open, told to the application, and the icon each tab carries.
 */
import { computed, watch } from 'vue'
import { iconOfKind } from '@/entities/tab'
import { AGENT } from '@/entities/tab'
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
  /** Returns the ID of the front/active tab. */
  const getActiveTabId = (): string => {
    const at = windowTabs.handle.front()
    if (at && at.kind !== AGENT) return at.id
    const beside = [...windowTabs.tabs.value]
      .reverse()
      .find((one) => windowTabs.getTab(one.id)?.kind.kind !== AGENT)
    return beside?.id ?? at?.id ?? ''
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

  const tabIcon = (id: string) => iconOfKind(windowTabs.getTab(id)?.kind.kind ?? '')

  return {
    open,
    getActiveTabId,
    tabIcon,
  }
}
