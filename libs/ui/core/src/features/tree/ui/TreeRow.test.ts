/**
 * What a row says about itself, and where the keyboard goes when it is renamed.
 */
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import TreeRow from './TreeRow.vue'
import type { ShownRow } from '../lib/row'

const ROW = {
  id: 'plans',
  name: 'Plans',
  level: 2,
  hasChildren: false,
  isOpen: false,
  isLast: false,
} as unknown as ShownRow

let mounted: { unmount: () => void } | null = null

const mountRow = (props: Record<string, unknown> = {}) => {
  const row = mount(TreeRow, {
    attachTo: document.body,
    props: {
      row: ROW,
      name: 'Plans',
      isSelected: false,
      isLifted: false,
      isDropInside: false,
      isDropAbove: false,
      isTabStop: false,
      isRenaming: false,
      mark: {},
      ...props,
    },
    slots: { default: '<i class="icon" />' },
  })
  mounted = row
  return row
}

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

describe('what a row says', () => {
  it('is a tree item at its level, with its whole name on it', () => {
    const row = mountRow()
    expect(row.attributes('role')).toBe('treeitem')
    expect(row.attributes('aria-level')).toBe('2')
    expect(row.get('.tree__name').attributes('title')).toBe('Plans')
    expect(row.find('.icon').exists()).toBe(true)
  })

  it('is the one stop of the tree only where it is the tab stop', async () => {
    const row = mountRow()
    expect(row.attributes('tabindex')).toBe('-1')
    await row.setProps({ isTabStop: true })
    expect(row.attributes('tabindex')).toBe('0')
  })

  it('says whether it is open only where it has children', async () => {
    const row = mountRow()
    expect(row.attributes('aria-expanded')).toBeUndefined()
    await row.setProps({ row: { ...ROW, hasChildren: true, isOpen: true } })
    expect(row.attributes('aria-expanded')).toBe('true')
  })

  it('marks itself selected, and not while a name is being typed into it', async () => {
    const row = mountRow({ isSelected: true })
    expect(row.attributes('data-selected')).toBe('')
    await row.setProps({ isRenaming: true })
    expect(row.attributes('data-selected')).toBeUndefined()
    expect(row.attributes('data-renaming')).toBe('')
  })
})

describe('renaming', () => {
  it('draws a field in place of the name, and takes the keyboard into it', async () => {
    const row = mountRow()
    await row.setProps({ isRenaming: true })
    await nextTick()
    await nextTick()

    expect(row.find('.tree__name').exists()).toBe(false)
    expect(document.activeElement).toBe(row.get('input').element)
  })

  it('takes the field away and draws the name again once it is over', async () => {
    const row = mountRow({ isRenaming: true })
    await row.setProps({ isRenaming: false })
    expect(row.find('input').exists()).toBe(false)
    expect(row.get('.tree__name').text()).toBe('Plans')
  })
})
