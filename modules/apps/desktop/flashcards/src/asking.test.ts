import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { AgentPort, AgentStep } from '@numen/ui'

import { asking } from './asking'
import type { Asked } from './core'

/** A card as the sitting hands one over. */
const card = (more: Partial<Asked> = {}): Asked => ({
  deck: 'decks/Words.md',
  section: '',
  card: '3f4g5h6j7k',
  face: 'Say it',
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  seen: true,
  ahead: null,
  ...more,
})

/** An agent that keeps what it was asked and says what a test told it to. */
const answers = (says: AgentStep[] = [{ kind: 'said', text: 'Because of the leaves.' }]) => {
  const asked: { text: string; focus: string; conversation: string }[] = []
  const over: string[] = []
  const agent: AgentPort = {
    async *ask(text, focus, conversation) {
      asked.push({ text, focus, conversation })
      for (const step of says) yield step
    },
    async finish(conversation) {
      over.push(conversation)
    },
  }
  return { agent, asked, over }
}

/** The panel, with the sitting around it standing in for the window. */
const panel = (more: { card?: Asked | null; unreachable?: string } = {}) => {
  const { agent, asked, over } = answers()
  const on = ref<Asked | null>(more.card === undefined ? card() : more.card)
  const said: string[] = []
  const held = asking({
    agent,
    card: () => on.value,
    unreachable: () => more.unreachable ?? '',
    says: (one) => said.push(one),
    // The words are put up as they arrive, so a test reads them without waiting
    // for a frame.
    paint: (draw) => draw(),
  })
  return { held, asked, over, said, on }
}

describe('the panel coming in', () => {
  it('does not come in between cards', () => {
    const { held } = panel({ card: null })
    held.opens()
    expect(held.open.value).toBe(false)
  })

  it('comes in on the card the sitting is on, turned or not', () => {
    const { held } = panel()
    held.opens()
    expect(held.open.value).toBe(true)
    expect(held.about.value?.card).toBe('3f4g5h6j7k')
  })

  // A gesture that does nothing is a gesture a person repeats.
  it('says why it cannot come in, and does not', () => {
    const { held, said } = panel({ unreachable: 'The agent could not be reached.' })
    held.opens()
    expect(held.open.value).toBe(false)
    expect(said).toEqual(['The agent could not be reached.'])
  })

  it('is put away and keeps what was said', async () => {
    const { held } = panel()
    held.opens()
    await held.send('why')
    held.shuts()
    expect(held.open.value).toBe(false)
    expect(held.turns.value.length).toBeGreaterThan(0)
  })
})

describe('one conversation to a card', () => {
  it('carries the card into the first question and not into the rest', async () => {
    const { held, asked } = panel()
    held.opens()

    await held.send('why is it called that')
    await held.send('and where does it grow')

    expect(asked[0]?.text).toContain('3f4g5h6j7k')
    expect(asked[0]?.text).toContain('decks/Words.md')
    expect(asked[0]?.text).toContain('why is it called that')
    expect(asked[1]?.text).toBe('and where does it grow')
  })

  it('asks about the deck the card stands in', async () => {
    const { held, asked } = panel()
    held.opens()
    await held.send('why')
    expect(asked[0]?.focus).toBe('decks/Words.md')
  })

  it('answers every question of one card in one conversation', async () => {
    const { held, asked } = panel()
    held.opens()
    await held.send('one')
    await held.send('two')
    expect(asked[0]?.conversation).toBe(asked[1]?.conversation)
  })

  // The card in front of a person is never answered out of the one behind it.
  it('ends the conversation when the card is answered, and tells the agent', async () => {
    const { held, over } = panel()
    held.opens()
    await held.send('why')

    held.ends()
    expect(held.open.value).toBe(false)
    expect(held.about.value).toBeNull()
    expect(held.turns.value).toEqual([])
    expect(over.length).toBe(1)
  })

  it('opens a conversation of its own for the next card', async () => {
    const { held, asked, on } = panel()
    held.opens()
    await held.send('one')
    const first = asked[0]?.conversation

    held.ends()
    on.value = card({ card: 'zpqrstvwxy' })
    held.opens()
    await held.send('two')

    expect(asked[1]?.conversation).not.toBe(first)
    expect(asked[1]?.text).toContain('zpqrstvwxy')
  })

  // A name stands for one conversation and is never given to a second.
  it('never gives one name to two conversations', async () => {
    const { held, asked, on } = panel()
    for (const mark of ['a', 'b', 'c']) {
      on.value = card({ card: mark })
      held.opens()
      await held.send('why')
      held.ends()
    }
    const named = asked.map((one) => one.conversation)
    expect(new Set(named).size).toBe(named.length)
  })

  it('sends nothing while no card is being asked about', async () => {
    const { held, asked } = panel()
    await held.send('why')
    expect(asked).toEqual([])
  })

  it('sends nothing when nothing was written', async () => {
    const { held, asked } = panel()
    held.opens()
    await held.send('')
    expect(asked).toEqual([])
  })
})
