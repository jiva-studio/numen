/**
 * The conversation view another plugin declares, mounted into the agent panel for as long as the
 * panel has a card's context, and given the keyboard when the panel opens.
 */
import { nextTick, onScopeDispose, watch, type Ref } from 'vue'
import { CONVERSATION, type ConversationContext, type MountedView } from '@numen/plugin-sdk'
import type { MountView } from '../../../shared/views'

export interface MountedConversationDeps {
  readonly element: Readonly<Ref<HTMLElement | null>>
  readonly context: () => ConversationContext | null
  readonly open: () => boolean
  readonly mountView: MountView
}

export function useMountedConversation(deps: MountedConversationDeps): void {
  let view: MountedView | null = null

  const unmount = () => {
    view?.dispose()
    view = null
  }

  watch(
    [deps.element, deps.context],
    ([element, context]) => {
      unmount()
      if (element && context) view = deps.mountView(CONVERSATION, element, context)
    },
    { immediate: true, flush: 'post' },
  )

  // A panel opened is a panel opened to write in, so the field takes the keyboard as it arrives.
  watch(deps.open, (isOpen) => {
    if (isOpen) void nextTick(() => view?.focus())
  })

  onScopeDispose(unmount)
}
