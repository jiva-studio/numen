import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Agent from './Agent.vue'
import type { Turn } from '@/features/thread'

const said = (id: string, text = 'said'): Turn => ({ id, voice: 'asked', text })
const back = (id: string, text = 'back'): Turn => ({ id, voice: 'answered', text })

/** How tall the area the conversation scrolls in is, and one turn in it. */
const SCREEN = 100

/**
 * An agent over a conversation that scrolls, since the document a test runs in
 * lays nothing out. One turn is one screenful.
 */
const createAgent = (turns: readonly Turn[]) => {
  const wrapper = mount(Agent, { props: { turns, modelValue: '' } })
  const thread = wrapper.find('.agent__thread').element as HTMLElement
  let top = 0
  Object.defineProperty(thread, 'clientHeight', { get: () => SCREEN })
  Object.defineProperty(thread, 'scrollHeight', {
    get: () => (wrapper.props('turns') as readonly Turn[]).length * SCREEN,
  })
  Object.defineProperty(thread, 'scrollTop', {
    get: () => top,
    set: (to: number) => {
      top = to
    },
  })

  return {
    wrapper,
    at: () => top,
    reads: async (from: number) => {
      top = from
      await wrapper.find('.agent__thread').trigger('scroll')
    },
    /** Every text the field has handed on, in the order it handed them on. */
    handed: (): readonly unknown[] =>
      (wrapper.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0]),
    /**
     * Written in the field and sent, the caller putting back what it is handed
     * before the key is pressed, which is what holding the field is.
     */
    sends: async (text: string) => {
      const field = wrapper.find('textarea')
      await field.setValue(text)
      await wrapper.setProps({ modelValue: text })
      await field.trigger('keydown', { key: 'Enter' })
    },
  }
}

describe('a question sent', () => {
  it('is carried to whoever answers it', async () => {
    const one = createAgent([said('1'), back('2')])

    await one.sends('what is a seat')

    expect(one.handed()).toEqual(['what is a seat'])
    expect(one.wrapper.emitted('submit')).toEqual([['what is a seat']])
  })

  it('brings the foot of the conversation into view', async () => {
    const one = createAgent([said('1'), back('2'), said('3')])
    await one.reads(0)

    await one.sends('and one more')
    await one.wrapper.setProps({ turns: [said('1'), back('2'), said('3'), said('4')] })

    expect(one.at()).toBe(3 * SCREEN)
  })
})

describe('a conversation read further up', () => {
  it('stays where it is read while an answer arrives', async () => {
    const one = createAgent([said('1'), back('2'), said('3')])
    await one.reads(0)

    await one.wrapper.setProps({ turns: [said('1'), back('2'), said('3'), back('4')] })

    expect(one.at()).toBe(0)
  })
})
