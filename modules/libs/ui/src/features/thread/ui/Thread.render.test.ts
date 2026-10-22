/**
 * How much of a thread is drawn again when one turn changes.
 */
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Thread from './Thread.vue'
import type { Turn } from '../lib/turn'

const { renderSpy } = vi.hoisted(() => ({ renderSpy: vi.fn() }))

vi.mock('@/shared/ui/prose/render', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/shared/ui/prose/render')>()
  return {
    ...original,
    render: (...args: Parameters<typeof original.render>) => {
      renderSpy(args[0])
      return original.render(...args)
    },
  }
})

const createAnswered = (id: string, text: string, unresolved?: readonly string[]): Turn =>
  unresolved === undefined
    ? { id, voice: 'response', text }
    : { id, voice: 'response', text, unresolved }

const countRenders = (text: string) =>
  renderSpy.mock.calls.filter(([rendered]) => rendered === text).length

beforeEach(() => renderSpy.mockClear())

describe('a reply arriving in the last turn', () => {
  it('draws only the turn it arrives in', async () => {
    const earlier = [createAnswered('1', 'first'), createAnswered('2', 'second')]
    const wrapper = mount(Thread, {
      props: { turns: [...earlier, createAnswered('3', 'th')] },
    })
    renderSpy.mockClear()

    await wrapper.setProps({ turns: [...earlier, createAnswered('3', 'thi')] })
    await wrapper.setProps({ turns: [...earlier, createAnswered('3', 'thir')] })

    expect(countRenders('first')).toBe(0)
    expect(countRenders('second')).toBe(0)
    expect(countRenders('thi')).toBe(1)
    expect(countRenders('thir')).toBe(1)
  })

  it('draws an earlier turn that was handed over again with equal unresolved addresses', async () => {
    const wrapper = mount(Thread, {
      props: { turns: [createAnswered('1', 'first', ['name://Nowhere'])] },
    })
    renderSpy.mockClear()

    await wrapper.setProps({ turns: [createAnswered('1', 'first', ['name://Nowhere'])] })

    expect(countRenders('first')).toBe(0)
  })

  it('draws a turn again when its unresolved addresses change', async () => {
    const wrapper = mount(Thread, {
      props: { turns: [createAnswered('1', 'first', ['name://Nowhere'])] },
    })
    renderSpy.mockClear()

    await wrapper.setProps({
      turns: [createAnswered('1', 'first', ['name://Nowhere', 'name://Elsewhere'])],
    })

    expect(countRenders('first')).toBe(1)
  })
})
