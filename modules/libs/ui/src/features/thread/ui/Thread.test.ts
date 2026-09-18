import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Thread from './Thread.vue'
import type { Turn } from '../lib/turn'

const createAsked = (id: string, text = 'said', state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'asked', text } : { id, voice: 'asked', text, state }

const createAnswered = (id: string, text = 'back', state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'answered', text } : { id, voice: 'answered', text, state }

const thread = (turns: readonly Turn[]) => mount(Thread, { props: { turns } })

describe('what is drawn', () => {
  it('says nothing was said yet on an empty thread', () => {
    expect(thread([]).text()).toContain('Nothing said yet')
  })

  it('draws no turns on an empty thread', () => {
    expect(thread([]).findAll('.thread__turn')).toHaveLength(0)
  })

  it('drops the empty line the moment there is a turn', () => {
    expect(thread([createAsked('1')]).text()).not.toContain('Nothing said yet')
  })

  it('draws one element per turn', () => {
    expect(
      thread([createAsked('1'), createAnswered('2'), createAsked('3')]).findAll('.thread__turn'),
    ).toHaveLength(3)
  })

  it('keeps the turns in the order they were handed over', () => {
    const wrapper = thread([createAsked('1', 'first'), createAnswered('2', 'second')])
    const turns = wrapper.findAll('.thread__turn')
    expect(turns[0]?.text()).toContain('first')
    expect(turns[1]?.text()).toContain('second')
  })

  it('says which voice each turn is in', () => {
    const turns = thread([createAsked('1'), createAnswered('2')]).findAll('.thread__turn')
    expect(turns[0]?.attributes('data-voice')).toBe('asked')
    expect(turns[1]?.attributes('data-voice')).toBe('answered')
  })

  it('draws a turn with no text at all', () => {
    expect(thread([createAsked('1', '')]).findAll('.thread__turn')).toHaveLength(1)
  })
})

describe('a turn that failed', () => {
  it('says so', () => {
    expect(thread([createAsked('1', 'gone', 'failed')]).text()).toContain('Did not send')
  })

  it('says nothing of the sort about a turn that settled', () => {
    expect(thread([createAsked('1')]).text()).not.toContain('Did not send')
  })
})

describe('a line about work that opens something', () => {
  const createDoing = (id: string, opens?: boolean): Turn =>
    opens === undefined
      ? { id, voice: 'doing', text: 'Read a document' }
      : { id, voice: 'doing', text: 'Read a document', isOpening: opens }

  it('can be pressed, and says which turn was pressed', async () => {
    const wrapper = thread([createDoing('1', true)])

    await wrapper.find('.thread__opens').trigger('click')

    expect(wrapper.emitted('open')).toEqual([[createDoing('1', true)]])
  })

  it('is a line and nothing to press where the turn opens nothing', () => {
    expect(
      thread([createDoing('1')])
        .find('.thread__opens')
        .exists(),
    ).toBe(false)
  })
})

describe('what the caller decides', () => {
  it('renders the body of a turn its own way when it says how', () => {
    const wrapper = mount(Thread, {
      props: { turns: [createAnswered('1', 'entropy.md')] },
      slots: { turn: '<code class="own">given</code>' },
    })
    expect(wrapper.find('code.own').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('entropy.md')
  })

  it('says its own thing about an empty thread', () => {
    const wrapper = mount(Thread, {
      props: { turns: [] },
      slots: { silence: 'ask it something' },
    })
    expect(wrapper.text()).toContain('ask it something')
    expect(wrapper.text()).not.toContain('Nothing said yet')
  })
})

/** How tall the area that scrolls is, and how tall one turn in it is. */
const SCREEN = 100

/**
 * A thread over an area that scrolls, since the document a test runs in lays
 * nothing out. One turn is one screenful.
 */
const createScrollingThread = (turns: readonly Turn[]) => {
  const wrapper = thread(turns)
  const area = wrapper.element as HTMLElement
  let top = 0
  Object.defineProperty(area, 'clientHeight', { get: () => SCREEN })
  Object.defineProperty(area, 'scrollHeight', {
    get: () => wrapper.props('turns').length * SCREEN,
  })
  Object.defineProperty(area, 'scrollTop', {
    get: () => top,
    set: (to: number) => {
      top = to
    },
  })

  return {
    wrapper,
    /** How far down it stands. */
    at: () => top,
    /** Read at a place of the person's own choosing. */
    reads: async (from: number) => {
      top = from
      await wrapper.trigger('scroll')
    },
    /** One more turn, as an answer would arrive. */
    arrives: (turn: Turn) => wrapper.setProps({ turns: [...wrapper.props('turns'), turn] }),
  }
}

describe('following the foot', () => {
  it('brings a turn that arrives into view', async () => {
    const one = createScrollingThread([createAsked('1'), createAnswered('2')])

    await one.arrives(createAsked('3'))

    expect(one.at()).toBe(2 * SCREEN)
  })

  it('follows an answer as it is written', async () => {
    const one = createScrollingThread([createAsked('1'), createAnswered('2', '')])

    await one.wrapper.setProps({ turns: [createAsked('1'), createAnswered('2', 'a first word')] })

    expect(one.at()).toBe(SCREEN)
  })

  it('leaves a reader who scrolled up where they are reading', async () => {
    const one = createScrollingThread([createAsked('1'), createAnswered('2'), createAsked('3')])
    await one.reads(0)

    await one.arrives(createAnswered('4'))

    expect(one.at()).toBe(0)
  })

  it('takes the foot up again once it is read back down to', async () => {
    const one = createScrollingThread([createAsked('1'), createAnswered('2'), createAsked('3')])
    await one.reads(0)
    await one.reads(2 * SCREEN)

    await one.arrives(createAnswered('4'))

    expect(one.at()).toBe(3 * SCREEN)
  })

  it('is asked back to the foot, wherever it was left', async () => {
    const one = createScrollingThread([createAsked('1'), createAnswered('2'), createAsked('3')])
    await one.reads(0)

    ;(one.wrapper.vm as unknown as { toFoot: (again?: boolean) => void }).toFoot(true)

    expect(one.at()).toBe(2 * SCREEN)
  })
})

/**
 * The step each part of a turn is set at.
 *
 * What was said and the line about work are set in the interface's own text,
 * and an answer is marked-up text a shade above it. The quiet step is for what
 * is said about a turn beside it.
 */
describe('the step each part is set at', () => {
  const createDoing = (id: string): Turn => ({ id, voice: 'doing', text: 'Thinking' })

  it("sets the thread in the interface's own text", () => {
    expect(thread([]).classes()).toContain('text-base')
  })

  it('leaves what was said at the step the thread is set in', () => {
    expect(
      thread([createAsked('1')])
        .find('.thread__body')
        .classes(),
    ).not.toContain('text-small')
  })

  it('sets a line about work at that step as well', () => {
    expect(
      thread([createDoing('1')])
        .find('.tool-call')
        .classes(),
    ).toContain('text-base')
  })

  it('says a turn did not send in the quiet step', () => {
    expect(
      thread([createAsked('1', 'gone', 'failed')])
        .find('.thread__failure')
        .classes(),
    ).toContain('text-small')
  })
})
