// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { AgentPort } from '@numen/plugin-sdk'
import ConversationView from './ConversationView.vue'

const agent: AgentPort = {
  async *ask() {
    yield* []
  },
  finish: async () => undefined,
}

describe('ConversationView', () => {
  it('names the field by the placeholder the caller passes', () => {
    const view = mount(ConversationView, {
      props: { context: { agent, focus: 'decks/Words.md', placeholder: 'Ask about this card' } },
    })
    expect(view.find('textarea').attributes('aria-label')).toBe('Ask about this card')
  })

  it('shows why the agent cannot be reached where the answers would be', () => {
    const view = mount(ConversationView, {
      props: {
        context: { agent, focus: '', getUnreachable: () => 'Agent unreachable.' },
      },
    })
    expect(view.text()).toContain('Agent unreachable.')
  })

  it('takes the keyboard into the field when the host focuses it', () => {
    const view = mount(ConversationView, {
      props: { context: { agent, focus: '' } },
      attachTo: document.body,
    })

    ;(view.vm as unknown as { focus: () => void }).focus()

    expect(document.activeElement).toBe(view.find('textarea').element)
    view.unmount()
  })
})
