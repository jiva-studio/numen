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

  it('is drawn solid and at the default size where it is given nothing', () => {
    const button = mount(Button)
    expect(button.classes()).toEqual(
      expect.arrayContaining([
        'inline-flex',
        'shrink-0',
        'items-center',
        'justify-center',
        'gap-2',
        'whitespace-nowrap',
        'font-sans',
        'text-base',
        'font-medium',
        'duration-hover',
        'ease-numen',
        'cursor-pointer',
        'transition-[background-color,color]',
        'ring-numen',
        'outline-none',
        'disabled:pointer-events-none',
        'disabled:opacity-50',
        '[&_svg]:pointer-events-none',
        '[&_svg]:shrink-0',
        'bg-accent',
        'text-accent-ink',
        'hover:bg-[color-mix(in_oklab,var(--numen-accent),var(--numen-ink)_8%)]',
        'rounded-node',
        'h-8',
        'px-3',
      ]),
    )
    expect(buttonVariants()).toContain('bg-accent')
    expect(buttonVariants()).toContain('h-8')
  })

  it('is a link that looks like a button where it is rendered as an anchor', () => {
    const button = mount(Button, { props: { as: 'a' }, attrs: { href: '/pages' } })
    expect(button.element.tagName).toBe('A')
    expect(button.attributes('href')).toBe('/pages')
    expect(button.classes()).toContain('bg-accent')
  })

  it('takes the element of its child where it is configured to', () => {
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
    [
      'solid',
      [
        'bg-accent',
        'text-accent-ink',
        'hover:bg-[color-mix(in_oklab,var(--numen-accent),var(--numen-ink)_8%)]',
      ],
    ],
    [
      'outline',
      [
        'border-field-rule',
        'bg-field',
        'text-ink',
        'border',
        'hover:bg-[color-mix(in_oklab,var(--numen-field-bg),var(--numen-ink)_8%)]',
      ],
    ],
    [
      'danger',
      [
        'border-alarm',
        'text-alarm',
        'border',
        'bg-[var(--numen-alarm-bg)]',
        'hover:bg-[color-mix(in_oklab,var(--numen-alarm-bg),var(--numen-surface)_40%)]',
      ],
    ],
    ['ghost', ['text-ink', 'hover:bg-raised']],
  ] as const)('draws the %s variant with its own ground', (variant, own) => {
    const button = mount(Button, { props: { variant } })
    expect(button.classes()).toEqual(expect.arrayContaining([...own]))
  })

  it.each([
    ['default', ['rounded-node', 'h-8', 'px-3']],
    ['small', ['rounded-node', 'text-small', 'h-7', 'px-2']],
    ['icon', ['size-action', 'rounded-pill']],
    ['icon-small', ['rounded-pill', 'size-7']],
  ] as const)('draws the %s size at its own height', (size, own) => {
    const button = mount(Button, { props: { size } })
    expect(button.classes()).toEqual(expect.arrayContaining([...own]))
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
