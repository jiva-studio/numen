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
  span: { from: 0, to: 0 },
  book: { from: 0, to: 0 },
  at: 0,
  highlights: [],
  otherHighlights: [],
  textSize,
  chapter: '',
  words: BOOK_WORDS,
})

/** An element the browser has measured at a size, which jsdom never does by itself. */
const createArea = (width: number, height: number): HTMLElement => {
  const area = document.createElement('div')
  Object.defineProperty(area, 'clientWidth', { value: width })
  Object.defineProperty(area, 'clientHeight', { value: height })
  return area
}

/** The columns of a book mounted over an area of a size. */
const setUp = (width: number, height: number, textSize = 1) => {
  const area = createArea(width, height)
  const paper = document.createElement('div')
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

describe('whether the reading area has been measured', () => {
  it('is so once it has a width and a height', () => {
    expect(setUp(800, 600).isMeasured.value).toBe(true)
  })

  it('is not, before it has either', () => {
    const columns = setUp(0, 0)

    expect(columns.isMeasured.value).toBe(false)
    columns.viewport.value = { width: 0, height: 600 }
    expect(columns.isMeasured.value).toBe(false)
    columns.viewport.value = { width: 800, height: 0 }
    expect(columns.isMeasured.value).toBe(false)
  })
})

describe('what the columns are set with', () => {
  it('is one column twice the width of the area, in a narrow one', () => {
    const columns = setUp(420, 600)

    expect(columns.setting.value['--book-run']).toBe('200%')
    expect(columns.setting.value['--book-column']).toBe('348px')
  })

  it('is two columns the width of the area, in a wide one', () => {
    const columns = setUp(1200, 600)

    expect(columns.setting.value['--book-run']).toBe('100%')
    expect(columns.setting.value['--book-column']).toBe('528px')
  })

  it('is the gap, the height of the area and the size of the text', () => {
    const columns = setUp(420, 600, 1.5)

    expect(columns.setting.value['--book-gap']).toBe(`${GAP}px`)
    expect(columns.setting.value['--book-height']).toBe('600px')
    expect(columns.setting.value['--book-size']).toBe('calc(var(--numen-prose-size) * 1.5)')
  })
})

describe('how tall a column is once the text is drawn', () => {
  /** The box the columns stand in, and the text set in it. */
  const createText = (lineHeight: string, paragraph?: string): HTMLElement => {
    const text = document.createElement('div')
    text.style.lineHeight = lineHeight
    if (paragraph) {
      const run = document.createElement('p')
      run.style.lineHeight = paragraph
      text.append(run)
    }
    return text
  }

  const readItem = (box: HTMLElement, text: HTMLElement): string => {
    setUp(800, 600).readColumns(box, text)
    return text.style.getPropertyValue('--book-paper')
  }

  it('is the whole lines of a paragraph that fit the box', () => {
    expect(readItem(createArea(800, 100), createText('10px', '30px'))).toBe('90px')
  })

  it('is the whole lines of the text itself where it holds no paragraph', () => {
    expect(readItem(createArea(800, 100), createText('30px'))).toBe('90px')
  })

  it('is the height of the box where the lines have no height to be read', () => {
    expect(readItem(createArea(800, 100), createText('normal'))).toBe('100px')
  })
})
