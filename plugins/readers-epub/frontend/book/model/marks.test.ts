/**
 * Where each run of the rendered document is located, and which runs are
 * marked.
 *
 * The browser's highlight registry is replaced by a stub, so a test reads which
 * ranges were added to which entry for which book.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, reactive, shallowRef } from 'vue'

import type { TextRange } from '@numen/ui'
import type { SettledBookProps } from '../lib/props'
import { HIGHLIGHT, OTHER_HIGHLIGHT } from './highlight'
import { useBookMarks } from './marks'

/** What the registry holds for an entry: the ranges of every book that marked it. */
class FakeHighlight {
  readonly ranges: readonly Range[]
  constructor(...ranges: Range[]) {
    this.ranges = ranges
  }
}

const registry = new Map<string, FakeHighlight>()

const getMarked = (entry: string): string[] =>
  (registry.get(entry)?.ranges ?? []).map((range) => range.toString())

const MARKUP = '<p><span data-offset="0">abcdef</span><span data-offset="6">ghijkl</span></p>'

const createPaper = (markup: string, left: readonly number[] = []): HTMLElement => {
  const paper = document.createElement('div')
  paper.innerHTML = markup
  document.body.append(paper)
  paper.querySelectorAll<HTMLElement>('[data-offset]').forEach((run, i) => {
    const pos = left[i] ?? 0
    run.getClientRects = () => [{ left: pos }] as unknown as DOMRectList
  })
  return paper
}

interface Unmountable {
  unmount(): void
}

const mounted = new Set<Unmountable>()

const unmount = (wrapper: Unmountable) => {
  mounted.delete(wrapper)
  wrapper.unmount()
}

const setUp = (
  paper: HTMLElement | null,
  highlights: TextRange[] = [],
  others: TextRange[] = [],
) => {
  const state = reactive({ highlights, otherHighlights: others })
  const props = state as unknown as SettledBookProps
  let marks!: ReturnType<typeof useBookMarks>
  const wrapper = mount(
    defineComponent({
      setup() {
        marks = useBookMarks(shallowRef(paper), props, () => 100)
        return () => null
      },
    }),
  )
  mounted.add(wrapper)
  return { props, state, wrapper, marks }
}

beforeEach(() => {
  vi.stubGlobal('CSS', { highlights: registry })
  vi.stubGlobal('Highlight', FakeHighlight)
})

afterEach(() => {
  for (const wrapper of mounted) wrapper.unmount()
  mounted.clear()
  registry.clear()
  vi.unstubAllGlobals()
  document.body.replaceChildren()
})

describe('the marks of a document', () => {
  it('are none before the runs are gathered', () => {
    expect(setUp(createPaper(MARKUP)).marks.marks.value).toStrictEqual([])
  })

  it('stand where each run is, measured from the near edge', () => {
    const { marks } = setUp(createPaper(MARKUP, [130, 480]))

    marks.gather()

    expect(marks.marks.value).toStrictEqual([
      { at: 0, x: 30 },
      { at: 6, x: 380 },
    ])
  })

  it('are left as they were where no document is drawn', () => {
    const { marks } = setUp(null)

    expect(() => marks.gather()).not.toThrow()
    expect(marks.marks.value).toStrictEqual([])
  })
})

describe('the place a name remains at', () => {
  it('is that of the run holding it once the runs are gathered', () => {
    const { marks } = setUp(
      createPaper(
        '<p><span data-offset="0">abc</span></p><h2 id="part"></h2><p><span data-offset="9">d</span></p>',
      ),
    )

    marks.gather()

    expect(marks.findFragmentOffset('part')).toBe(9)
  })

  it('is nothing before the runs are gathered, or where no document is drawn', () => {
    const { marks } = setUp(
      createPaper(
        '<p><span data-offset="0">abc</span></p><h2 id="part"></h2><p><span data-offset="9">d</span></p>',
      ),
    )

    expect(marks.findFragmentOffset('part')).toBeUndefined()
    expect(setUp(null).marks.findFragmentOffset('part')).toBeUndefined()
  })
})

describe('the runs marked', () => {
  it('are none before the runs are gathered', () => {
    const { marks } = setUp(createPaper(MARKUP), [{ start: 1, end: 3 }])

    marks.markRuns()

    expect(getMarked(HIGHLIGHT)).toStrictEqual([])
    expect(getMarked(OTHER_HIGHLIGHT)).toStrictEqual([])
  })

  it('put the highlights and the others each into their own entry', () => {
    const { marks } = setUp(createPaper(MARKUP), [{ start: 1, end: 3 }], [{ start: 7, end: 10 }])
    marks.gather()

    marks.markRuns()

    expect(getMarked(HIGHLIGHT)).toStrictEqual(['bc'])
    expect(getMarked(OTHER_HIGHLIGHT)).toStrictEqual(['hij'])
  })

  it('are marked again when the highlights change', async () => {
    const { marks, state } = setUp(createPaper(MARKUP))
    marks.gather()

    state.highlights = [{ start: 0, end: 2 }]
    await nextTick()

    expect(getMarked(HIGHLIGHT)).toStrictEqual(['ab'])
    expect(getMarked(OTHER_HIGHLIGHT)).toStrictEqual([])
  })

  it('are marked again when the other highlights change', async () => {
    const { marks, state } = setUp(createPaper(MARKUP))
    marks.gather()

    state.otherHighlights = [{ start: 6, end: 8 }]
    await nextTick()

    expect(getMarked(HIGHLIGHT)).toStrictEqual([])
    expect(getMarked(OTHER_HIGHLIGHT)).toStrictEqual(['gh'])
  })

  it('are removed when the book is disposed', () => {
    const { marks, wrapper } = setUp(createPaper(MARKUP), [{ start: 1, end: 3 }])
    marks.gather()
    marks.markRuns()
    expect(getMarked(HIGHLIGHT)).toStrictEqual(['bc'])

    unmount(wrapper)

    expect(getMarked(HIGHLIGHT)).toStrictEqual([])
  })
})

describe('the runs of a copy of the paper marked', () => {
  it('put the highlights over the runs of the copy', () => {
    const { marks } = setUp(createPaper(MARKUP), [{ start: 1, end: 3 }], [{ start: 7, end: 10 }])
    const copy = createPaper(MARKUP)

    marks.markCopy(copy)

    expect(getMarked(HIGHLIGHT)).toStrictEqual(['bc'])
    expect(getMarked(OTHER_HIGHLIGHT)).toStrictEqual(['hij'])
    for (const entry of [HIGHLIGHT, OTHER_HIGHLIGHT]) {
      for (const range of registry.get(entry)?.ranges ?? []) {
        expect(copy.contains(range.commonAncestorContainer)).toBe(true)
      }
    }
  })

  it('are removed, and only those of the copy, when the copy is unmarked twice', () => {
    const { marks } = setUp(createPaper(MARKUP), [{ start: 1, end: 3 }])
    marks.gather()
    marks.markRuns()
    const copy = createPaper(MARKUP)
    marks.markCopy(copy)
    expect(getMarked(HIGHLIGHT)).toStrictEqual(['bc', 'bc'])

    marks.unmarkCopy(copy)
    marks.unmarkCopy(copy)

    expect(getMarked(HIGHLIGHT)).toStrictEqual(['bc'])
  })

  it('are removed when the component is unmounted', () => {
    const { wrapper, marks } = setUp(createPaper(MARKUP), [{ start: 1, end: 3 }])
    const copy = createPaper(MARKUP)
    marks.markCopy(copy)
    expect(getMarked(HIGHLIGHT)).toStrictEqual(['bc'])

    unmount(wrapper)

    expect(getMarked(HIGHLIGHT)).toStrictEqual([])
  })
})
