/**
 * What pressing something in an agent tab comes to, asked without a screen.
 *
 * A question carries nothing about what is open, and a passage named in an
 * answer opens the source it is in.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref, shallowRef } from 'vue'
import type { Conversation, Turn } from '@numen/ui'
import { createTabTitle, useAgentConversation, type AgentTabState } from './useAgentConversation'
import { createAgentKind } from '../kind'
import type { Span } from '@/shared/span'
import type { Model } from '@/entities/settings'
import { useWindowTabs, type WindowHandle } from '@/entities/tab'
import { AGENT } from '@/entities/tab'

/** A conversation that records what it was asked, and where its lines point. */
const createConversation = (locations: Record<string, { path: string; span: Span }> = {}) => {
  const asked: [string, string][] = []
  const stopped: string[] = []
  const said = ref<Turn[]>([])
  const conversation: Conversation = {
    turns: said,
    isWorking: ref(false),
    ask: async (text, focus) => {
      asked.push([text, focus])
    },
    getSourceLocation: (turn) => locations[turn] ?? null,
    stop: () => stopped.push('stop'),
    finish: () => stopped.push('finish'),
  }
  return { conversation, said, asked, stopped }
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
  const opened: [string, readonly Span[]][] = []
  const beside: string[] = []
  const state = useAgentConversation(fake.conversation, {
    openFileAt: (path, ...spans) => opened.push([path, spans]),
    openFileBeside: (path) => beside.push(path),
    resolve: async (written) =>
      new Map(written.filter((one) => notes[one]).map((one) => [one, notes[one]!])),
    unreachable: () => '',
  })
  return { state, opened, beside, ...fake }
}

/** A setting the window lists, found at the place given. */
const listed = (namedAt: readonly string[], name: string, presence: Model['presence']): Model => ({
  namedAt,
  name,
  title: name,
  group: '',
  isDefault: false,
  writes: [],
  presence,
})

/** A window of agent tabs, with a conversation of its own for each. */
const tabs = (about = { path: '', title: '' }) => {
  const conversations: ReturnType<typeof tab>[] = []
  const held = useWindowTabs()
  const agents = createAgentKind(
    held.handle,
    () => {
      const one = tab()
      conversations.push(one)
      return one.state
    },
    () => about,
  )
  held.registerKinds([agents.kind])

  /** An agent tab of this window, and what it holds. */
  const openTab = async () => {
    const id = await held.openTabOfKind(AGENT)
    return { id, state: held.getTabStateIn<AgentTabState>(id, AGENT)! }
  }
  /** The person is in this tab now. */
  const showTab = (id: string) => held.onTabShown(id)
  const closeTab = (id: string) => held.releaseTab(id)
  /** Every agent tab on screen, the one in front last. */
  const open = () => held.tabs.value.map((one) => one.id)
  return { ...agents, openTab, showTab, closeTab, open, conversations }
}

/** A line of an answer, as the panel hands one back. */
const turn = (id: string, text: string): Turn =>
  ({ id, text, voice: 'said', state: 'done' }) as unknown as Turn

describe('a question sent', () => {
  it('names nothing the person has open, which the window reports itself', () => {
    const one = tab()

    one.state.send('what is this about')

    expect(one.asked).toEqual([['what is this about', '']])
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

  it('opens the note it names beside what the person is looking at', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.said.value = [turn('said', 'It sits under [[Thermodynamics]].')]
    await nextTick()
    const press = createPress()

    one.state.followLink(one.said.value[0]!, 'name://Thermodynamics', press.press)

    expect(one.beside).toEqual(['physics/Thermodynamics.md'])
  })

  it('opens nothing, and goes nowhere else, where no note answers to it', async () => {
    const one = tab()
    one.said.value = [turn('said', 'It sits under [[Nowhere]].')]
    await nextTick()
    const press = createPress()

    one.state.followLink(one.said.value[0]!, 'name://Nowhere', press.press)

    expect(one.beside).toEqual([])
  })
})

describe('a turn of an answer', () => {
  it('says which of the notes it names reach nothing', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.said.value = [turn('said', 'Under [[Thermodynamics]], beside [[Nowhere]].')]

    await nextTick()
    await nextTick()

    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Nowhere'])
  })
})

describe('something to ask about a note', () => {
  it('opens an agent to carry it in a window with none', async () => {
    const window = tabs()

    await window.askQuestion('Note.md — ')

    expect(window.open()).toHaveLength(1)
    expect(window.conversations[0]?.state.userQuestion.value).toBe('Note.md — ')
  })

  it('goes to the agent the person was last in, and puts it in front', async () => {
    const window = tabs()
    const first = await window.openTab()
    const second = await window.openTab()
    window.showTab(first.id)
    window.showTab(second.id)

    await window.askQuestion('Note.md — ')

    expect(window.open()).toHaveLength(2)
    expect(second.state.userQuestion.value).toBe('Note.md — ')
    expect(first.state.userQuestion.value).toBe('')
  })

  it('opens another once the one the person was last in has closed', async () => {
    const window = tabs()
    const one = await window.openTab()
    window.showTab(one.id)
    window.closeTab(one.id)

    await window.askQuestion('Note.md — ')

    expect(window.open()).toHaveLength(1)
    expect(window.conversations[1]?.state.userQuestion.value).toBe('Note.md — ')
  })
})

describe('an agent tab that closes', () => {
  it('tells the conversation it is over, so the agent lets go of what it kept', async () => {
    const window = tabs()
    const one = await window.openTab()

    window.closeTab(one.id)

    expect(window.conversations[0]?.stopped).toEqual(['finish'])
  })
})

describe('what an agent tab is called', () => {
  it('is the first thing asked of it, shortened', async () => {
    const window = tabs()
    const one = await window.openTab()
    window.conversations[0]!.said.value = [
      { id: 'a', voice: 'asked', text: 'what is this whole vault about', state: 'done' },
    ] as unknown as Turn[]

    expect(window.kind.getTitle?.(one.state)).toBe('what is this whole…')
  })

  it('is the word for an agent while nothing has been asked of it', async () => {
    const window = tabs()
    const one = await window.openTab()

    expect(window.kind.getTitle?.(one.state)).toBe('Agent')
  })
})

describe('what a tab is called by a question', () => {
  it('is the question itself when it is short enough to read', () => {
    expect(createTabTitle('what is here?')).toBe('what is here?')
  })

  it('is the first line of one written over several', () => {
    expect(createTabTitle('what is here?\nand below it?')).toBe('what is here?')
  })

  it('is the first line with anything on it', () => {
    expect(createTabTitle('\nwhat is here?')).toBe('what is here?')
  })

  it('is cut at a word rather than through one', () => {
    const asked = 'what is the note about entropy joined to'

    expect(createTabTitle(asked, 24)).toBe('what is the note about…')
  })

  it('is cut where it has to be when there is nothing to cut at', () => {
    expect(createTabTitle('x'.repeat(40), 10)).toBe(`${'x'.repeat(10)}…`)
  })

  it('is nothing when nothing has been asked', () => {
    expect(createTabTitle('')).toBe('')
  })
})

describe('what a command asked over an agent tab is over', () => {
  it('is the note the conversation is about, which is no note of the tab itself', async () => {
    const window = tabs({ path: 'physics/Ontology.md', title: 'Ontology' })
    const one = await window.openTab()

    expect(window.kind.getTarget!(one.state)).toStrictEqual({
      path: 'physics/Ontology.md',
      title: 'Ontology',
    })
  })
})

describe('agent model selection and settings', () => {
  it('offers nothing where the settings list nothing', () => {
    const one = tab()
    expect(one.state.modelOptions.value).toEqual([])
    expect(one.state.selectedAgentId.value).toBe('')
    expect(one.state.selectedModelId.value).toBe('')
  })

  it('writes the agent and model chosen', () => {
    const fake = createConversation()
    const written: [readonly string[], unknown][] = []
    const state = useAgentConversation(fake.conversation, {
      openFileAt: () => {},
      openFileBeside: () => {},
      resolve: async () => new Map(),
      unreachable: () => '',
      writeSetting: async (path, value) => {
        written.push([path, value])
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
    expect(written).toEqual([
      [['agent', 'use'], 'zeta'],
      [['agent', 'zeta', 'model'], 'zeta-small'],
    ])
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
      getModelsAt: (at) => models.value.filter((one) => one.namedAt.join('.') === at.join('.')),
    })
    expect(state.selectedAgentId.value).toBe('')

    models.value = [
      { ...listed(['agent', 'use'], 'zeta', 'present'), title: 'Zeta Agent' },
      listed(['agent', 'use'], 'omega', 'not fetched'),
      {
        ...listed(['agent', 'zeta', 'model'], 'zeta-small', 'nothing to fetch'),
        title: 'Zeta Small',
      },
      {
        ...listed(['agent', 'zeta', 'model'], 'zeta-large', 'nothing to fetch'),
        title: 'Zeta Large',
        isDefault: true,
      },
    ]
    await nextTick()

    const shown = state.modelOptions.value.map((o) => [o.agentTitle, o.modelId, o.modelTitle])
    expect(shown).toEqual([
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
    one.said.value = [turn('a', 'See [[Nowhere]].'), turn('b', 'Then [[Elsewhere]].')]
    await nextTick()
    await nextTick()
    const [first, second] = one.state.turns.value

    one.said.value = [one.said.value[0]!, one.said.value[1]!, turn('c', 'Nothing')]
    await nextTick()
    await nextTick()

    expect(one.state.turns.value[0]).toBe(first)
    expect(one.state.turns.value[1]).toBe(second)
  })

  it('draws a turn again when its unresolved addresses change', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.said.value = [turn('a', 'See [[Nowhere]].')]
    await nextTick()
    await nextTick()
    const first = one.state.turns.value[0]

    one.said.value = [turn('a', 'See [[Nowhere]] and [[Elsewhere]].')]
    await nextTick()
    await nextTick()

    expect(one.state.turns.value[0]).not.toBe(first)
    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Nowhere', 'name://Elsewhere'])
  })
})

describe('what an agent tab is called while answers stand before the question', () => {
  it('is the first thing asked, whatever was said before it', async () => {
    const window = tabs()
    const one = await window.openTab()
    window.conversations[0]!.said.value = [
      { id: 'a', voice: 'said', text: 'an answer', state: 'done' },
      { id: 'b', voice: 'asked', text: 'the question', state: 'done' },
    ] as unknown as Turn[]

    expect(window.kind.getTitle?.(one.state)).toBe('the question')
  })
})

describe('something to ask where the window holds no state for the tab', () => {
  it('still puts the tab in front', async () => {
    const shown: string[] = []
    const handle: WindowHandle = {
      openTab: async () => 'new',
      openTabBeside: async () => 'beside',
      show: (id) => void shown.push(id),
      closeTab: () => {},
      each: () => [],
      last: <TabState>() => ({ id: 'agent-1', state: {} as TabState }),
      front: () => null,
      getTabState: () => null,
    }
    const agents = createAgentKind(
      handle,
      () => tab().state,
      () => ({ path: '', title: '' }),
    )

    await agents.askQuestion('Note.md — ')

    expect(shown).toStrictEqual(['agent-1'])
  })
})

describe('the agent tab as the window keeps it', () => {
  it('is a tool the trailing pane opens with, in front of the rest', () => {
    const { kind } = tabs()

    expect(kind.isTool).toBe(true)
    expect(kind.placement).toStrictEqual({ pane: 'aside', order: 0 })
  })
})
