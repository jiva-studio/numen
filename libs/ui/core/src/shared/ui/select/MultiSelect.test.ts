/**
 * The line of a select with several choices in force, the list it opens and what a press emits.
 * The list is drawn at the end of the document, so the tests query the document.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import MultiSelect from './MultiSelect.vue'

type MultiSelectProps = InstanceType<typeof MultiSelect>['$props']

let mounted: { unmount: () => void } | null = null

const mountMultiple = (selected: readonly string[], props: Partial<MultiSelectProps> = {}) => {
  const control = mount(MultiSelect, {
    props: {
      selected,
      placeholder: 'Nothing chosen',
      choices: [
        { id: 'small', text: 'Small', detail: 'First group' },
        { id: 'medium', text: 'Medium', detail: 'Second group' },
        { id: 'large', text: 'Large', detail: 'First group' },
      ],
      ...props,
    },
    attachTo: document.body,
  })
  mounted = control
  return control
}

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

const settle = async () => {
  await nextTick()
  await nextTick()
}

const line = (control: ReturnType<typeof mountMultiple>) =>
  control.get('button[data-slot="select"]')
const getOptions = () => Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]'))
const getMarks = () => document.body.querySelectorAll('.searchable-list__mark')
const getSelections = (control: ReturnType<typeof mountMultiple>): readonly unknown[] =>
  (control.emitted('update:selected') ?? []).map((response) => (response as unknown[])[0])
const getToggles = (control: ReturnType<typeof mountMultiple>): readonly unknown[] =>
  control.emitted('toggle') ?? []

const openSelect = async (control: ReturnType<typeof mountMultiple>) => {
  await line(control).trigger('click')
  await settle()
}

describe('the line at rest', () => {
  it('says the stand-in where none is chosen', () => {
    expect(line(mountMultiple([])).text()).toContain('Nothing chosen')
  })

  it('says the text of the one chosen', () => {
    expect(line(mountMultiple(['medium'])).text()).toBe('Medium')
  })

  it('says the first text and how many more are chosen', () => {
    expect(line(mountMultiple(['large', 'small', 'medium'])).text()).toContain('Large +2')
  })

  it('writes the count of the rest as the caller words it', () => {
    const control = mountMultiple(['large', 'small'], { getMoreText: (count) => `and ${count}` })
    expect(line(control).text()).toContain('Large and 1')
  })

  it('stays closed while it is disabled', async () => {
    const control = mountMultiple([], { isDisabled: true })
    await openSelect(control)
    expect(getOptions()).toHaveLength(0)
  })
})

describe('the list', () => {
  it('marks every chosen row once, and keeps the second line', async () => {
    const control = mountMultiple(['small', 'large'])
    await openSelect(control)
    expect(getOptions().map((one) => one.getAttribute('aria-current'))).toStrictEqual([
      'true',
      null,
      'true',
    ])
    expect(getMarks()).toHaveLength(2)
    expect(document.body.querySelector('.searchable-list__detail')?.textContent).toBe('First group')
  })

  it('emits a choice switched on and stays open', async () => {
    const control = mountMultiple(['small'])
    await openSelect(control)
    getOptions()[1]!.click()
    await nextTick()
    expect(getSelections(control)).toStrictEqual([['small', 'medium']])
    expect(getToggles(control)).toStrictEqual([['medium', true]])
    expect(getOptions()).toHaveLength(3)
  })

  it('emits a chosen choice switched off when it is pressed again', async () => {
    const control = mountMultiple(['small', 'medium'])
    await openSelect(control)
    getOptions()[0]!.click()
    await nextTick()
    expect(getSelections(control)).toStrictEqual([['medium']])
    expect(getToggles(control)).toStrictEqual([['small', false]])
  })

  it('narrows the choices by what is typed and toggles with Enter', async () => {
    const control = mountMultiple([])
    await openSelect(control)
    const search = document.body.querySelector<HTMLInputElement>(
      '[data-numen-searchable-list] input',
    )!
    search.value = 'lar'
    search.dispatchEvent(new Event('input'))
    await nextTick()
    expect(getOptions()).toHaveLength(1)
    search.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await nextTick()
    expect(getToggles(control)).toStrictEqual([['large', true]])
  })
})

describe('the keyboard', () => {
  it('opens the list with the arrow keys', async () => {
    const control = mountMultiple([])
    await line(control).trigger('keydown', { key: 'ArrowDown' })
    await settle()
    expect(getOptions()).toHaveLength(3)
  })

  it('closes the list on Escape and gives the keyboard back to the line', async () => {
    const control = mountMultiple([])
    await openSelect(control)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await settle()
    expect(getOptions()).toHaveLength(0)
    expect(line(control).attributes('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(line(control).element)
  })
})
