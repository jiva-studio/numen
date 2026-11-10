/**
 * The columns one document is set in: whether the area has been measured, what
 * the columns are set with, and how tall a column is once the text is drawn.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, shallowRef } from 'vue'

import { useBookColumns } from './columns'
import { GAP } from '../lib/spread'
import { BOOK_WORDS } from '../lib/words'
import type { SettledBookProps } from '../lib/props'

const props = (textSize: number): SettledBookProps => ({
  markup: '',
  path: '',
  span: { start: 0, end: 0 },
  book: { start: 0, end: 0 },
  at: 0,
  target: null,
  highlights: [],
  otherHighlights: [],
  textSize,
  chapter: '',
  words: BOOK_WORDS,
})

const createArea = (width: number, height: number): HTMLElement => {
  const area = document.createElement('div')
  Object.defineProperty(area, 'clientWidth', { value: width })
  Object.defineProperty(area, 'clientHeight', { value: height })
  return area
}

const mountColumns = (area: HTMLElement, paper: HTMLElement, textSize: number) => {
  let columns!: ReturnType<typeof useBookColumns>
  mount(
    defineComponent({
      setup() {
        columns = useBookColumns({
          area: shallowRef(area),
          paper: shallowRef(paper),
          props: props(textSize),
          getEdge: () => 0,
        })
        return () => null
      },
    }),
  )
  return columns
}

const setUp = (width: number, height: number, textSize = 1) =>
  mountColumns(createArea(width, height), document.createElement('div'), textSize)

describe('whether the reading area has been measured', () => {
  it('is so once it has a width and a height', () => {
    expect(setUp(800, 600).sizing.isMeasured.value).toBe(true)
  })

  it('is not, before it has either', () => {
    const columns = setUp(0, 0)

    expect(columns.sizing.isMeasured.value).toBe(false)
    columns.sizing.viewport.value = { width: 0, height: 600 }
    expect(columns.sizing.isMeasured.value).toBe(false)
    columns.sizing.viewport.value = { width: 800, height: 0 }
    expect(columns.sizing.isMeasured.value).toBe(false)
  })
})

describe('what the columns are set with', () => {
  it('is one column twice the width of the area, in a narrow one', () => {
    const columns = setUp(420, 600)

    expect(columns.sizing.setting.value['--book-run']).toBe('200%')
    expect(columns.sizing.setting.value['--book-column']).toBe('348px')
  })

  it('is two columns the width of the area, in a wide one', () => {
    const columns = setUp(1200, 600)

    expect(columns.sizing.setting.value['--book-run']).toBe('100%')
    expect(columns.sizing.setting.value['--book-column']).toBe('528px')
  })

  it('is the gap, the height of the area and the size of the text', () => {
    const columns = setUp(420, 600, 1.5)

    expect(columns.sizing.setting.value['--book-gap']).toBe(`${GAP}px`)
    expect(columns.sizing.setting.value['--book-height']).toBe('600px')
    expect(columns.sizing.setting.value['--book-size']).toBe('calc(var(--numen-prose-size) * 1.5)')
  })
})

describe('how tall a column is once the text is drawn', () => {
  /** The box the columns stand in, and the text set in it. */
  const createText = (lineHeight: string, paragraph?: string): HTMLElement => {
    const text = document.createElement('div')
    text.setAttribute('style', `line-height: ${lineHeight}`)
    if (paragraph) {
      const run = document.createElement('p')
      run.setAttribute('style', `line-height: ${paragraph}`)
      text.append(run)
    }
    return text
  }

  const readItem = (box: HTMLElement, text: HTMLElement): number =>
    setUp(800, 600).readColumns(box, text)

  it('is the whole lines of a paragraph that fit the box', () => {
    expect(readItem(createArea(800, 100), createText('10px', '30px'))).toBe(90)
  })

  it('is the whole lines of the text itself where it holds no paragraph', () => {
    expect(readItem(createArea(800, 100), createText('30px'))).toBe(90)
  })

  it('is the height of the box where the lines have no height to be read', () => {
    expect(readItem(createArea(800, 100), createText('normal'))).toBe(100)
  })

  it('keeps the single-column width at the box width when only the first column has text', () => {
    const area = createArea(420, 600)
    const paper = document.createElement('div')
    const run = document.createElement('p')
    run.dataset['offset'] = '0'
    Object.defineProperty(run, 'getClientRects', { value: () => [{ left: 0 }] })
    paper.append(run)
    const columns = mountColumns(area, paper, 1)
    const box = createArea(420, 600)
    const text = createText('20px')
    Object.defineProperty(text, 'clientWidth', { value: 840 })
    Object.defineProperty(text, 'scrollWidth', { value: 840 })

    columns.readColumns(box, text)

    expect(columns.laid.flow.value.contentWidth).toBe(420)
    expect(columns.laid.spreadCount.value).toBe(1)
  })
})

describe('the flow the marks were read in', () => {
  /** Text whose scroll width is twice its client width. */
  const createLaidText = (scrollWidth: number): HTMLElement => {
    const text = document.createElement('div')
    Object.defineProperty(text, 'clientWidth', { value: scrollWidth / 2 })
    Object.defineProperty(text, 'scrollWidth', { value: scrollWidth })
    return text
  }

  it('stays the one of the last reading while the area changes size', () => {
    const columns = setUp(1200, 600)
    columns.readColumns(createArea(1200, 600), createLaidText(4800))

    columns.sizing.viewport.value = { width: 700, height: 600 }

    expect(columns.laid.flow.value).toEqual({
      contentWidth: 4800,
      width: 1200,
      gap: GAP,
      columns: 2,
    })
    expect(columns.laid.spreadCount.value).toBe(4)
    expect(columns.sizing.setting.value['--book-run']).toBe('200%')
  })

  it('becomes the one of the new size once the columns are read again', () => {
    const columns = setUp(1200, 600)
    columns.readColumns(createArea(1200, 600), createLaidText(4800))
    columns.sizing.viewport.value = { width: 700, height: 600 }

    columns.readColumns(createArea(700, 600), createLaidText(5600))

    expect(columns.laid.flow.value).toEqual({
      contentWidth: 5600,
      width: 700,
      gap: GAP,
      columns: 1,
    })
    expect(columns.laid.spreadCount.value).toBe(8)
  })
})
