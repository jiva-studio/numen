/**
 * The agent tab where the plugin has no agent row. With no agent connection at all the question fails,
 * says so, and offers the settings page of the plugin; with an agent connection the core finds, it is
 * answered.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AgentTab from './AgentTab.vue'
import { useConversation } from '../thread'
import { useAgentConversation } from '../model/useAgentConversation'
import { createAgentConnections, failWith, say, type TaskScript } from '../fixtures'
import { WORDS } from '../words'

const NO_AGENT: TaskScript = { end: failWith('no_connection', 'no connection serves acp.task') }

const mountTab = (script: TaskScript = NO_AGENT) => {
  let settingsOpened = 0
  const { host } = createAgentConnections(script)
  const state = useAgentConversation(
    useConversation({
      connections: host,
      words: WORDS,
      conversation: 'conversation:empty',
      paint: (draw) => draw(),
    }),
    {
      openFile: () => undefined,
      openFileAt: () => undefined,
      openFileAdjacent: () => undefined,
      resolveLinks: async () => [],
      openSettings: () => {
        settingsOpened += 1
      },
    },
  )
  return { wrapper: mount(AgentTab, { props: { state } }), state, getOpened: () => settingsOpened }
}

const ask = async (tab: ReturnType<typeof mountTab>) => {
  tab.state.draft.send('How many pear trees grow by the gate?')
  while (tab.state.conversation.isAnswerPending.value) await new Promise((go) => setTimeout(go, 0))
  await tab.wrapper.vm.$nextTick()
}

describe('the agent tab with no agent set up', () => {
  it('says no agent is set up under the question', async () => {
    const tab = mountTab()

    await ask(tab)

    expect(tab.wrapper.text()).toContain('No agent is set up.')
    expect(tab.wrapper.find('textarea').exists()).toBe(true)
  })

  it('opens the settings of the plugin from the failed turn', async () => {
    const tab = mountTab()
    await ask(tab)

    const button = tab.wrapper
      .findAll('button')
      .find((one) => one.text() === WORDS.openSettings)!
    await button.trigger('click')

    expect(tab.getOpened()).toBe(1)
  })
})

describe('the agent tab with an agent connection and no agent row', () => {
  it('answers the question and does not say that no agent is set up', async () => {
    const tab = mountTab({ frames: [say('Seven pear trees grow by the gate.')] })

    await ask(tab)

    expect(tab.wrapper.text()).toContain('Seven pear trees grow by the gate.')
    expect(tab.wrapper.text()).not.toContain('No agent is set up.')
  })
})
