/**
 * What pressing something in an agent tab comes to, asked without a screen.
 *
 * A question carries nothing about what is open, and a passage named in an
 * answer opens the source it is in.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import type { Conversation, Turn } from '@numen/ui'
import { agentKind, talking, type AgentTabState } from './kind'
import type { Span } from '../shared/core'
import { windowing } from '../shared/tabs/windowing'
import { AGENT } from '../shared/tabs/workspace'

/** A talk that records what it was asked, and the places its lines name. */
const talked = (places: Record<string, { path: string; span: Span }> = {}) => {
  const asked: [string, string][] = []
  const stopped: string[] = []
  const said = ref<Turn[]>([])
  const talk: Conversation = {
    turns: said,
    working: ref(false),
    ask: async (text, focus) => {
      asked.push([text, focus])
    },
    place: (turn) => places[turn] ?? null,
    stop: () => stopped.push('stop'),
    finish: () => stopped.push('finish'),
  }
  return { talk, said, asked, stopped }
}

/**
 * An agent tab with the window it is drawn in written down. The vault answers
 * with the notes it was handed, and reaches nothing for every other address.
 */
const tab = (
  places: Record<string, { path: string; span: Span }> = {},
  notes: Record<string, string> = {},
) => {
  const talk = talked(places)
  const opened: [string, readonly Span[]][] = []
  const beside: string[] = []
  const state = talking(talk.talk, {
    opens: (path, ...spans) => opened.push([path, spans]),
    beside: (path) => beside.push(path),
    resolve: async (written) =>
      new Map(written.filter((one) => notes[one]).map((one) => [one, notes[one]!])),
    unreachable: () => '',
  })
  return { state, opened, beside, ...talk }
}

/** A window of agent tabs, with a talk of its own for each. */
const tabs = (about = { path: '', title: '' }) => {
  const talks: ReturnType<typeof tab>[] = []
  const held = windowing()
  const agents = agentKind(
    held.handle,
    () => {
      const one = tab()
      talks.push(one)
      return one.state
    },
    () => about,
  )
  held.declares([agents.kind])

  /** An agent tab of this window, and what it holds. */
  const holds = async () => {
    const id = await held.opens(AGENT)
    return { id, state: held.holdsIn<AgentTabState>(id, AGENT)! }
  }
  /** The person is in this tab now. */
  const enters = (id: string) => held.shown(id)
  const shuts = (id: string) => held.shut(id)
  /** Every agent tab on screen, the one in front last. */
  const open = () => held.tabs.value.map((one) => one.id)
  return { ...agents, holds, enters, shuts, open, talks }
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
    one.state.writing('half a question')

    one.state.send('half a question')

    expect(one.state.asked.value).toBe('')
  })
})

describe('a line about work pressed', () => {
  it('opens the place that call was on', () => {
    const one = tab({ call: { path: 'Source.pdf', span: { from: 10, to: 14 } } })

    one.state.opensTurn(turn('call', 'read Source.pdf'))

    expect(one.opened).toEqual([['Source.pdf', [{ from: 10, to: 14 }]]])
  })

  it('opens nothing for a line that names no place', () => {
    const one = tab()

    one.state.opensTurn(turn('said', 'a sentence'))

    expect(one.opened).toEqual([])
  })
})

describe('a link inside an answer', () => {
  const pressed = () => {
    let prevented = false
    const press = { preventDefault: () => (prevented = true) } as unknown as MouseEvent
    return { press, was: () => prevented }
  }

  it('opens the place it names, with the other places that answer names in it', () => {
    const one = tab()
    const text =
      'see [here](numen:Source.pdf?start=10&length=4) and [there](numen:Source.pdf?start=90&length=2)'
    const press = pressed()

    one.state.followed(turn('said', text), 'numen:Source.pdf?start=10&length=4', press.press)

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
    const press = pressed()

    one.state.followed(turn('said', 'read https://example.com'), 'https://example.com', press.press)

    expect(press.was()).toBe(false)
    expect(one.opened).toEqual([])
  })

  it('opens the note it names beside what the person is looking at', async () => {
    const one = tab({}, { 'name://Thermodynamics': 'physics/Thermodynamics.md' })
    one.said.value = [turn('said', 'It sits under [[Thermodynamics]].')]
    await nextTick()
    const press = pressed()

    one.state.followed(one.said.value[0]!, 'name://Thermodynamics', press.press)

    expect(one.beside).toEqual(['physics/Thermodynamics.md'])
  })

  it('opens nothing, and goes nowhere else, where no note answers to it', async () => {
    const one = tab()
    one.said.value = [turn('said', 'It sits under [[Nowhere]].')]
    await nextTick()
    const press = pressed()

    one.state.followed(one.said.value[0]!, 'name://Nowhere', press.press)

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

    await window.asks('Note.md — ')

    expect(window.open()).toHaveLength(1)
    expect(window.talks[0]?.state.asked.value).toBe('Note.md — ')
  })

  it('goes to the agent the person was last in, and puts it in front', async () => {
    const window = tabs()
    const first = await window.holds()
    const second = await window.holds()
    window.enters(first.id)
    window.enters(second.id)

    await window.asks('Note.md — ')

    expect(window.open()).toHaveLength(2)
    expect(second.state.asked.value).toBe('Note.md — ')
    expect(first.state.asked.value).toBe('')
  })

  it('opens another once the one the person was last in has closed', async () => {
    const window = tabs()
    const one = await window.holds()
    window.enters(one.id)
    window.shuts(one.id)

    await window.asks('Note.md — ')

    expect(window.open()).toHaveLength(1)
    expect(window.talks[1]?.state.asked.value).toBe('Note.md — ')
  })
})

describe('an agent tab that closes', () => {
  it('tells the talk it is over, so the agent lets go of what it kept', async () => {
    const window = tabs()
    const one = await window.holds()

    window.shuts(one.id)

    expect(window.talks[0]?.stopped).toEqual(['finish'])
  })
})

describe('what an agent tab is called', () => {
  it('is the first thing asked of it, shortened', async () => {
    const window = tabs()
    const one = await window.holds()
    window.talks[0]!.said.value = [
      { id: 'a', voice: 'asked', text: 'what is this whole vault about', state: 'done' },
    ] as unknown as Turn[]

    expect(window.kind.called(one.state)).toBe('what is this whole…')
  })

  it('is the word for an agent while nothing has been asked of it', async () => {
    const window = tabs()
    const one = await window.holds()

    expect(window.kind.called(one.state)).toBe('Agent')
  })
})

describe('what a command asked over an agent tab is over', () => {
  it('is the note the talk is about, which is no note of the tab itself', async () => {
    const window = tabs({ path: 'physics/Ontology.md', title: 'Ontology' })
    const one = await window.holds()

    expect(window.kind.at!(one.state)).toStrictEqual({
      path: 'physics/Ontology.md',
      title: 'Ontology',
    })
  })
})
