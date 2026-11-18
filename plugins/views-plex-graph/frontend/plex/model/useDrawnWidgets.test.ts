/** The group of widgets of a node is measured while a widget draws, and released when the node is removed. */
import { effectScope, nextTick, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useDrawnWidgets } from './useDrawnWidgets'

const createRows = () => {
  const rows = document.createElement('div')
  const row = document.createElement('div')
  row.append(document.createElement('span'))
  rows.append(row)
  return rows
}

describe('the widgets of a node', () => {
  it('are released when the node is removed', async () => {
    const change = vi.fn()
    const rows = ref<HTMLElement | null>(null)
    const scope = effectScope()
    scope.run(() => useDrawnWidgets(rows, change))

    rows.value = createRows()
    await nextTick()
    expect(change).toHaveBeenLastCalledWith(rows.value)

    scope.stop()

    expect(change).toHaveBeenLastCalledWith(null)
  })

  it('are not released by a node that never drew them', () => {
    const change = vi.fn()
    const scope = effectScope()
    scope.run(() => useDrawnWidgets(ref<HTMLElement | null>(null), change))

    scope.stop()

    expect(change).not.toHaveBeenCalled()
  })
})
