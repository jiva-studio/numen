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
      isDragged: false,
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

describe('TreeRow attributes', () => {
  it('renders a treeitem with level, title, and icon slot', () => {
    const row = mountRow()
    expect(row.attributes('role')).toBe('treeitem')
    expect(row.attributes('aria-level')).toBe('2')
    expect(row.get('.tree__name').attributes('title')).toBe('Plans')
    expect(row.find('.icon').exists()).toBe(true)
  })

  it('sets tabindex to 0 only when isTabStop is true', async () => {
    const row = mountRow()
    expect(row.attributes('tabindex')).toBe('-1')
    await row.setProps({ isTabStop: true })
    expect(row.attributes('tabindex')).toBe('0')
  })

  it('sets aria-expanded only when row has children', async () => {
    const row = mountRow()
    expect(row.attributes('aria-expanded')).toBeUndefined()
    await row.setProps({ row: { ...ROW, hasChildren: true, isOpen: true } })
    expect(row.attributes('aria-expanded')).toBe('true')
  })

  it('hides data-selected while isRenaming is true', async () => {
    const row = mountRow({ isSelected: true })
    expect(row.attributes('data-selected')).toBe('')
    await row.setProps({ isRenaming: true })
    expect(row.attributes('data-selected')).toBeUndefined()
    expect(row.attributes('data-renaming')).toBe('')
  })
})

describe('renaming', () => {
  it('renders an input and focuses it when isRenaming becomes true', async () => {
    const row = mountRow()
    await row.setProps({ isRenaming: true })
    await nextTick()
    await nextTick()

    expect(row.find('.tree__name').exists()).toBe(false)
    expect(document.activeElement).toBe(row.get('input').element)
  })

  it('renders the row name when isRenaming becomes false', async () => {
    const row = mountRow({ isRenaming: true })
    await row.setProps({ isRenaming: false })
    expect(row.find('input').exists()).toBe(false)
    expect(row.get('.tree__name').text()).toBe('Plans')
  })
})
