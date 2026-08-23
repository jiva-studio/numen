/**
 * What pressing something in an agent tab comes to, asked without a screen.
 *
 * A question carries the note the person is looking at, and a place named in
 * an answer opens the source it is in.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Turn } from '@numen/ui'
import { agentKind, talking, type Held } from './kind'
import type { Conversation } from '../conversation'
import type { Run } from '../reading'
import type { Host } from '../windowing'

/** A talk that records what it was asked, and the places its lines name. */
const talked = (places: Record<string, { path: string; start: number; length: number }> = {}) => {
  const asked: [string, string][] = []
  const stopped: string[] = []
  const talk: Conversation = {
    turns: ref<Turn[]>([]),
    working: ref(false),
    ask: async (text, focus) => {
      asked.push([text, focus])
    },
    place: (turn) => places[turn] ?? null,
    stop: () => stopped.push('stop'),
    finish: () => stopped.push('finish'),
  }
  return { talk, asked, stopped }
}

/** An agent tab with the window it is drawn in written down. */
const tab = (
  looking = 'Looking.md',
  places: Record<string, { path: string; start: number; length: number }> = {},
) => {
  const talk = talked(places)
  const opened: [string, readonly Run[]][] = []
  const held = talking(talk.talk, {
    looking: () => looking,
    opens: (path, ...runs) => opened.push([path, runs]),
    unreachable: () => '',
  })
  return { held, opened, ...talk }
}

/** A window that opens agent tabs and records which one was put in front. */
const host = (kind: () => { opens(at: string, id: string): unknown }) => {
  const opened: string[] = []
  const front: string[] = []
  let next = 0
  const given: Host = {
    opens: async () => {
      const id = `agent:${++next}`
      kind().opens('', id)
      opened.push(id)
      front.push(id)
      return id
    },
    beside: async () => '',
    shows: (id) => front.push(id),
    closes: () => {},
  }
  return { given, opened, front }
}

/** The agent tabs of a window, with a talk of its own for each. */
const tabs = () => {
  const talks: ReturnType<typeof tab>[] = []
  let made: ReturnType<typeof agentKind>
  const window = host(() => made.kind)
  made = agentKind(window.given, () => {
    const one = tab()
    talks.push(one)
    return one.held
  })
  /** A tab of this window under an identity of its own, and what it holds. */
  const holds = (id: string) => made.kind.opens('', id) as Held
  return { kind: made.kind, asks: made.asks, holds, talks, ...window }
}

/** A line of an answer, as the panel hands one back. */
const turn = (id: string, text: string): Turn =>
  ({ id, text, voice: 'said', state: 'done' }) as unknown as Turn

describe('a question sent', () => {
  it('carries the note the person is looking at', () => {
    const one = tab('Looking.md')

    one.held.send('what is this about')

    expect(one.asked).toEqual([['what is this about', 'Looking.md']])
  })

  it('empties the composer, so the question is not sent twice', () => {
    const one = tab()
    one.held.writing('half a question')

    one.held.send('half a question')

    expect(one.held.asked.value).toBe('')
  })
})

describe('a line about work pressed', () => {
  it('opens the place that call was on', () => {
    const one = tab('Looking.md', { call: { path: 'Source.pdf', start: 10, length: 4 } })

    one.held.opensTurn(turn('call', 'read Source.pdf'))

    expect(one.opened).toEqual([['Source.pdf', [{ start: 10, length: 4 }]]])
  })

  it('opens nothing for a line that names no place', () => {
    const one = tab()

    one.held.opensTurn(turn('said', 'a sentence'))

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

    one.held.followed(turn('said', text), 'numen:Source.pdf?start=10&length=4', press.press)

    expect(press.was()).toBe(true)
    expect(one.opened).toEqual([
      [
        'Source.pdf',
        [
          { start: 10, length: 4 },
          { start: 90, length: 2 },
        ],
      ],
    ])
  })

  it('is left alone when it names nowhere in the vault', () => {
    const one = tab()
    const press = pressed()

    one.held.followed(turn('said', 'read https://example.com'), 'https://example.com', press.press)

    expect(press.was()).toBe(false)
    expect(one.opened).toEqual([])
  })
})

describe('something to ask about a note', () => {
  it('opens an agent to carry it in a window with none', async () => {
    const window = tabs()

    await window.asks('Note.md — ')

    expect(window.opened).toHaveLength(1)
    expect(window.talks[0]?.held.asked.value).toBe('Note.md — ')
  })

  it('goes to the agent the person was last in, and puts it in front', async () => {
    const window = tabs()
    const first = window.holds('agent:one')
    const second = window.holds('agent:two')
    window.kind.shown?.(first, 'agent:one')
    window.kind.shown?.(second, 'agent:two')

    await window.asks('Note.md — ')

    expect(window.opened).toEqual([])
    expect(window.front.at(-1)).toBe('agent:two')
    expect(window.talks[1]?.held.asked.value).toBe('Note.md — ')
    expect(window.talks[0]?.held.asked.value).toBe('')
  })

  it('opens another once the one the person was last in has closed', async () => {
    const window = tabs()
    const one = window.holds('agent:one')
    window.kind.shown?.(one, 'agent:one')
    window.kind.shuts?.(one, 'agent:one')

    await window.asks('Note.md — ')

    expect(window.opened).toHaveLength(1)
    expect(window.talks[1]?.held.asked.value).toBe('Note.md — ')
  })
})

describe('an agent tab that closes', () => {
  it('tells the talk it is over, so the agent lets go of what it kept', () => {
    const window = tabs()
    const one = window.holds('agent:one')

    window.kind.shuts?.(one, 'agent:one')

    expect(window.talks[0]?.stopped).toEqual(['finish'])
  })
})

describe('what an agent tab is called', () => {
  it('is the first thing asked of it, shortened', () => {
    const window = tabs()
    const one = window.holds('agent:one')
    one.turns.value = [
      { id: 'a', voice: 'asked', text: 'what is this whole vault about', state: 'done' },
    ] as unknown as Turn[]

    expect(window.kind.called(one)).toBe('what is this whole…')
  })

  it('is the word for an agent while nothing has been asked of it', () => {
    const window = tabs()
    const one = window.holds('agent:one')

    expect(window.kind.called(one)).toBe('Agent')
  })
})
