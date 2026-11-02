/**
 * What the searchable list does: the rows a query keeps, where the keyboard is,
 * what Enter and a click choose, how it is put away, and what it announces.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import SearchableList from './SearchableList.vue'
import type { ListItem } from './rows'

const MODELS: ListItem[] = [
  { id: 'opus', text: 'Opus 5.5', group: 'Claude Code' },
  { id: 'sonnet', text: 'Sonnet 5.5', group: 'Claude Code' },
  { id: 'gemini', text: 'Gemini 4', group: 'Antigravity', detail: 'by Google' },
]

type ListProps = InstanceType<typeof SearchableList>['$props']

let mounted: { unmount: () => void } | null = null

const mountList = (props: Partial<ListProps> = {}, slots: Record<string, string> = {}) => {
  const list = mount(SearchableList, {
    props: { items: MODELS, label: 'Models', placeholder: 'Search models', ...props },
    slots,
    attachTo: document.body,
  })
  mounted = list
  return list
}

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

const getInput = (list: ReturnType<typeof mountList>) => list.get('input')
const getOptions = (list: ReturnType<typeof mountList>) => list.findAll('[role="option"]')
const getTexts = (list: ReturnType<typeof mountList>) =>
  getOptions(list).map((one) => one.text().replace(/\s+/g, ' '))
const getActive = (list: ReturnType<typeof mountList>) =>
  getOptions(list).findIndex((one) => one.attributes('aria-selected') === 'true')
const press = (list: ReturnType<typeof mountList>, key: string) =>
  getInput(list).trigger('keydown', { key })

describe('the frame', () => {
  it('holds the search line first and the list after it, in one container', () => {
    const list = mountList()
    const frame = list.get('[data-numen-searchable-list]')
    expect(frame.element.firstElementChild).toBe(getInput(list).element)
    expect(frame.element.contains(list.get('[role="listbox"]').element)).toBe(true)
  })

  it('takes the keyboard on the search line as it appears', async () => {
    const list = mountList()
    await nextTick()
    await nextTick()
    expect(document.activeElement).toBe(getInput(list).element)
  })
})

describe('the rows', () => {
  it('are every item, in the order given', () => {
    expect(getTexts(mountList())).toStrictEqual(['Opus 5.5', 'Sonnet 5.5', 'Gemini 4by Google'])
  })

  it('keep the items that name every word typed', async () => {
    const list = mountList()
    await getInput(list).setValue('claude son')
    expect(getTexts(list)).toStrictEqual(['Sonnet 5.5'])
  })

  it('are left as given where the caller does the filtering', async () => {
    const list = mountList({ isFiltered: false })
    await getInput(list).setValue('zzz')
    expect(getOptions(list)).toHaveLength(3)
  })

  it('say so where no item is left', async () => {
    const list = mountList({ emptyText: 'No models' })
    await getInput(list).setValue('zzz')
    expect(getOptions(list)).toHaveLength(0)
    expect(list.get('[role="status"]').text()).toBe('No models')
  })

  it('say that an answer is awaited where none is there yet', () => {
    const list = mountList({ items: [], isPending: true, pendingText: 'Searching' })
    expect(list.get('[role="status"]').text()).toBe('Searching')
    expect(list.get('[data-numen-searchable-list]').attributes('aria-busy')).toBe('true')
  })

  it('stay where they are while an answer is awaited', async () => {
    const list = mountList()
    await list.setProps({ isPending: true })
    expect(getOptions(list)).toHaveLength(3)
  })

  it('put the name of a group over the first row of the group only', () => {
    const list = mountList()
    const names = list.findAll('.searchable-list__heading').map((one) => one.text())
    expect(names).toStrictEqual(['Claude Code', 'Antigravity'])
  })

  it('draw what the slots give before and after the words of an item', () => {
    const list = mountList(
      {},
      { leading: '<i class="before">L</i>', trailing: '<i class="after">T</i>' },
    )
    expect(list.findAll('.before')).toHaveLength(3)
    expect(list.findAll('.after')).toHaveLength(3)
  })

  it('mark the item in force', () => {
    const list = mountList({ modelValue: 'sonnet' })
    expect(getOptions(list).map((one) => one.attributes('aria-current'))).toStrictEqual([
      undefined,
      'true',
      undefined,
    ])
  })

  it('stay shut until something is typed where a query is needed', async () => {
    const list = mountList({ needsQuery: true })
    expect(list.get('.searchable-list__area').attributes('data-open')).toBe('false')
    expect(getOptions(list)).toHaveLength(0)
    await getInput(list).setValue('o')
    expect(list.get('.searchable-list__area').attributes('data-open')).toBe('true')
    expect(getOptions(list).length).toBeGreaterThan(0)
  })
})

describe('the keyboard', () => {
  it('starts on the first row, or on the item in force', () => {
    expect(getActive(mountList())).toBe(0)
    mounted?.unmount()
    expect(getActive(mountList({ modelValue: 'sonnet' }))).toBe(1)
  })

  it('moves down and up, and round at both ends', async () => {
    const list = mountList()
    await press(list, 'ArrowDown')
    expect(getActive(list)).toBe(1)
    await press(list, 'ArrowUp')
    await press(list, 'ArrowUp')
    expect(getActive(list)).toBe(2)
    await press(list, 'ArrowDown')
    expect(getActive(list)).toBe(0)
  })

  it('goes to the ends on Home and End while nothing is typed, and leaves the caret otherwise', async () => {
    const list = mountList()
    await press(list, 'End')
    expect(getActive(list)).toBe(2)
    await press(list, 'Home')
    expect(getActive(list)).toBe(0)
    await getInput(list).setValue('s')
    await press(list, 'End')
    expect(getActive(list)).toBe(0)
  })

  it('chooses the active row on Enter, and reports it as the item in force', async () => {
    const list = mountList()
    await press(list, 'ArrowDown')
    await press(list, 'Enter')
    expect(list.emitted('select')).toStrictEqual([['sonnet']])
    expect(list.emitted('update:modelValue')).toStrictEqual([['sonnet']])
  })

  it('chooses a row by a click, and moves to it with the pointer', async () => {
    const list = mountList()
    await getOptions(list)[2]!.trigger('pointerenter')
    expect(getActive(list)).toBe(2)
    await getOptions(list)[2]!.trigger('click')
    expect(list.emitted('select')).toStrictEqual([['gemini']])
  })

  it('keeps the row where the user put it as the items are replaced', async () => {
    const list = mountList({ isFiltered: false })
    await press(list, 'ArrowDown')
    await press(list, 'ArrowDown')
    await list.setProps({ items: [...MODELS, { id: 'extra', text: 'Extra' }] })
    expect(getActive(list)).toBe(2)
    expect(getOptions(list)).toHaveLength(4)
  })

  it('returns to the first row as the query changes', async () => {
    const list = mountList()
    await press(list, 'ArrowDown')
    await getInput(list).setValue('5.5')
    expect(getActive(list)).toBe(0)
  })

  it('does nothing on Enter where there is no row', async () => {
    const list = mountList()
    await getInput(list).setValue('zzz')
    await press(list, 'Enter')
    expect(list.emitted('select')).toBeUndefined()
  })
})

describe('the action row', () => {
  const FOUND: ListItem[] = [
    { id: 'notes/books-about.md', text: 'Books about' },
    { id: 'notes/books.md', text: 'Books' },
  ]
  const action = '<b class="act">Create "{{ params.query }}"</b>'
  const mountAction = (props: Partial<ListProps> = {}) =>
    mountList(
      { items: [], isFiltered: false, hasAction: true, ...props },
      { action: `<template #action="params">${action}</template>` },
    )

  it('shows no row until something is typed', () => {
    expect(getOptions(mountAction())).toHaveLength(0)
  })

  it('is the first row and the active row, and takes the query', async () => {
    const list = mountAction({ items: FOUND })
    await getInput(list).setValue('Boo')
    expect(getTexts(list)).toStrictEqual(['Create "Boo"', 'Books about', 'Books'])
    expect(getActive(list)).toBe(0)
  })

  it('stays row 1, the same element and active, as the items arrive', async () => {
    const list = mountAction()
    await getInput(list).setValue('Boo')
    const row = getOptions(list)[0]!.element
    await list.setProps({ items: FOUND })
    expect(getOptions(list)[0]!.element).toBe(row)
    expect(getTexts(list)[0]).toBe('Create "Boo"')
    expect(getActive(list)).toBe(0)
  })

  it('takes Enter and a click as an act with the typed query', async () => {
    const list = mountAction({ items: FOUND })
    await getInput(list).setValue(' Boo ')
    await press(list, 'Enter')
    await getOptions(list)[0]!.trigger('click')
    expect(list.emitted('act')).toStrictEqual([['Boo'], ['Boo']])
    expect(list.emitted('select')).toBeUndefined()
  })

  it('gives row 1 to the item that is exactly the query, in the place of the action', async () => {
    const list = mountAction()
    await getInput(list).setValue('books')
    const row = getOptions(list)[0]!.element
    await list.setProps({ items: FOUND })
    expect(getTexts(list)).toStrictEqual(['Books', 'Books about'])
    expect(getOptions(list)[0]!.element).toBe(row)
    expect(getActive(list)).toBe(0)
    await press(list, 'Enter')
    expect(list.emitted('select')).toStrictEqual([['notes/books.md']])
    expect(list.emitted('act')).toBeUndefined()
  })
})

describe('being put away', () => {
  it('asks to close on Escape', async () => {
    const list = mountList()
    await press(list, 'Escape')
    expect(list.emitted('close')).toHaveLength(1)
  })

  it('asks to close on a press outside, and not on a press inside', () => {
    const list = mountList()
    getInput(list).element.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(list.emitted('close')).toBeUndefined()
    document.body.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(list.emitted('close')).toHaveLength(1)
  })
})

describe('what it announces', () => {
  it('is a combobox over a listbox, pointing at the active option', async () => {
    const list = mountList()
    const input = getInput(list)
    const box = list.get('[role="listbox"]')
    expect(input.attributes('role')).toBe('combobox')
    expect(input.attributes('aria-expanded')).toBe('true')
    expect(input.attributes('aria-controls')).toBe(box.attributes('id'))
    expect(input.attributes('aria-activedescendant')).toBe(getOptions(list)[0]!.attributes('id'))
    await press(list, 'ArrowDown')
    expect(input.attributes('aria-activedescendant')).toBe(getOptions(list)[1]!.attributes('id'))
  })

  it('names the frame, the list and the search line', () => {
    const list = mountList()
    expect(list.get('[data-numen-searchable-list]').attributes('aria-label')).toBe('Models')
    expect(list.get('[role="listbox"]').attributes('aria-label')).toBe('Models')
    expect(getInput(list).attributes('aria-label')).toBe('Search models')
  })

  it('points at nothing where there is no row', async () => {
    const list = mountList()
    await getInput(list).setValue('zzz')
    expect(getInput(list).attributes('aria-expanded')).toBe('false')
    expect(getInput(list).attributes('aria-activedescendant')).toBeUndefined()
  })
})
