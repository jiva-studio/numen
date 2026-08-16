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

describe('the caret', () => {
  it('sits on an answer still arriving at the end', () => {
    expect(thread([back('1', 'half a s', 'arriving')]).find('.thread__caret').exists()).toBe(true)
  })

  it('is not drawn on a turn that settled', () => {
    expect(thread([back('1')]).find('.thread__caret').exists()).toBe(false)
  })

  it('is not drawn on an arriving turn that something follows', () => {
    const wrapper = thread([back('1', 'left behind', 'arriving'), said('2')])
    expect(wrapper.find('.thread__caret').exists()).toBe(false)
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
