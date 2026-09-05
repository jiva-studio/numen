/**
 * A box as tall as what it holds: what the ground behind it carries, what a
 * caller's attributes land on, and what is said when something is typed.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AutosizeTextarea from './AutosizeTextarea.vue'

const mountBox = (props: Record<string, unknown> = {}) =>
  mount(AutosizeTextarea, { props: { text: 'said', ...props } })

const box = (one: ReturnType<typeof mountBox>) => one.get<HTMLTextAreaElement>('.autosize__box')

describe('the box', () => {
  it('holds the text it is given', () => {
    expect(box(mountBox({ text: 'two words' })).element.value).toBe('two words')
  })

  // The cell is sized by the ground, so the ground has to carry the same text
  // the box does or the box is the wrong height for it.
  it('is sized by a ground carrying the same text', () => {
    const one = mountBox({ text: 'a line\nand another' })
    expect(one.get('.autosize').attributes('data-autosize')).toBe('a line\nand another')
  })

  it('is one line and never scrolls itself', () => {
    expect(box(mountBox()).attributes('rows')).toBe('1')
  })

  it('says the text as it now reads when something is typed into it', async () => {
    const one = mountBox()
    await box(one).setValue('said more')
    expect(one.emitted('write')).toEqual([['said more']])
  })
})

describe('what a caller hands it', () => {
  // A caller lays out the cell and talks to the box, so the two are told
  // apart: everything but the class and the style is the box's.
  it('lays the class and the style on the cell', () => {
    const one = mount(AutosizeTextarea, {
      props: { text: 'said' },
      attrs: { class: 'w-40', style: 'margin: 4px' },
    })
    expect(one.get('.autosize').attributes('style')).toContain('margin: 4px')
    expect(box(one).classes()).not.toContain('w-40')
  })

  it('puts everything else on the box', () => {
    const one = mount(AutosizeTextarea, {
      props: { text: 'said' },
      attrs: { 'aria-label': 'The front of the card', placeholder: 'Nothing yet', id: 'front' },
    })
    expect(box(one).attributes('aria-label')).toBe('The front of the card')
    expect(box(one).attributes('placeholder')).toBe('Nothing yet')
    expect(box(one).attributes('id')).toBe('front')
    expect(one.get('.autosize').attributes('aria-label')).toBeUndefined()
  })

  it('hands back the box itself, for a caller putting the caret in it', () => {
    const one = mountBox()
    expect(one.vm.box).toBe(box(one).element)
  })
})
