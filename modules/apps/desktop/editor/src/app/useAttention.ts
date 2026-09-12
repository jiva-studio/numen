/**
 * Attention reporting to the application and tab icon resolution.
 */
import { computed, watch } from 'vue'
import { iconOfKind } from '@/entities/tab'
import { AGENT } from '@/entities/tab'
import type { Attention } from '@/entities/tab'
import type { VaultPort } from '@/app/ports/vault'
import type { OpenTab, useWindowTabs } from '@/entities/tab'

export interface AttentionDeps {
  core: Pick<VaultPort, 'setFocus'>
  /** The window tabs manager. */
  tabs?: ReturnType<typeof useWindowTabs>
  held?: ReturnType<typeof useWindowTabs>
}

export function useAttention({ core, tabs, held }: AttentionDeps) {
  const windowTabsManager = (tabs ?? held)!
  /** Returns the ID of the front/active tab. */
  const getActiveTabId = (): string => {
    const at = windowTabsManager.handle.front()
    if (at && at.kind !== AGENT) return at.id
    const beside = [...windowTabsManager.tabs.value]
      .reverse()
      .find((one) => windowTabsManager.getTab(one.id)?.kind.kind !== AGENT)
    return beside?.id ?? at?.id ?? ''
  }
  const looked = getActiveTabId

  const getAttention = (): Attention => ({
    front: getActiveTabId(),
    tabs: windowTabsManager.tabs.value.map(({ id, title }) => {
      const one = windowTabsManager.getTab(id)
      const said = (one?.kind.getAttention?.(one.state) ?? one?.kind.attends?.(one.state)) as
        | OpenTab<'document' | 'recording' | 'book'>
        | undefined
      return {
        id,
        kind: one?.kind.kind ?? '',
        title,
        path: said?.path ?? '',
        ...(said?.document ? { document: said.document } : {}),
        ...(said?.recording ? { recording: said.recording } : {}),
        ...(said?.book ? { book: said.book } : {}),
      }
    }),
  })

  const attention = computed<Attention>(() => getAttention())
  watch(
    attention,
    (open) => {
      void core.setFocus(open).catch((why) => {
        console.error('what the window has open was not told:', why)
      })
    },
    { immediate: true },
  )

  const tabIcon = (id: string) => iconOfKind(windowTabsManager.getTab(id)?.kind.kind ?? '')

  return {
    attention,
    getActiveTabId,
    looked,
    tabIcon,
  }
}
