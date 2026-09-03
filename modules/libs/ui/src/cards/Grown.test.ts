/**
 * A box as tall as what it holds: what the ground behind it carries, what a
 * caller's attributes land on, and what is said when something is typed.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Grown from './Grown.vue'

const mountGrown = (props: Record<string, unknown> = {}) =>
  mount(Grown, { props: { text: 'said', ...props } })

const box = (grown: ReturnType<typeof mountGrown>) =>
  grown.get<HTMLTextAreaElement>('.grown__box')

describe('the box', () => {
  it('holds the text it is given', () => {
    expect(box(mountGrown({ text: 'two words' })).element.value).toBe('two words')
  })

  // The cell is sized by the ground, so the ground has to carry the same text
  // the box does or the box is the wrong height for it.
  it('is sized by a ground carrying the same text', () => {
    const grown = mountGrown({ text: 'a line\nand another' })
    expect(grown.get('.grown').attributes('data-grown')).toBe('a line\nand another')
  })

  it('is one line and never scrolls itself', () => {
    expect(box(mountGrown()).attributes('rows')).toBe('1')
  })

  it('says the text as it now reads when something is typed into it', async () => {
    const grown = mountGrown()
    await box(grown).setValue('said more')
    expect(grown.emitted('write')).toEqual([['said more']])
  })
})

describe('what a caller hands it', () => {
  // A caller lays out the cell and talks to the box, so the two are told
  // apart: everything but the class and the style is the box's.
  it('lays the class and the style on the cell', () => {
    const grown = mount(Grown, {
      props: { text: 'said' },
      attrs: { class: 'w-40', style: 'margin: 4px' },
    })
    expect(grown.get('.grown').attributes('style')).toContain('margin: 4px')
    expect(box(grown).classes()).not.toContain('w-40')
  })

  it('puts everything else on the box', () => {
    const grown = mount(Grown, {
      props: { text: 'said' },
      attrs: { 'aria-label': 'The front of the card', placeholder: 'Nothing yet', id: 'front' },
    })
    expect(box(grown).attributes('aria-label')).toBe('The front of the card')
    expect(box(grown).attributes('placeholder')).toBe('Nothing yet')
    expect(box(grown).attributes('id')).toBe('front')
    expect(grown.get('.grown').attributes('aria-label')).toBeUndefined()
  })

  it('hands back the box itself, for a caller putting the caret in it', () => {
    const grown = mountGrown()
    expect(grown.vm.box).toBe(box(grown).element)
  })
})
