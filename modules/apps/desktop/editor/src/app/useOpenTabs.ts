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
  held: ReturnType<typeof useWindowTabs>
}

export function useOpenTabs({ core, held }: OpenTabsDeps) {
  /** Returns the ID of the front/active tab. */
  const getActiveTabId = (): string => {
    const at = held.handle.front()
    if (at && at.kind !== AGENT) return at.id
    const beside = [...held.tabs.value]
      .reverse()
      .find((one) => held.getTab(one.id)?.kind.kind !== AGENT)
    return beside?.id ?? at?.id ?? ''
  }

  const getOpenTabs = (): OpenTabs => ({
    front: getActiveTabId(),
    tabs: held.tabs.value.map(({ id, title }) => {
      const one = held.getTab(id)
      const said = one?.kind.getOpenTab?.(one.state) as
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

  const open = computed<OpenTabs>(() => getOpenTabs())
  watch(
    open,
    (now) => {
      void core.writeOpenTabs(now).catch((why) => {
        console.error('what the window has open was not told:', why)
      })
    },
    { immediate: true },
  )

  const tabIcon = (id: string) => iconOfKind(held.getTab(id)?.kind.kind ?? '')

  return {
    open,
    getActiveTabId,
    tabIcon,
  }
}
