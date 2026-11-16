import { mount } from '@vue/test-utils'
import { h } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import TreeRow from './TreeRow.vue'
import type { ShownRow } from '../lib/row'

const ROW = {
  id: 'plans/Friday',
  name: 'Friday',
  level: 2,
  hasChildren: false,
  isOpen: false,
  isLast: false,
} as unknown as ShownRow

let mounted: { unmount: () => void } | null = null

const mountRow = (slots: Record<string, unknown>) => {
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
    },
    slots: { default: '<i class="icon" />', ...slots },
  })
  mounted = row
  return row
}

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

describe('the end of a tree row', () => {
  it('draws what the end slot gives after the name, inside the row', () => {
    const row = mountRow({ end: () => h('b', { 'data-end': '' }, 'two dots') })

    const end = row.get('[data-end]')
    expect(end.text()).toBe('two dots')
    expect(row.element.contains(end.element)).toBe(true)
    const name = row.get('.tree__name').element
    expect(
      name.compareDocumentPosition(end.element) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  it('gives the end slot the id of the row', () => {
    const asked: string[] = []
    mountRow({
      end: ({ id }: { id: string }) => {
        asked.push(id)
        return h('b', { 'data-end': id })
      },
    })

    expect(asked).toContain('plans/Friday')
  })
})
