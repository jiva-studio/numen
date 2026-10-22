/**
 * Where each run of the drawn document stands, and which of them are marked.
 *
 * The browser's highlight registry is stood in for, so a test reads which
 * ranges were put into which entry for which book.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, reactive, shallowRef } from 'vue'

import type { Span } from '@/shared/lib/span'
import type { SettledBookProps } from '../lib/props'
import { HIGHLIGHT, OTHER_HIGHLIGHT, highlight, unhighlight } from '../lib/highlight'
import { useBookMarks } from './marks'

vi.mock('../lib/highlight', async (original) => ({
  ...(await original<typeof import('../lib/highlight')>()),
  highlight: vi.fn(),
  unhighlight: vi.fn(),
}))

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

const setUp = (paper: HTMLElement | null, highlights: Span[] = [], others: Span[] = []) => {
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
  return { props, state, wrapper, marks }
}

const getText = (call: unknown[]): string[] => (call[2] as Range[]).map((range) => range.toString())

afterEach(() => {
  vi.clearAllMocks()
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
    const { marks } = setUp(createPaper(MARKUP), [{ from: 1, to: 3 }])

    marks.markRuns()

    expect(vi.mocked(highlight).mock.calls.map(getText)).toStrictEqual([[], []])
  })

  it('put the highlights and the others each into their own entry', () => {
    const { marks, props } = setUp(createPaper(MARKUP), [{ from: 1, to: 3 }], [{ from: 7, to: 10 }])
    marks.gather()

    marks.markRuns()

    const calls = vi.mocked(highlight).mock.calls
    expect(calls.map((call) => [call[0], call[1]])).toStrictEqual([
      [HIGHLIGHT, props],
      [OTHER_HIGHLIGHT, props],
    ])
    expect(calls.map(getText)).toStrictEqual([['bc'], ['hij']])
  })

  it('are marked again when the highlights change', async () => {
    const { marks, state } = setUp(createPaper(MARKUP))
    marks.gather()

    state.highlights = [{ from: 0, to: 2 }]
    await nextTick()

    expect(
      vi.mocked(highlight).mock.calls.map((call) => [call[0], ...getText(call)]),
    ).toStrictEqual([[HIGHLIGHT, 'ab'], [OTHER_HIGHLIGHT]])
  })

  it('are marked again when the other highlights change', async () => {
    const { marks, state } = setUp(createPaper(MARKUP))
    marks.gather()

    state.otherHighlights = [{ from: 6, to: 8 }]
    await nextTick()

    expect(
      vi.mocked(highlight).mock.calls.map((call) => [call[0], ...getText(call)]),
    ).toStrictEqual([[HIGHLIGHT], [OTHER_HIGHLIGHT, 'gh']])
  })

  it('are taken back when the book goes', () => {
    const { wrapper, props } = setUp(createPaper(MARKUP))

    wrapper.unmount()

    expect(unhighlight).toHaveBeenCalledExactlyOnceWith(props)
  })
})
