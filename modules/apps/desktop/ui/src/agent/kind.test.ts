/**
 * What pressing something in an agent tab comes to, asked without a screen.
 *
 * A question carries the note the person is looking at, and a place named in
 * an answer opens the source it is in.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Turn } from '@numen/ui'
import { talking } from './kind'
import type { Conversation } from '../conversation'
import type { Run } from '../reading'

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
  })
  return { held, opened, ...talk }
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
