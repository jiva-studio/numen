// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import type { AgentPort, ConversationContext } from '@numen/plugin-sdk'

import AgentPanel from './AgentPanel.vue'
import { useAgentPanel } from '../model/panel'
import type { AgentPanelState } from '../model/panel'
import { MOUNT_VIEW, type MountView } from '../../../shared/views'
import type { ReviewCard } from '../../../entities/card'

/** A card as the session hands one over. */
const card: ReviewCard = {
  deck: 'decks/Words.md',
  section: '',
  cardId: '3f4g5h6j7k',
  template: 'Say it',
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isNew: false,
  ahead: null,
}

const agent: AgentPort = {
  async *ask() {},
  async finish() {},
}

/** The panel over one card, with the session around it standing in for it. */
const createPanel = (): AgentPanelState => {
  const open = ref(false)
  return useAgentPanel({
    agent: () => agent,
    card: () => card,
    unreachable: () => '',
    open: () => open.value,
    showPanel: (isOpen) => {
      open.value = isOpen
    },
    showNotice: () => {},
  })
}

/** A host that records each view mounted, writes its focus into the element, and counts focus calls. */
const createHost = () => {
  const mounts: { capability: string; context: ConversationContext; isDisposed: boolean }[] = []
  let focusCount = 0
  const mountView: MountView = (capability, element, context) => {
    const one = { capability, context: context as ConversationContext, isDisposed: false }
    mounts.push(one)
    const note = document.createElement('p')
    note.textContent = one.context.focus
    element.append(note)
    return {
      focus: () => focusCount++,
      dispose: () => {
        one.isDisposed = true
        note.remove()
      },
    }
  }
  return { mountView, mounts, getFocusCount: () => focusCount }
}

const mountPanel = (panel: AgentPanelState, mountView?: MountView) =>
  mount(AgentPanel, {
    props: { panel },
    attachTo: document.body,
    ...(mountView ? { global: { provide: { [MOUNT_VIEW as symbol]: mountView } } } : {}),
  })

describe('AgentPanel', () => {
  it('mounts the conversation view with the context of the card', async () => {
    const host = createHost()
    const panel = createPanel()
    const one = mountPanel(panel, host.mountView)

    panel.openPanel()
    await nextTick()

    expect(host.mounts.map((it) => it.capability)).toEqual(['conversation'])
    expect(host.mounts[0]?.context.focus).toBe('decks/Words.md')
    expect(one.text()).toBe('decks/Words.md')
  })

  // A panel opened is a panel opened to write in.
  it('focuses the view when the panel opens', async () => {
    const host = createHost()
    const panel = createPanel()
    mountPanel(panel, host.mountView)

    panel.openPanel()
    await nextTick()
    await nextTick()

    expect(host.getFocusCount()).toBe(1)
  })

  it('takes the view away when the conversation ends', async () => {
    const host = createHost()
    const panel = createPanel()
    const one = mountPanel(panel, host.mountView)
    panel.openPanel()
    await nextTick()

    panel.endConversation()
    await nextTick()

    expect(host.mounts[0]?.isDisposed).toBe(true)
    expect(one.text()).toBe('')
  })

  it('takes the view away with the panel', async () => {
    const host = createHost()
    const panel = createPanel()
    const one = mountPanel(panel, host.mountView)
    panel.openPanel()
    await nextTick()

    one.unmount()

    expect(host.mounts[0]?.isDisposed).toBe(true)
  })

  // A window where no plugin declares a conversation view still runs its review.
  it('stays empty where nothing mounts a view', async () => {
    const panel = createPanel()
    const one = mountPanel(panel)

    panel.openPanel()
    await nextTick()

    expect(one.find('section').element.childElementCount).toBe(0)
  })
})
