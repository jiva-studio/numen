/**
 * Attention reporting to the application and tab icon resolution.
 */
import { computed, watch } from 'vue'
import { iconOfKind } from '../shared/icons'
import { AGENT } from '../shared/tabs/workspace'
import type { Attention, Core } from '../shared/core'
import type { OpenTab, windowTabs } from '../shared/tabs/windowTabs'

export interface AttentionDeps {
  core: Core
  held: ReturnType<typeof windowTabs>
}

export function useAttention({ core, held }: AttentionDeps) {
  const looked = (): string => {
    const at = held.handle.front()
    if (at && at.kind !== AGENT) return at.id
    const beside = [...held.tabs.value]
      .reverse()
      .find((one) => held.heldIn(one.id)?.kind.kind !== AGENT)
    return beside?.id ?? at?.id ?? ''
  }

  const attends = (): Attention => ({
    front: looked(),
    tabs: held.tabs.value.map(({ id, title }) => {
      const one = held.heldIn(id)
      const said = one?.kind.attends?.(one.state) as
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

  const attention = computed<Attention>(() => attends())
  watch(
    attention,
    (open) => {
      void core.attending(open).catch((why) => {
        console.error('what the window has open was not told:', why)
      })
    },
    { immediate: true },
  )

  const tabIcon = (id: string) => iconOfKind(held.heldIn(id)?.kind.kind ?? '')

  return {
    attention,
    looked,
    tabIcon,
  }
}
