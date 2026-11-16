/**
 * What a searchable list does in its multiple mode: the check mark of each row, the order of the
 * rows, what Enter and the action key do, when the "create" row is there, and what is not emitted.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { h, nextTick } from 'vue'
import SearchableList from './SearchableList.vue'
import type { ListItem } from './rows'

const TREES: ListItem[] = [
  { id: 'birch', text: 'Birch', state: 'unset' },
  { id: 'willow', text: 'Willow', state: 'unset' },
  { id: 'cedar', text: 'Cedar', state: 'mixed' },
  { id: 'alder', text: 'Alder', state: 'unset' },
  { id: 'maple', text: 'Maple', state: 'unset' },
  { id: 'fir', text: 'Fir', state: 'set' },
]

type ListProps = InstanceType<typeof SearchableList>['$props']

let mounted: { unmount: () => void } | null = null

const mountList = (props: Partial<ListProps> = {}) => {
  const list = mount(SearchableList, {
    props: {
      items: TREES,
      label: 'Trees',
      placeholder: 'Tree',
      multiple: true,
      ...props,
    },
    slots: { action: ({ query }: { query: string }) => h('span', `Create ${query}`) },
    attachTo: document.body,
  })
  mounted = list
  return list
}

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

type Mounted = ReturnType<typeof mountList>

const getInput = (list: Mounted) => list.get('input')
const getOptions = (list: Mounted) => list.findAll('[role="option"]')
const getTexts = (list: Mounted) => getOptions(list).map((one) => one.text().trim())
const getChecks = (list: Mounted) => getOptions(list).map((one) => one.attributes('aria-checked'))
const getActiveText = (list: Mounted) =>
  getOptions(list)
    .find((one) => one.attributes('aria-selected') === 'true')
    ?.text()
    .trim()
const press = (list: Mounted, key: string, more: KeyboardEventInit = {}) =>
  getInput(list).trigger('keydown', { key, ...more })

describe('the check marks', () => {
  it('say checked for a set item, mixed for a mixed one and not checked for an unset one', () => {
    const list = mountList({
      items: [
        { id: 'a', text: 'Alder', state: 'set' },
        { id: 'b', text: 'Birch', state: 'mixed' },
        { id: 'c', text: 'Cedar', state: 'unset' },
      ],
    })
    expect(getChecks(list)).toStrictEqual(['true', 'mixed', 'false'])
  })

  it('follow the state of an item as the caller changes it', async () => {
    const list = mountList({ items: [{ id: 'a', text: 'Alder', state: 'unset' }] })
    await list.setProps({ items: [{ id: 'a', text: 'Alder', state: 'set' }] })
    expect(getChecks(list)).toStrictEqual(['true'])
  })

  it('tell a screen reader that several rows can be checked', () => {
    expect(mountList().get('[role="listbox"]').attributes('aria-multiselectable')).toBe('true')
  })
})

describe('the order of the rows', () => {
  it('puts the set and the mixed items first, the recent ones next in the order given, and the rest by title', () => {
    const list = mountList({ recent: ['fir', 'willow', 'maple'] })
    expect(getTexts(list)).toStrictEqual(['Cedar', 'Fir', 'Willow', 'Maple', 'Alder', 'Birch'])
  })

  it('shows an item that is set and recent once, among the set ones', () => {
    const list = mountList({ recent: ['fir'] })
    expect(getTexts(list).filter((text) => text === 'Fir')).toHaveLength(1)
    expect(getTexts(list).indexOf('Fir')).toBeLessThan(getTexts(list).indexOf('Alder'))
  })

  it('ignores a recent id that no item has', () => {
    const list = mountList({ recent: ['gone', 'maple'] })
    expect(getTexts(list)).toStrictEqual(['Cedar', 'Fir', 'Maple', 'Alder', 'Birch', 'Willow'])
  })

  it('keeps that order among the items a typed word keeps', async () => {
    const list = mountList({ recent: ['fir', 'willow', 'maple'] })
    await getInput(list).setValue('a')
    expect(getTexts(list)).toStrictEqual(['Cedar', 'Maple', 'Alder'])
  })

  it('moves an item up among the set ones when the caller sets it', async () => {
    const list = mountList({
      items: [
        { id: 'a', text: 'Alder', state: 'unset' },
        { id: 'b', text: 'Birch', state: 'unset' },
        { id: 'c', text: 'Cedar', state: 'unset' },
      ],
    })
    await list.setProps({
      items: [
        { id: 'a', text: 'Alder', state: 'unset' },
        { id: 'b', text: 'Birch', state: 'unset' },
        { id: 'c', text: 'Cedar', state: 'set' },
      ],
    })
    expect(getTexts(list)).toStrictEqual(['Cedar', 'Alder', 'Birch'])
  })

  it('keeps the keyboard on the row it toggled when that row moves', async () => {
    const list = mountList({
      items: [
        { id: 'a', text: 'Alder', state: 'unset' },
        { id: 'b', text: 'Birch', state: 'unset' },
        { id: 'c', text: 'Cedar', state: 'unset' },
      ],
    })
    await press(list, 'ArrowDown')
    await press(list, 'ArrowDown')
    expect(getActiveText(list)).toBe('Cedar')

    await press(list, 'Enter')
    await list.setProps({
      items: [
        { id: 'a', text: 'Alder', state: 'unset' },
        { id: 'b', text: 'Birch', state: 'unset' },
        { id: 'c', text: 'Cedar', state: 'set' },
      ],
    })
    expect(getActiveText(list)).toBe('Cedar')
  })
})

describe('the keys', () => {
  it('toggles the highlighted row on Enter and does not choose it', async () => {
    const list = mountList()
    await press(list, 'Enter')
    expect(list.emitted('toggle')).toStrictEqual([['cedar']])
    expect(list.emitted('select')).toBeUndefined()
    expect(list.emitted('update:modelValue')).toBeUndefined()
  })

  it('toggles the row the arrows moved to', async () => {
    const list = mountList()
    await press(list, 'ArrowDown')
    await press(list, 'Enter')
    expect(list.emitted('toggle')).toStrictEqual([['fir']])
  })

  it('keeps the list open and the rows in place after Enter', async () => {
    const list = mountList()
    await press(list, 'Enter')
    await press(list, 'Enter')
    expect(list.emitted('close')).toBeUndefined()
    expect(getOptions(list)).toHaveLength(6)
    expect(list.emitted('toggle')).toStrictEqual([['cedar'], ['cedar']])
  })

  it('toggles the highlighted row on the action key with Control or with Meta', async () => {
    const list = mountList()
    await press(list, 'k', { ctrlKey: true })
    await press(list, 'ArrowDown')
    await press(list, 'k', { metaKey: true })
    expect(list.emitted('toggle')).toStrictEqual([['cedar'], ['fir']])
    expect(list.emitted('close')).toBeUndefined()
  })

  it('does not toggle on the letter k alone', async () => {
    const list = mountList()
    await press(list, 'k')
    expect(list.emitted('toggle')).toBeUndefined()
  })

  it('asks to be closed on Escape and toggles nothing', async () => {
    const list = mountList()
    await press(list, 'Escape')
    expect(list.emitted('close')).toHaveLength(1)
    expect(list.emitted('toggle')).toBeUndefined()
  })

  it('does not toggle a disabled item', async () => {
    const list = mountList({
      items: [{ id: 'a', text: 'Alder', state: 'unset', isDisabled: true }],
    })
    await press(list, 'Enter')
    expect(list.emitted('toggle')).toBeUndefined()
  })

  it('toggles the row that is clicked, and keeps the list open', async () => {
    const list = mountList()
    await getOptions(list)[1]!.trigger('click')
    expect(list.emitted('toggle')).toStrictEqual([['fir']])
    expect(list.emitted('close')).toBeUndefined()
  })
})

describe('the create row', () => {
  it('stands first once something is typed that no item is named', async () => {
    const list = mountList({ hasAction: true })
    await getInput(list).setValue('Harbour')
    expect(getTexts(list)[0]).toBe('Create Harbour')
  })

  it('is what Enter asks for, with the text as typed, and the list stays open', async () => {
    const list = mountList({ hasAction: true })
    await getInput(list).setValue('Harbour')
    await press(list, 'Enter')
    expect(list.emitted('act')).toStrictEqual([['Harbour']])
    expect(list.emitted('toggle')).toBeUndefined()
    expect(list.emitted('close')).toBeUndefined()
  })

  it('is asked for by the action key too', async () => {
    const list = mountList({ hasAction: true })
    await getInput(list).setValue('Harbour')
    await press(list, 'k', { ctrlKey: true })
    expect(list.emitted('act')).toStrictEqual([['Harbour']])
  })

  it('is not there while nothing is typed, and every item is', () => {
    const list = mountList({ hasAction: true })
    expect(getTexts(list)).toHaveLength(6)
    expect(getTexts(list).some((text) => text.startsWith('Create'))).toBe(false)
  })

  it('is not there while only spaces are typed', async () => {
    const list = mountList({ hasAction: true })
    await getInput(list).setValue('   ')
    expect(getTexts(list).some((text) => text.startsWith('Create'))).toBe(false)
  })

  it('is not there when an item has the typed text as its title, in any case and with spaces around', async () => {
    const list = mountList({ hasAction: true })
    await getInput(list).setValue('  cedar ')
    expect(getTexts(list)).toStrictEqual(['Cedar'])
  })

  it('is there while the typed text only begins an item title', async () => {
    const list = mountList({ hasAction: true })
    await getInput(list).setValue('ced')
    await nextTick()
    expect(getTexts(list)).toStrictEqual(['Create ced', 'Cedar'])
  })

  it('is there beside the items found, ahead of them', async () => {
    const list = mountList({ hasAction: true, recent: ['maple'] })
    await getInput(list).setValue('a')
    expect(getTexts(list)).toStrictEqual(['Create a', 'Cedar', 'Maple', 'Alder'])
  })
})
