/**
 * What a line of choices says, what it opens, and what it hands back.
 *
 * The choices are drawn at the end of the document, so they are read off the
 * document. A control a hand can open and a keyboard cannot is not a control.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import Select from './Select.vue'

const CHOICES = [
  { id: 'small', text: 'Small' },
  { id: 'medium', text: 'Medium' },
  { id: 'large', text: 'Large' },
]

type SelectProps = InstanceType<typeof Select>['$props']

let mounted: { unmount: () => void } | null = null

const mountSelect = (props: Partial<SelectProps> = {}) => {
  const control = mount(Select, {
    props: { choices: CHOICES, modelValue: 'small', ...props },
    attachTo: document.body,
  })
  mounted = control
  return control
}

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

/** Twice: the choices measure and take the keyboard a tick after they open. */
const settle = async () => {
  await nextTick()
  await nextTick()
}

const line = (control: ReturnType<typeof mountSelect>) => control.get('button[data-slot="select"]')
const rows = () => Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item'))
const words = () => rows().map((one) => one.querySelector('.menu__text')?.textContent?.trim())
const shelves = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('.menu__group-name')).map((one) =>
    one.textContent?.trim(),
  )

/** Every choice the control has handed on, in the order it handed them on. */
const getEmitted = (control: ReturnType<typeof mountSelect>): readonly unknown[] =>
  (control.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

const openSelect = async (control: ReturnType<typeof mountSelect>) => {
  await line(control).trigger('click')
  await settle()
}

describe('the line at rest', () => {
  it('says what is in force', () => {
    expect(line(mountSelect({ modelValue: 'medium' })).text()).toContain('Medium')
  })

  it('says the value itself where it is none of the choices', () => {
    expect(line(mountSelect({ modelValue: 'enormous' })).text()).toContain('enormous')
    expect(getEmitted(mountSelect({ modelValue: 'enormous' }))).toStrictEqual([])
  })

  it('says the stand-in where nothing is in force', () => {
    const control = mountSelect({ modelValue: '', placeholder: 'Nothing' })
    expect(line(control).text()).toContain('Nothing')
  })

  it('draws the mark itself, and opens nothing until it is asked', () => {
    const control = mountSelect()
    expect(control.find('.select__mark').exists()).toBe(true)
    expect(line(control).attributes('aria-expanded')).toBe('false')
    expect(rows()).toHaveLength(0)
  })

  it('is not opened while nobody may turn it', async () => {
    const control = mountSelect({ disabled: true })
    expect(line(control).attributes('disabled')).toBeDefined()
    await openSelect(control)
    expect(rows()).toHaveLength(0)
  })

  it('answers to the label that names it', () => {
    const control = mountSelect({ id: 'settings-size' } as Partial<SelectProps>)
    expect(line(control).attributes('id')).toBe('settings-size')
  })
})

describe('the choices offered', () => {
  it('are drawn in the order they were given', async () => {
    const control = mountSelect()
    await openSelect(control)
    expect(words()).toStrictEqual(['Small', 'Medium', 'Large'])
  })

  it('stand on the shelves they name, in the runs they arrive in', async () => {
    const control = mountSelect({
      choices: [
        { id: 'numen', text: 'numen', group: 'Ships with numen' },
        { id: 'sea', text: 'sea', group: 'Yours' },
      ],
      modelValue: 'numen',
    })
    await openSelect(control)
    expect(shelves()).toStrictEqual(['Ships with numen', 'Yours'])
  })

  it('stand on no shelf where they name none', async () => {
    await openSelect(mountSelect())
    expect(shelves()).toStrictEqual([])
  })

  it('carry the second line a choice was given', async () => {
    const control = mountSelect({
      choices: [{ id: 'small', text: 'Small', detail: 'somewhere/small.onnx' }],
      modelValue: 'small',
    })
    await openSelect(control)
    expect(document.body.querySelector('.menu__detail')?.textContent?.trim()).toBe(
      'somewhere/small.onnx',
    )
  })

  it('say so where there are none', async () => {
    const control = mountSelect({ choices: [], modelValue: '' })
    await openSelect(control)
    expect(document.body.querySelector('.menu__silence')?.textContent?.trim()).toBe(
      'Nothing to choose',
    )
  })
})

describe('choosing', () => {
  it('hands back the identifier it was given', async () => {
    const control = mountSelect()
    await openSelect(control)
    await rows()[2]!.click()
    expect(getEmitted(control)).toStrictEqual(['large'])
  })

  it('hands nothing back where the choices were put away untouched', async () => {
    const control = mountSelect()
    await openSelect(control)
    document.dispatchEvent(new Event('scroll', { bubbles: true }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(getEmitted(control)).toStrictEqual([])
    expect(line(control).attributes('aria-expanded')).toBe('false')
  })
})

describe('the keyboard', () => {
  it('opens the choices standing on the one in force', async () => {
    const control = mountSelect({ modelValue: 'medium' })
    await line(control).trigger('keydown', { key: 'ArrowDown' })
    await settle()
    expect(document.activeElement).toBe(rows()[1])
  })

  it('walks the choices with the arrows', async () => {
    const control = mountSelect({ modelValue: 'small' })
    await line(control).trigger('keydown', { key: 'ArrowDown' })
    await settle()
    rows()[0]!.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    await nextTick()
    expect(document.activeElement).toBe(rows()[1])
  })

  it('jumps to the choice a letter begins', async () => {
    const control = mountSelect({ modelValue: 'small' })
    await line(control).trigger('keydown', { key: 'ArrowUp' })
    await settle()
    rows()[0]!.dispatchEvent(new KeyboardEvent('keydown', { key: 'l', bubbles: true }))
    await nextTick()
    expect(document.activeElement).toBe(rows()[2])
  })

  it('gives the keyboard back to the line it was opened from', async () => {
    const control = mountSelect()
    await openSelect(control)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(document.activeElement).toBe(line(control).element)
  })
})

describe('how wide the choices are drawn', () => {
  it('are told how wide the line asking for them is', async () => {
    const control = mountSelect()
    // jsdom measures nothing, so the width is the one the box reports.
    line(control).element.getBoundingClientRect = () =>
      ({ left: 0, bottom: 32, width: 288 }) as DOMRect
    await openSelect(control)

    expect(
      document.body.querySelector<HTMLElement>('.menu')?.style.getPropertyValue('--asking'),
    ).toBe('288px')
  })
})
