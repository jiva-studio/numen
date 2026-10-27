/**
 * The notes an answer names: when the core is asked about a turn's wikilinks, and which turns are
 * drawn again as the answers come back.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import type { Conversation, Turn } from '../thread'
import { useAgentConversation } from './useAgentConversation'
import type { TextRange } from '@numen/plugin-sdk'
import { createLinkResolver } from '../fixtures/links'

/** A line of an answer. */
const turn = (id: string, text: string): Turn => ({ id, text, voice: 'response', state: 'settled' })

/** An agent tab over turns already said, and a vault that answers for the notes given. */
const open = (turn: Turn[] = [], notes: Record<string, string> = {}) => {
  const turns = ref<Turn[]>(turn)
  const conversation: Conversation = {
    turns,
    isWorking: ref(false),
    ask: async () => {},
    getSourceLocation: () => null,
    stop: () => {},
    finish: () => {},
  }
  const requests: string[] = []
  const openedCalls: [string, readonly TextRange[]][] = []
  const besideItems: string[] = []
  const resolveLinks = createLinkResolver(notes)
  const state = useAgentConversation(conversation, {
    openFileAt: (path, ...spans) => openedCalls.push([path, spans]),
    openFileBeside: (path) => besideItems.push(path),
    resolveLinks: (text) => {
      requests.push(text)
      return resolveLinks(text)
    },
    unreachable: () => '',
  })
  return { state, turns, requests, opened: openedCalls, besideItems }
}

/** Lets the answers come back and the turns be drawn. */
const settle = async () => {
  for (let step = 0; step < 6; step++) await nextTick()
}

/** The text of a turn changed at its current position, as it grows while active. */
const rewrite = (one: Turn, text: string) => {
  ;(one as { text: string }).text = text
}

const createPress = () => {
  let prevented = false
  const press = { preventDefault: () => (prevented = true) } as unknown as MouseEvent
  return { press, was: () => prevented }
}

describe('the notes an answer names', () => {
  it('requests the links of existing turns on creation', () => {
    const one = open([turn('a', 'See [[Entropy]].')])

    expect(one.requests).toEqual(['See [[Entropy]].'])
  })

  it('requests the links once per turn text', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])
    one.turns.value = [...one.turns.value, turn('b', 'And [[Entropy]] with [[Order]].')]
    await settle()
    void one.state.turns.value

    expect(one.requests).toEqual(['See [[Entropy]].', 'And [[Entropy]] with [[Order]].'])
  })

  it('skips the request for a text without brackets', async () => {
    const one = open([turn('a', 'Nothing named here.')])
    await settle()

    expect(one.requests).toEqual([])
  })

  it('requests the links again when the last turn grows', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])
    await settle()

    rewrite(one.turns.value[0]!, 'See [[Entropy]] and [[Order]].')
    await settle()

    expect(one.requests).toEqual(['See [[Entropy]].', 'See [[Entropy]] and [[Order]].'])
  })

  it('requests the latest text after the pending request completes', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])

    rewrite(one.turns.value[0]!, 'See [[Order]].')
    one.turns.value = [...one.turns.value]
    await settle()

    expect(one.requests).toEqual(['See [[Entropy]].', 'See [[Order]].'])
    expect(one.state.turns.value[0]?.unresolved).toEqual(['name://Order'])
  })
})

describe('a turn drawn with the notes that reach nothing', () => {
  it('is the turn itself where it names no note', async () => {
    const one = open([turn('a', 'See nothing.')])
    await settle()

    expect(one.state.turns.value[0]).toBe(one.turns.value[0])
  })

  it('carries the links the core read, and no address where every note was found', async () => {
    const one = open([turn('a', 'See [[Entropy|heat]].')], { 'name://Entropy': 'Entropy.md' })
    await settle()

    expect(one.state.turns.value[0]?.links).toEqual([
      { raw: '[[Entropy|heat]]', href: 'name://Entropy', text: 'heat' },
    ])
    expect(one.state.turns.value[0]?.unresolved).toEqual([])
  })

  it('is the same turn while nothing about its links changed', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])
    await settle()

    const before = one.state.turns.value[0]

    one.turns.value = [...one.turns.value, turn('b', 'Plain.')]
    await settle()

    expect(one.state.turns.value[0]).toBe(before)
  })

  it('keeps the previous links while the longer text resolves', async () => {
    const one = open([turn('a', 'See [[Entropy]].')])
    await settle()

    one.turns.value = [turn('a', 'See [[Entropy]] and more')]

    expect(one.state.turns.value[0]?.links?.map((link) => link.raw)).toEqual(['[[Entropy]]'])
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
          { start: 10, end: 14 },
          { start: 90, end: 92 },
        ],
      ],
    ])
  })

  it('opens the target note of a wikilink in the adjacent pane', async () => {
    const one = open([turn('a', 'See [[Entropy]].')], { 'name://Entropy': 'Entropy.md' })
    await settle()

    one.state.followLink(one.turns.value[0]!, 'name://Entropy', createPress().press)

    expect(one.besideItems).toEqual(['Entropy.md'])
  })

  it('opens no note for an address the core reached no note for', async () => {
    const one = open([turn('a', 'See [[Nowhere]].')])
    await settle()
    const press = createPress()

    one.state.followLink(one.turns.value[0]!, 'name://Nowhere', press.press)

    expect(one.besideItems).toEqual([])
    expect(press.was()).toBe(false)
  })
})
