/**
 * What a button is drawn as and what it is rendered as. The sizes and colours
 * are classes of the tokens, so they are read off the element.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Button from './Button.vue'
import { buttonVariants } from './variants'

describe('what a button is', () => {
  it('is a button element that is not submitted by default', () => {
    const button = mount(Button, { slots: { default: 'Save' } })
    expect(button.element.tagName).toBe('BUTTON')
    expect(button.attributes('data-slot')).toBe('button')
    expect(button.text()).toBe('Save')
  })

  it('is drawn solid and at the default size where it is told nothing', () => {
    const button = mount(Button)
    expect(button.classes()).toEqual(expect.arrayContaining(['bg-accent', 'h-8', 'px-3']))
  })

  it('is a link that looks like a button where it is rendered as an anchor', () => {
    const button = mount(Button, { props: { as: 'a' }, attrs: { href: '/notes' } })
    expect(button.element.tagName).toBe('A')
    expect(button.attributes('href')).toBe('/notes')
    expect(button.classes()).toContain('bg-accent')
  })

  it('takes the element of its child where it is told to', () => {
    const button = mount(Button, {
      props: { asChild: true },
      slots: { default: '<a href="/x" class="own">Open</a>' },
    })
    expect(button.element.tagName).toBe('A')
    expect(button.classes()).toEqual(expect.arrayContaining(['own', 'bg-accent']))
  })
})

describe('the variants', () => {
  it.each([
    ['solid', 'bg-accent'],
    ['outline', 'border-field-rule'],
    ['danger', 'border-alarm'],
    ['ghost', 'hover:bg-raised'],
  ] as const)('draws the %s variant with its own ground', (variant, own) => {
    const button = mount(Button, { props: { variant } })
    expect(button.classes()).toContain(own)
  })

  it.each([
    ['small', 'h-7'],
    ['icon', 'size-action'],
    ['icon-small', 'size-7'],
  ] as const)('draws the %s size at its own height', (size, own) => {
    const button = mount(Button, { props: { size } })
    expect(button.classes()).toContain(own)
  })

  it('leaves out the colour of a variant that the next one does not have', () => {
    expect(buttonVariants({ variant: 'ghost' })).not.toContain('bg-accent')
  })

  it('lets a class of the caller win over a class of the size', () => {
    const button = mount(Button, { props: { class: 'h-12' } })
    expect(button.classes()).toContain('h-12')
    expect(button.classes()).not.toContain('h-8')
  })

  it('disables its pointer while it is disabled', () => {
    const button = mount(Button, { attrs: { disabled: true } })
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.classes()).toContain('disabled:pointer-events-none')
  })
})
