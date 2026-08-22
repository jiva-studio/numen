import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Thread from './Thread.vue'
import type { Turn } from './model'

const said = (id: string, text = 'said', state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'asked', text } : { id, voice: 'asked', text, state }

const back = (id: string, text = 'back', state?: Turn['state']): Turn =>
  state === undefined
    ? { id, voice: 'answered', text }
    : { id, voice: 'answered', text, state }

const thread = (turns: readonly Turn[]) => mount(Thread, { props: { turns } })

describe('what is drawn', () => {
  it('says nothing was said yet on an empty thread', () => {
    expect(thread([]).text()).toContain('Nothing said yet')
  })

  it('draws no turns on an empty thread', () => {
    expect(thread([]).findAll('.thread__turn')).toHaveLength(0)
  })

  it('drops the empty line the moment there is a turn', () => {
    expect(thread([said('1')]).text()).not.toContain('Nothing said yet')
  })

  it('draws one element per turn', () => {
    expect(thread([said('1'), back('2'), said('3')]).findAll('.thread__turn')).toHaveLength(3)
  })

  it('keeps the turns in the order they were handed over', () => {
    const wrapper = thread([said('1', 'first'), back('2', 'second')])
    const turns = wrapper.findAll('.thread__turn')
    expect(turns[0]?.text()).toContain('first')
    expect(turns[1]?.text()).toContain('second')
  })

  it('says which voice each turn is in', () => {
    const turns = thread([said('1'), back('2')]).findAll('.thread__turn')
    expect(turns[0]?.attributes('data-voice')).toBe('asked')
    expect(turns[1]?.attributes('data-voice')).toBe('answered')
  })

  it('draws a turn with no text at all', () => {
    expect(thread([said('1', '')]).findAll('.thread__turn')).toHaveLength(1)
  })
})


describe('a turn that failed', () => {
  it('says so', () => {
    expect(thread([said('1', 'gone', 'failed')]).text()).toContain('Did not send')
  })

  it('says nothing of the sort about a turn that settled', () => {
    expect(thread([said('1')]).text()).not.toContain('Did not send')
  })
})

describe('a line about work that opens something', () => {
  const doing = (id: string, opens?: boolean): Turn =>
    opens === undefined
      ? { id, voice: 'doing', text: 'Read a document' }
      : { id, voice: 'doing', text: 'Read a document', opens }

  it('can be pressed, and says which turn was pressed', async () => {
    const wrapper = thread([doing('1', true)])

    await wrapper.find('.thread__opens').trigger('click')

    expect(wrapper.emitted('open')).toEqual([[doing('1', true)]])
  })

  it('is a line and nothing to press where the turn opens nothing', () => {
    expect(thread([doing('1')]).find('.thread__opens').exists()).toBe(false)
  })
})

describe('the places a turn speaks about', () => {
  const answered: Turn = {
    id: '2',
    voice: 'answered',
    text: 'The book says opinions differ.',
    places: [
      { id: 'a', name: 'page 39' },
      { id: 'b', name: 'page 43' },
    ],
  }

  it('are drawn under it, in the order they were given', () => {
    const wrapper = thread([answered])

    const drawn = wrapper.findAll('.thread__place')

    expect(drawn.map((one) => one.text())).toStrictEqual(['page 39', 'page 43'])
  })

  it('say which turn and which place was pressed', async () => {
    const wrapper = thread([answered])

    await wrapper.findAll('.thread__place')[1]?.trigger('click')

    expect(wrapper.emitted('go')).toEqual([[answered, { id: 'b', name: 'page 43' }]])
  })

  it('are nothing at all where a turn speaks about none', () => {
    expect(thread([back('1')]).findAll('.thread__place')).toHaveLength(0)
  })
})

describe('what the caller decides', () => {
  it('renders the body of a turn its own way when it says how', () => {
    const wrapper = mount(Thread, {
      props: { turns: [back('1', 'entropy.md')] },
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
