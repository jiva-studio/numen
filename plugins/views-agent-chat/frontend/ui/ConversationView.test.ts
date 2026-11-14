// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ConversationView from './ConversationView.vue'
import { createAgentConnections } from '../fixtures'
import { WORDS } from '../words'

const { host: connections } = createAgentConnections({})

describe('ConversationView', () => {
  it('names the field by the placeholder the caller passes', () => {
    const view = mount(ConversationView, {
      props: { context: { connections, placeholder: 'Ask about this card' } },
    })
    expect(view.find('textarea').attributes('aria-label')).toBe('Ask about this card')
  })

  it('says there are no messages yet where the answers would be', () => {
    const view = mount(ConversationView, { props: { context: { connections } } })
    expect(view.text()).toContain(WORDS.noTranscript)
  })

  it('takes the keyboard into the field when the host focuses it', () => {
    const view = mount(ConversationView, {
      props: { context: { connections } },
      attachTo: document.body,
    })

    ;(view.vm as unknown as { focus: () => void }).focus()

    expect(document.activeElement).toBe(view.find('textarea').element)
    view.unmount()
  })
})
