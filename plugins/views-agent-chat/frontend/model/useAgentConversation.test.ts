/**
 * What pressing something in an agent tab evaluates to, asked without a screen.
 *
 * A question carries nothing about what is open, and a passage named in an
 * answer opens the source it is in.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref, shallowRef } from 'vue'
import type { Conversation, Turn } from '../thread'
import type { AgentModel, Span } from '@numen/plugin-sdk'
import { useAgentConversation } from './useAgentConversation'

type Model = AgentModel & { readonly nameAt: readonly string[] }

/** A conversation that records what it was asked, and where its lines point. */
const createConversation = (locations: Record<string, { path: string; span: Span }> = {}) => {
  const request: [string, string][] = []
  const stopCalls: string[] = []
  const turn = ref<Turn[]>([])
  const conversation: Conversation = {
    turns: turn,
    isWorking: ref(false),
    ask: async (text, focus) => {
      request.push([text, focus])
    },
    getSourceLocation: (turn) => locations[turn] ?? null,
    stop: () => stopCalls.push('stop'),
    finish: () => stopCalls.push('finish'),
  }
  return { conversation, turns: turn, requests: request, stopped: stopCalls }
}

/**
 * An agent tab with the window it is drawn in written down. The vault answers
 * with the notes it was handed, and reaches nothing for every other address.
 */
const tab = (
  locations: Record<string, { path: string; span: Span }> = {},
  notes: Record<string, string> = {},
) => {
  const fake = createConversation(locations)
  const openedCalls: [string, readonly Span[]][] = []
  const besideItems: string[] = []
  const state = useAgentConversation(fake.conversation, {
    openFileAt: (path, ...spans) => openedCalls.push([path, spans]),
    openFileBeside: (path) => besideItems.push(path),
    resolve: async (content) =>
      new Map(content.filter((one) => notes[one]).map((one) => [one, notes[one]!])),
    unreachable: () => '',
  })
  return { state, opened: openedCalls, besideItems, ...fake }
}

const items = (nameAt: readonly string[], name: string, presence: Model['presence']): Model => ({
  nameAt,
  name,
  title: name,
  isDefault: false,
  presence,
})
/** A line of an answer, as the panel hands one back. */
const turn = (id: string, text: string): Turn => ({ id, text, voice: 'response', state: 'settled' })

describe('a question sent', () => {
  it('is empty currently open, which the window reports itself', () => {
    const one = tab()

    one.state.send('what is this about')

    expect(one.requests).toEqual([['what is this about', '']])
  })

  it('empties the composer, so the question is not sent twice', () => {
    const one = tab()
    one.state.setQuestion('half a question')

    one.state.send('half a question')

    expect(one.state.userQuestion.value).toBe('')
  })
})

describe('a line about work pressed', () => {
  it('opens the place that call was on', () => {
    const one = tab({ call: { path: 'Source.pdf', span: { from: 10, to: 14 } } })

    one.state.openTurnSource(turn('call', 'read Source.pdf'))

    expect(one.opened).toEqual([['Source.pdf', [{ from: 10, to: 14 }]]])
  })

  it('opens nothing for a line that names no place', () => {
    const one = tab()

    one.state.openTurnSource(turn('said', 'a sentence'))

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

    one.state.followLink(turn('said', text), 'numen:Source.pdf?start=10&length=4', press.press)

    expect(press.was()).toBe(true)
    expect(one.opened).toEqual([
      [
        'Source.pdf',
        [
          { from: 10, to: 14 },
          { from: 90, to: 92 },
        ],
      ],
    ])
  })

  it('is left alone when it names nowhere in the vault', () => {
    const one = tab()
    const press = createPress()

    one.state.followLink(
      turn('said', 'read https://example.com'),
      'https://example.com',
      press.press,
    )

    expect(press.was()).toBe(false)
    expect(one.opened).toEqual([])
  })

  it('opens the note it names along with what the person is looking at', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.turns.value = [turn('said', 'It sits under [[Thermodynamics]].')]
    await nextTick()
    const press = createPress()

    one.state.followLink(one.turns.value[0]!, 'name://Thermodynamics', press.press)

    expect(one.besideItems).toEqual(['physics/Thermodynamics.md'])
  })

  it('opens nothing, and goes nowhere else, where no note answers to it', async () => {
    const one = tab()
    one.turns.value = [turn('said', 'It sits under [[Nowhere]].')]
    await nextTick()
    const press = createPress()

    one.state.followLink(one.turns.value[0]!, 'name://Nowhere', press.press)

    expect(one.besideItems).toEqual([])
  })
})

describe('a turn of an answer', () => {
  it('says which of the notes it names reach nothing', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.turns.value = [turn('said', 'Under [[Thermodynamics]], beside [[Nowhere]].')]

    await nextTick()
    await nextTick()

    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Nowhere'])
  })
})

describe('agent model selection and settings', () => {
  it('offers nothing where the settings list nothing', () => {
    const one = tab()
    expect(one.state.modelOptions.value).toEqual([])
    expect(one.state.selectedAgentId.value).toBe('')
    expect(one.state.selectedModelId.value).toBe('')
  })

  it('selects the agent and model without mutating settings', () => {
    const fake = createConversation()
    const content: [readonly string[], unknown][] = []
    const state = useAgentConversation(fake.conversation, {
      openFileAt: () => {},
      openFileBeside: () => {},
      resolve: async () => new Map(),
      unreachable: () => '',
      writeSetting: async (path, value) => {
        content.push([path, value])
      },
    })

    state.selectModel({
      agentId: 'zeta',
      agentTitle: 'Zeta Agent',
      modelId: 'zeta-small',
      modelTitle: 'Zeta Small',
      isAvailable: true,
    })

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual([
      'zeta',
      'zeta-small',
    ])
    expect(content).toEqual([])
  })

  it('selects a model without a writeSetting dependency', () => {
    const one = tab()
    one.state.selectModel({
      agentId: 'zeta',
      agentTitle: 'Zeta Agent',
      modelId: 'zeta-small',
      modelTitle: 'Zeta Small',
      isAvailable: true,
    })

    expect(one.state.selectedModelId.value).toBe('zeta-small')
  })

  it('offers the models the settings list, with the core title and default', async () => {
    const fake = createConversation()
    const models = shallowRef<readonly Model[]>([])
    const state = useAgentConversation(fake.conversation, {
      openFileAt: () => {},
      openFileBeside: () => {},
      resolve: async () => new Map(),
      unreachable: () => '',
      getSetting: (path) => (path[1] === 'use' ? 'zeta' : undefined),
      getModelsAt: (categoryPath) =>
        models.value.filter((one) => one.nameAt.join('.') === categoryPath.join('.')),
    })
    expect(state.selectedAgentId.value).toBe('')

    models.value = [
      { ...items(['agent', 'use'], 'zeta', 'present'), title: 'Zeta Agent' },
      items(['agent', 'use'], 'omega', 'not fetched'),
      {
        ...items(['agent', 'zeta', 'model'], 'zeta-small', 'nothing to fetch'),
        title: 'Zeta Small',
      },
      {
        ...items(['agent', 'zeta', 'model'], 'zeta-large', 'nothing to fetch'),
        title: 'Zeta Large',
        isDefault: true,
      },
    ]
    await nextTick()

    const optionsList = state.modelOptions.value.map((o) => [o.agentTitle, o.modelId, o.modelTitle])
    expect(optionsList).toEqual([
      ['Zeta Agent', 'zeta-small', 'Zeta Small'],
      ['Zeta Agent', 'zeta-large', 'Zeta Large'],
    ])
    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual([
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
    const [first, second] = one.state.turns.value

    one.turns.value = [one.turns.value[0]!, one.turns.value[1]!, turn('c', 'Nothing')]
    await nextTick()
    await nextTick()

    expect(one.state.turns.value[0]).toBe(first)
    expect(one.state.turns.value[1]).toBe(second)
  })

  it('draws a turn again when its unresolved addresses change', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.turns.value = [turn('a', 'See [[Nowhere]].')]
    await nextTick()
    await nextTick()
    const first = one.state.turns.value[0]

    one.turns.value = [turn('a', 'See [[Nowhere]] and [[Elsewhere]].')]
    await nextTick()
    await nextTick()

    expect(one.state.turns.value[0]).not.toBe(first)
    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Nowhere', 'name://Elsewhere'])
  })
})
