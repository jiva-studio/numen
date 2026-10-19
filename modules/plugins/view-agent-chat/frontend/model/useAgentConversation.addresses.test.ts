/**
 * The notes an answer names: when each is looked up, and which turns are drawn
 * again as the lookups come back.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import * as ui from '@numen/ui'
import type { Conversation, Turn } from '@numen/ui'
import { useAgentConversation } from './useAgentConversation'
import type { Span } from '@numen/plugin-sdk'

vi.mock('@numen/ui', async (importOriginal) => {
  const original = await importOriginal<typeof import('@numen/ui')>()
  return { ...original, getWikilinks: vi.fn(original.getWikilinks) }
})

/** A line of an answer. */
const turn = (id: string, text: string): Turn =>
  ({ id, text, voice: 'said', state: 'done' }) as unknown as Turn

/** An agent tab over turns already said, and a vault that answers for the notes given. */
const open = (said: Turn[] = [], notes: Record<string, string> = {}) => {
  const turns = ref<Turn[]>(said)
  const conversation: Conversation = {
    turns,
    isWorking: ref(false),
    ask: async () => {},
    getSourceLocation: () => null,
    stop: () => {},
    finish: () => {},
  }
  const resolved: (readonly string[])[] = []
  const opened: [string, readonly Span[]][] = []
  const beside: string[] = []
  const state = useAgentConversation(conversation, {
    openFileAt: (path, ...spans) => opened.push([path, spans]),
    openFileBeside: (path) => beside.push(path),
    resolve: async (written) => {
      resolved.push(written)
      return new Map(written.filter((one) => notes[one]).map((one) => [one, notes[one]!]))
    },
    unreachable: () => '',
  })
  return { state, turns, resolved, opened, beside }
}

/** Lets the lookups come back and the turns be drawn. */
const settle = async () => {
  for (let step = 0; step < 4; step++) await nextTick()
}

/** The text of a turn changed where it stands, as it grows while being said. */
const rewrite = (one: Turn, text: string) => {
  ;(one as { text: string }).text = text
}

const createPress = () => {
  let prevented = false
  const press = { preventDefault: () => (prevented = true) } as unknown as MouseEvent
  return { press, was: () => prevented }
}

describe('the notes an answer names', () => {
  it('are looked up as soon as the tab is made over turns already said', () => {
    const one = open([turn('a', 'See [[Entropy]].')])

    expect(one.resolved).toEqual([['name://Entropy']])
  })

  it('are looked up once each, however many turns name them', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])
    one.turns.value = [...one.turns.value, turn('b', 'And [[Entropy]] with [[Order]].')]
    await settle()

    expect(one.resolved).toEqual([['name://Entropy'], ['name://Order']])
  })

  it('are looked up again when the last turn grows while it is being said', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])
    await settle()

    rewrite(one.turns.value[0]!, 'See [[Entropy]] and [[Order]].')
    await settle()

    expect(one.resolved).toEqual([['name://Entropy'], ['name://Order']])
  })

  it('are read out of a turn once while its text stays the same', async () => {
    vi.mocked(ui.getWikilinks).mockClear()
    const one = open([turn('a', 'See [[Entropy]].')])
    await settle()
    void one.state.turns.value

    expect(vi.mocked(ui.getWikilinks)).toHaveBeenCalledTimes(1)
  })

  it('are read out of a turn again when its text changes', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])
    await settle()

    rewrite(one.turns.value[0]!, 'See [[Order]].')
    await settle()

    expect(one.resolved.at(-1)).toEqual(['name://Order'])
  })
})

describe('a turn drawn with the notes that reach nothing', () => {
  it('is the turn itself where every note it names was found', async () => {
    const one = open([turn('a', 'See [[Entropy]].')], { 'name://Entropy': 'Entropy.md' })
    await settle()

    expect(one.state.turns.value[0]).toBe(one.turns.value[0])
    expect(one.state.turns.value[0]).not.toHaveProperty('unresolved')
  })

  it('names the notes again once they change while the turn stays', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])
    await settle()
    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Entropy'])

    rewrite(one.turns.value[0]!, 'See [[Entropy]] and [[Order]].')
    await settle()

    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Entropy', 'name://Order'])
  })

  it('names the notes again when as many are named but not the same', async () => {
    const one = open([turn('a', 'See [[Entropy]] and [[Order]].')])
    await settle()
    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Entropy', 'name://Order'])

    rewrite(one.turns.value[0]!, 'See [[Entropy]] and [[Chaos]].')
    await settle()

    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Entropy', 'name://Chaos'])
  })
})

describe('a link inside an answer', () => {
  it('lights only the places of the file it names', () => {
    const one = open()
    const text =
      '[a](numen:Source.pdf?start=10&length=4) [b](numen:Other.pdf?start=5&length=2) [c](numen:Source.pdf?start=90&length=2)'

    one.state.followLink(turn('a', text), 'numen:Source.pdf?start=10&length=4', createPress().press)

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

  it('opens no note for an address that is not a note', async () => {
    const one = open([turn('a', 'See [[web://page]].')], { 'web://page': 'Page.md' })
    await settle()
    const press = createPress()

    one.state.followLink(one.turns.value[0]!, 'web://page', press.press)

    expect(one.beside).toEqual([])
    expect(press.was()).toBe(false)
  })
})
