import { mount } from '@vue/test-utils'
import { effectScope, nextTick, ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import TreeRow from './TreeRow.vue'
import type { ShownRow } from '../lib/row'
import { useRenameFocus } from '../model/useRenameFocus'

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
  it('renders a treeitem with level, name, and icon slot without a native title attribute', () => {
    const row = mountRow()
    expect(row.attributes('role')).toBe('treeitem')
    expect(row.attributes('aria-level')).toBe('2')
    expect(row.get('.tree__name').text()).toBe('Plans')
    expect(row.get('.tree__name').attributes('title')).toBeUndefined()
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

  it('focuses only when renaming turns on and a field is present', async () => {
    const isRenaming = ref(true)
    const focus = vi.fn<() => void>()
    const field = ref<{ focus(): void } | null>({ focus })
    const getIsRenaming = () => isRenaming.value
    const scope = effectScope()
    scope.run(() => useRenameFocus(getIsRenaming, field))

    isRenaming.value = false
    await nextTick()
    await nextTick()
    expect(focus).not.toHaveBeenCalled()

    field.value = null
    isRenaming.value = true
    await nextTick()
    await nextTick()
    expect(focus).not.toHaveBeenCalled()
    scope.stop()
  })
})

describe('a row status', () => {
  it('is a named cloud or a named loading ring at the end of the row, and none otherwise', async () => {
    const row = mountRow()
    expect(row.find('[data-status]').exists()).toBe(false)

    await row.setProps({ row: { ...ROW, status: { kind: 'cloud', label: 'Not here' } } })
    expect(row.get('[data-status="cloud"]').attributes('aria-label')).toBe('Not here')

    await row.setProps({ row: { ...ROW, status: { kind: 'loading', label: 'Coming' } } })
    expect(row.get('[data-status="loading"]').attributes('aria-label')).toBe('Coming')
    expect(row.find('[data-status="cloud"]').exists()).toBe(false)
  })

  it('draws the one progress ring for a loading row, filled where the progress is known', async () => {
    const row = mountRow({ row: { ...ROW, status: { kind: 'loading', label: 'Coming' } } })
    expect(row.get('[data-status="loading"] .progress-ring').attributes('data-known')).toBe('false')

    await row.setProps({
      row: { ...ROW, status: { kind: 'loading', label: 'Coming', progress: 0.4 } },
    })
    const ring = row.get('[data-status="loading"] .progress-ring')
    expect(ring.attributes('data-known')).toBe('true')
    expect(ring.get('.progress-ring__arc').attributes('stroke-dasharray')).toBe('40 100')
  })
})
