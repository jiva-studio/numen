/**
 * The send/stop disc of an agent tab across one question and its answer, rendered with the
 * transitions of the real component and an agent whose steps arrive one task apart.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import type { AgentPort, AgentStep } from '@numen/plugin-sdk'
import AgentTab from './AgentTab.vue'
import { useConversation } from '../thread'
import { useAgentConversation } from '../model/useAgentConversation'
import { WORDS } from '../words'

const QUESTION = 'How many pear trees grow in the east orchard?'

const ANSWER: readonly AgentStep[] = [
  { kind: 'thinking' },
  { kind: 'toolCall', tool: 'note_search', subject: 'East orchard', written: 0 },
  { kind: 'answered' },
  { kind: 'text', text: 'The east orchard ' },
  { kind: 'text', text: 'has twelve pear trees.' },
  { kind: 'stopped', error: '' },
]

const waitTask = () => new Promise<void>((resolve) => setTimeout(resolve, 0))

const waitTransitions = () => new Promise<void>((resolve) => setTimeout(resolve, 200))

const createPort = (
  steps: readonly AgentStep[],
  onStep: (index: number) => void,
  isHeld = false,
): AgentPort => ({
  async *ask(query) {
    for (const [index, step] of steps.entries()) {
      await waitTask()
      if (query.signal.aborted) return
      onStep(index)
      yield step
    }
    if (!isHeld) return
    await new Promise<void>((resolve) => query.signal.addEventListener('abort', () => resolve()))
  },
  finish: async () => undefined,
})

const readDisc = (root: Element): string => {
  const buttons = [...root.querySelectorAll('.disc button')]
  return buttons
    .map((button) => {
      const action = button.getAttribute('aria-label') === WORDS.stop ? 'stop' : 'send'
      return `${action}:${button.hasAttribute('disabled') ? 'barred' : 'pressable'}`
    })
    .join('+')
}

const mounted: { unmount: () => void }[] = []

afterEach(() => {
  for (const one of mounted.splice(0)) one.unmount()
})

const mountTab = (createAgent: (sample: () => void) => AgentPort) => {
  const values: string[] = []
  let element: Element | null = null
  const sample = () => {
    if (!element) return
    const value = readDisc(element)
    if (values.at(-1) !== value) values.push(value)
  }
  const state = useAgentConversation(
    useConversation({
      agent: createAgent(sample),
      words: WORDS,
      conversation: 'conversation:orchard',
      paint: (draw) => draw(),
    }),
    {
      openFileAt: () => undefined,
      openFileAdjacent: () => undefined,
      resolveLinks: async () => [],
      unreachable: () => '',
      settings: {
        getModelsAt: (path) =>
          path.join('/') === 'app/agent/use'
            ? [{ name: 'orchard', title: 'Orchard', isDefault: true, presence: 'present' }]
            : [{ name: 'main', title: 'Main', isDefault: true, presence: 'present' }],
      },
    },
  )
  const wrapper = mount(AgentTab, {
    props: { state },
    attachTo: document.body,
    global: { stubs: { transition: false } },
  })
  mounted.push(wrapper)
  const observer = new MutationObserver(sample)
  const record = () => {
    element = wrapper.element
    observer.observe(wrapper.element, { subtree: true, childList: true, attributes: true })
    sample()
  }
  return { state, wrapper, values, record, sample }
}

describe('the disc across one answer', () => {
  it('shows send, then stop for as long as the answer is on its way, then send', async () => {
    const tab = mountTab((sample) => createPort(ANSWER, sample))
    await tab.wrapper.get('textarea').setValue(QUESTION)
    tab.record()

    await tab.wrapper.get('.disc button').trigger('click')
    tab.sample()
    expect(tab.state.conversation.isAnswerPending.value).toBe(true)
    expect(readDisc(tab.wrapper.element)).toBe('stop:pressable')
    while (tab.state.conversation.isAnswerPending.value) await waitTask()
    await waitTransitions()

    expect(tab.values).toEqual(['send:pressable', 'stop:pressable', 'send:barred'])
  })

  it('shows send once stop is pressed in the middle of the answer', async () => {
    let press = () => undefined as unknown
    const tab = mountTab((sample) =>
      createPort(
        ANSWER.slice(0, 4),
        (index) => {
          sample()
          if (index === 3) setTimeout(() => press(), 0)
        },
        true,
      ),
    )
    press = () => tab.wrapper.get('.disc button').trigger('click')
    await tab.wrapper.get('textarea').setValue(QUESTION)
    tab.record()

    await tab.wrapper.get('.disc button').trigger('click')
    tab.sample()
    while (tab.state.conversation.isAnswerPending.value) await waitTask()
    await waitTransitions()

    expect(tab.values).toEqual(['send:pressable', 'stop:pressable', 'send:barred'])
    expect(tab.state.addresses.turns.value.map((turn) => turn.text)).toContain('The east orchard ')
  })
})
