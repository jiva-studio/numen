/**
 * What the rows do inside their frame: which one is marked, what a press and
 * a pointer say, and how the frame scrolls to the row the keyboard is on.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LIST_CONTEXT } from './context'
import SearchableListRows from './SearchableListRows.vue'
import type { ListRow } from './rows'

const ROWS: readonly ListRow[] = [
  { key: 'a', kind: 'item', item: { id: 'a', text: 'Opus', group: 'Claude' } },
  { key: 'b', kind: 'item', item: { id: 'b', text: 'Sonnet', group: 'Claude' } },
  { key: 'c', kind: 'item', item: { id: 'c', text: 'Mini', group: 'Numen' } },
]

const mountRows = (props: Record<string, unknown> = {}) =>
  mount(SearchableListRows, {
    attachTo: document.body,
    global: { provide: { [LIST_CONTEXT as symbol]: { slots: {}, getQuery: () => '' } } },
    props: {
      rows: ROWS,
      activeIndex: 1,
      current: 'a',
      listId: 'models',
      label: 'Models',
      ...props,
    },
  })

const createRect = (top: number, bottom: number): DOMRect =>
  ({ top, bottom, left: 0, right: 100, width: 100, height: bottom - top }) as DOMRect

/** The frame is `view` tall of `content`, and the active row is at `row`. */
function arrange(list: HTMLElement, frame: [number, number], row: [number, number], tall = true) {
  Object.defineProperty(list, 'scrollHeight', { value: tall ? 300 : 100, configurable: true })
  Object.defineProperty(list, 'clientHeight', { value: 100, configurable: true })
  vi.spyOn(list, 'getBoundingClientRect').mockReturnValue(createRect(...frame))
  const active = list.querySelector('[aria-selected="true"]')!
  vi.spyOn(active, 'getBoundingClientRect').mockReturnValue(createRect(...row))
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('what the rows say', () => {
  it('is a list of options, named, one of them selected and one of them in force', () => {
    const rows = mountRows()
    const list = rows.get('[role="listbox"]')
    expect(list.attributes('aria-label')).toBe('Models')
    const options = rows.findAll('[role="option"]')
    expect(options.map((one) => one.attributes('aria-selected'))).toEqual([
      'false',
      'true',
      'false',
    ])
    expect(options.map((one) => one.attributes('aria-current'))).toEqual([
      'true',
      undefined,
      undefined,
    ])
    expect(options.map((one) => one.attributes('id'))).toEqual(['models-0', 'models-1', 'models-2'])
  })

  it('names a group once, over the row that begins it', () => {
    const rows = mountRows()
    expect(rows.findAll('.searchable-list__heading').map((one) => one.text())).toEqual([
      'Claude',
      'Numen',
    ])
  })

  it('returns the key of the row that was pressed or pointed at', async () => {
    const rows = mountRows()
    const options = rows.findAll('[role="option"]')
    await options[2]?.trigger('click')
    await options[0]?.trigger('pointerenter')
    expect(rows.emitted('choose')).toEqual([['c']])
    expect(rows.emitted('hover')).toEqual([['a']])
  })
})

describe('scrolling to the row the keyboard is on', () => {
  it('moves the list up where the row is above the frame', () => {
    const rows = mountRows()
    const list = rows.get('[role="listbox"]').element as HTMLElement
    list.scrollTop = 80
    arrange(list, [100, 200], [70, 90])

    rows.vm.scrollToActive()

    expect(list.scrollTop).toBe(50)
  })

  it('moves the list down where the row is under the frame', () => {
    const rows = mountRows()
    const list = rows.get('[role="listbox"]').element as HTMLElement
    list.scrollTop = 0
    arrange(list, [100, 200], [190, 230])

    rows.vm.scrollToActive()

    expect(list.scrollTop).toBe(30)
  })

  it('leaves the list where the row is in sight', () => {
    const rows = mountRows()
    const list = rows.get('[role="listbox"]').element as HTMLElement
    list.scrollTop = 40
    arrange(list, [100, 200], [120, 140])

    rows.vm.scrollToActive()

    expect(list.scrollTop).toBe(40)
  })

  it('leaves the list where it has nothing to scroll', () => {
    const rows = mountRows()
    const list = rows.get('[role="listbox"]').element as HTMLElement
    list.scrollTop = 0
    arrange(list, [100, 200], [190, 230], false)

    rows.vm.scrollToActive()

    expect(list.scrollTop).toBe(0)
  })

  it('leaves the list where no row is selected', () => {
    const rows = mountRows({ activeIndex: -1 })
    const list = rows.get('[role="listbox"]').element as HTMLElement
    Object.defineProperty(list, 'scrollHeight', { value: 300, configurable: true })
    Object.defineProperty(list, 'clientHeight', { value: 100, configurable: true })
    list.scrollTop = 10

    rows.vm.scrollToActive()

    expect(list.scrollTop).toBe(10)
  })
})
