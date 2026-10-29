/**
 * How much of a thread is drawn again when one turn changes.
 */
import { mount } from '@vue/test-utils'
import type { Component } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Thread from './Thread.vue'
import type { Turn } from '../lib/turn'

const { renderSpy } = vi.hoisted(() => ({ renderSpy: vi.fn() }))

// Prose of @numen/ui, counting each time it is drawn with the text it is drawn with.
vi.mock('@numen/ui', async (importOriginal) => {
  const original = await importOriginal<typeof import('@numen/ui')>()
  const { defineComponent, h } = await import('vue')
  const Prose = defineComponent({
    inheritAttrs: false,
    setup:
      (_, { attrs }) =>
      () => {
        renderSpy(attrs['text'])
        return h(original.Prose as Component, attrs)
      },
  })
  return { ...original, Prose }
})

const createResponded = (id: string, text: string, unresolved?: readonly string[]): Turn =>
  unresolved === undefined
    ? { id, voice: 'response', text }
    : { id, voice: 'response', text, unresolved }

const countRenders = (text: string) =>
  renderSpy.mock.calls.filter(([rendered]) => rendered === text).length

beforeEach(() => renderSpy.mockClear())

describe('a reply arriving in the last turn', () => {
  it('draws only the turn it arrives in', async () => {
    const earlier = [createResponded('1', 'first'), createResponded('2', 'second')]
    const wrapper = mount(Thread, {
      props: { turns: [...earlier, createResponded('3', 'th')] },
    })
    renderSpy.mockClear()

    await wrapper.setProps({ turns: [...earlier, createResponded('3', 'thi')] })
    await wrapper.setProps({ turns: [...earlier, createResponded('3', 'thir')] })

    expect(countRenders('first')).toBe(0)
    expect(countRenders('second')).toBe(0)
    expect(countRenders('thi')).toBe(1)
    expect(countRenders('thir')).toBe(1)
  })

  it('draws a turn again when its unresolved addresses change', async () => {
    const wrapper = mount(Thread, {
      props: { turns: [createResponded('1', 'first', ['name://Nowhere'])] },
    })
    renderSpy.mockClear()

    await wrapper.setProps({
      turns: [createResponded('1', 'first', ['name://Nowhere', 'name://Elsewhere'])],
    })

    expect(countRenders('first')).toBe(1)
  })
})

describe('a reply naming notes', () => {
  it('draws the links of the turn as links, and reports a press on one', async () => {
    const turn: Turn = {
      id: '1',
      voice: 'response',
      text: 'It sits under [[Thermodynamics]].',
      links: [{ raw: '[[Thermodynamics]]', href: 'name://Thermodynamics', text: 'Thermo' }],
    }
    const wrapper = mount(Thread, { props: { turns: [turn] } })

    const link = wrapper.find('a')
    await link.trigger('click')

    expect(link.text()).toBe('Thermo')
    expect(wrapper.emitted('follow')?.[0]?.slice(0, 2)).toEqual([turn, 'name://Thermodynamics'])
  })
})
