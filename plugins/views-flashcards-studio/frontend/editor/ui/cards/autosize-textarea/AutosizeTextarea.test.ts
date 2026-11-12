/**
 * A textarea whose height fits its content: the mirror element behind it that
 * carries the same text, where caller attributes are applied, and the event
 * emitted on typing.
 */
import { mount } from '@vue/test-utils'
import { h } from 'vue'
import { describe, expect, it } from 'vitest'
import AutosizeTextarea from './AutosizeTextarea.vue'

const mountBox = (props: Record<string, unknown> = {}) =>
  mount(AutosizeTextarea, { props: { text: 'said', ...props } })

const box = (one: ReturnType<typeof mountBox>) => one.get<HTMLTextAreaElement>('.autosize__box')

describe('the box', () => {
  it('holds the text it is given', () => {
    expect(box(mountBox({ text: 'two words' })).element.value).toBe('two words')
  })

  //   // The cell is sized by the mirror element, so the mirror has to carry the same
  //   // text as the textarea or the textarea has the wrong height.
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

describe('what a caller passes it', () => {
  //   // A caller styles the cell and addresses the textarea, so the two are
  //   // separated: everything except the class and the style goes to the textarea.
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

  it('returns the box itself, for a caller putting the caret in it', () => {
    const one = mountBox()
    expect(one.vm.box).toBe(box(one).element)
  })
})

describe('what the box is called', () => {
  it('is named by a label pointing at the id it is given', () => {
    const named = mount(
      () => [
        h('label', { for: 'front' }, 'Front'),
        h(AutosizeTextarea, { text: 'said', id: 'front' }),
      ],
      { attachTo: document.body },
    )
    const field = named.get<HTMLTextAreaElement>('textarea').element

    expect([...(field.labels ?? [])].map((label) => label.textContent)).toEqual(['Front'])
    named.unmount()
  })

  it('makes up an id of its own where it is given none, and no two alike', () => {
    const pair = mount(() =>
      h('div', [h(AutosizeTextarea, { text: 'said' }), h(AutosizeTextarea, { text: 'said' })]),
    )
    const ids = pair.findAll('textarea').map((field) => field.attributes('id'))

    expect(ids.every(Boolean)).toBe(true)
    expect(new Set(ids).size).toBe(2)
  })
})
