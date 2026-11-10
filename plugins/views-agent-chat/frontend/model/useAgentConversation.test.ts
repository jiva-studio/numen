/**
 * The result of activating an element in an agent tab, tested without a screen.
 *
 * A question does not depend on what is open, and a passage named in an answer
 * opens the source it is in.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref, shallowRef } from 'vue'
import type { Conversation } from '../thread'
import type { Turn } from '../types'
import type { AgentModel, TextRange } from '@numen/plugin-sdk'
import { useAgentConversation } from './useAgentConversation'
import { createLinkResolver } from '../fixtures/links'

type Model = AgentModel & { readonly nameAt: readonly string[] }

const createConversation = (locations: Record<string, { path: string; range: TextRange }> = {}) => {
  const request: [string, string][] = []
  const stopCalls: string[] = []
  const turn = ref<Turn[]>([])
  const conversation: Conversation = {
    turns: turn,
    isAnswerPending: ref(false),
    ask: async (text, focus) => {
      request.push([text, focus])
    },
    getSourceLocation: (turn) => locations[turn] ?? null,
    stop: () => stopCalls.push('stop'),
    finish: () => stopCalls.push('finish'),
  }
  return { conversation, turns: turn, requests: request, stopped: stopCalls }
}

const tab = (
  locations: Record<string, { path: string; range: TextRange }> = {},
  notes: Record<string, string> = {},
) => {
  const fake = createConversation(locations)
  const openedCalls: [string, readonly TextRange[]][] = []
  const adjacentItems: string[] = []
  const state = useAgentConversation(fake.conversation, {
    openFileAt: (path, ...spans) => openedCalls.push([path, spans]),
    openFileAdjacent: (path) => adjacentItems.push(path),
    resolveLinks: createLinkResolver(notes),
    unreachable: () => '',
  })
  return { state, opened: openedCalls, adjacentItems, ...fake }
}

const items = (nameAt: readonly string[], name: string, presence: Model['presence']): Model => ({
  nameAt,
  name,
  title: name,
  isDefault: false,
  presence,
})
const turn = (id: string, text: string): Turn => ({ id, text, voice: 'response', state: 'settled' })

describe('a question sent', () => {
  it('is empty currently open, which the window reports itself', () => {
    const one = tab()

    one.state.draft.send('what is this about')

    expect(one.requests).toEqual([['what is this about', '']])
  })

  it('empties the composer, so the question is not sent twice', () => {
    const one = tab()
    one.state.draft.setQuestion('half a question')

    one.state.draft.send('half a question')

    expect(one.state.draft.userQuestion.value).toBe('')
  })
})

describe('a line about work pressed', () => {
  it('opens the place that call was on', () => {
    const one = tab({ call: { path: 'Source.pdf', range: { start: 10, end: 14 } } })

    one.state.draft.openTurnSource(turn('call', 'read Source.pdf'))

    expect(one.opened).toEqual([['Source.pdf', [{ start: 10, end: 14 }]]])
  })

  it('opens nothing for a line that names no place', () => {
    const one = tab()

    one.state.draft.openTurnSource(turn('turn-1', 'a sentence'))

    expect(one.opened).toEqual([])
  })
})

describe('a link inside an answer', () => {
  const createPress = () => {
    let prevented = false
    const press = { preventDefault: () => (prevented = true) } as unknown as MouseEvent
    return { press, was: () => prevented }
  }

  it('opens the place it names, with the other places that answer names in it', () => {
    const one = tab()
    const text =
      'see [here](numen:Source.pdf?start=10&length=4) and [there](numen:Source.pdf?start=90&length=2)'
    const press = createPress()

    one.state.addresses.followLink(
      turn('turn-1', text),
      'numen:Source.pdf?start=10&length=4',
      press.press,
    )

    expect(press.was()).toBe(true)
    expect(one.opened).toEqual([
      [
        'Source.pdf',
        [
          { start: 10, end: 14 },
          { start: 90, end: 92 },
        ],
      ],
    ])
  })

  it('is left alone when it names nowhere in the vault', () => {
    const one = tab()
    const press = createPress()

    one.state.addresses.followLink(
      turn('turn-1', 'read https://example.com'),
      'https://example.com',
      press.press,
    )

    expect(press.was()).toBe(false)
    expect(one.opened).toEqual([])
  })

  it('opens the note it names along with what the person is looking at', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.turns.value = [turn('turn-1', 'It sits under [[Thermodynamics]].')]
    await nextTick()
    const press = createPress()

    one.state.addresses.followLink(one.turns.value[0]!, 'name://Thermodynamics', press.press)

    expect(one.adjacentItems).toEqual(['physics/Thermodynamics.md'])
  })

  it('opens nothing, and goes nowhere else, where no note answers to it', async () => {
    const one = tab()
    one.turns.value = [turn('turn-1', 'It sits under [[Nowhere]].')]
    await nextTick()
    const press = createPress()

    one.state.addresses.followLink(one.turns.value[0]!, 'name://Nowhere', press.press)

    expect(one.adjacentItems).toEqual([])
  })
})

describe('a turn of an answer', () => {
  it('says which of the notes it names reach nothing', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.turns.value = [turn('turn-1', 'Under [[Thermodynamics]], beside [[Nowhere]].')]

    await nextTick()
    await nextTick()

    expect(one.state.addresses.turns.value[0]?.unresolved).toEqual(['name://Nowhere'])
  })
})

describe('agent model selection and settings', () => {
  it('offers nothing where the settings list nothing', () => {
    const one = tab()
    expect(one.state.models.modelOptions.value).toEqual([])
    expect(one.state.models.selectedAgentId.value).toBe('')
    expect(one.state.models.selectedModelId.value).toBe('')
  })

  it('selects the agent and model without mutating settings', () => {
    const fake = createConversation()
    const content: [readonly string[], unknown][] = []
    const state = useAgentConversation(fake.conversation, {
      openFileAt: () => undefined,
      openFileAdjacent: () => undefined,
      resolveLinks: async () => [],
      unreachable: () => '',
      settings: {
        writeSetting: async (path, value) => {
          content.push([path, value])
        },
      },
    })

    state.models.selectModel({
      agentId: 'zeta',
      agentTitle: 'Zeta Agent',
      modelId: 'zeta-small',
      modelTitle: 'Zeta Small',
      isAvailable: true,
    })

    expect([state.models.selectedAgentId.value, state.models.selectedModelId.value]).toEqual([
      'zeta',
      'zeta-small',
    ])
    expect(content).toEqual([])
  })

  it('selects a model without a writeSetting dependency', () => {
    const one = tab()
    one.state.models.selectModel({
      agentId: 'zeta',
      agentTitle: 'Zeta Agent',
      modelId: 'zeta-small',
      modelTitle: 'Zeta Small',
      isAvailable: true,
    })

    expect(one.state.models.selectedModelId.value).toBe('zeta-small')
  })

  it('offers the models the settings list, with the core title and default', async () => {
    const fake = createConversation()
    const models = shallowRef<readonly Model[]>([])
    const state = useAgentConversation(fake.conversation, {
      openFileAt: () => undefined,
      openFileAdjacent: () => undefined,
      resolveLinks: async () => [],
      unreachable: () => '',
      settings: {
        getSetting: (path) => (path[2] === 'use' ? 'zeta' : undefined),
        getModelsAt: (categoryPath) =>
          models.value.filter((one) => one.nameAt.join('.') === categoryPath.join('.')),
      },
    })
    expect(state.models.selectedAgentId.value).toBe('')

    models.value = [
      { ...items(['app', 'agent', 'use'], 'zeta', 'present'), title: 'Zeta Agent' },
      items(['app', 'agent', 'use'], 'omega', 'not fetched'),
      {
        ...items(['app', 'agent', 'zeta', 'model'], 'zeta-small', 'nothing to fetch'),
        title: 'Zeta Small',
      },
      {
        ...items(['app', 'agent', 'zeta', 'model'], 'zeta-large', 'nothing to fetch'),
        title: 'Zeta Large',
        isDefault: true,
      },
    ]
    await nextTick()

    const optionsList = state.models.modelOptions.value.map((o) => [
      o.agentTitle,
      o.modelId,
      o.modelTitle,
    ])
    expect(optionsList).toEqual([
      ['Zeta Agent', 'zeta-small', 'Zeta Small'],
      ['Zeta Agent', 'zeta-large', 'Zeta Large'],
    ])
    expect([state.models.selectedAgentId.value, state.models.selectedModelId.value]).toEqual([
      'zeta',
      'zeta-large',
    ])
  })
})

describe('turns drawn again', () => {
  it('keeps the turn it drew while its unresolved addresses are the same', async () => {
    const one = tab()
    one.turns.value = [turn('a', 'See [[Nowhere]].'), turn('b', 'Then [[Elsewhere]].')]
    await nextTick()
    await nextTick()
    const [first, second] = one.state.addresses.turns.value

    one.turns.value = [one.turns.value[0]!, one.turns.value[1]!, turn('c', 'Nothing')]
    await nextTick()
    await nextTick()

    expect(one.state.addresses.turns.value[0]).toBe(first)
    expect(one.state.addresses.turns.value[1]).toBe(second)
  })

  it('draws a turn again when its unresolved addresses change', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.turns.value = [turn('a', 'See [[Nowhere]].')]
    await nextTick()
    await nextTick()
    const first = one.state.addresses.turns.value[0]

    one.turns.value = [turn('a', 'See [[Nowhere]] and [[Elsewhere]].')]
    await nextTick()
    await nextTick()

    expect(one.state.addresses.turns.value[0]).not.toBe(first)
    expect(one.state.addresses.turns.value[0]?.unresolved).toEqual([
      'name://Nowhere',
      'name://Elsewhere',
    ])
  })
})
