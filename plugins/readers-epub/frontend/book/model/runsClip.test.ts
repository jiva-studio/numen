/**
 * Spans at the boundary of a rendered document whose first run begins past the start of
 * the book. The text is Cyrillic, two bytes a letter.
 */
import { describe, expect, it } from 'vitest'
import { getRuns, createRangesOver } from './runs'

const DOCUMENT =
  '<h2><span data-offset="40">Один</span><span data-offset="48">\n</span></h2>' +
  '<p><span data-offset="49">два</span></p>'

const createMarkupPaper = (markup: string): HTMLElement => {
  const paper = document.createElement('div')
  paper.innerHTML = markup
  document.body.append(paper)
  return paper
}

describe('a span at the start of a document rendered from the middle of the book', () => {
  it('highlights nothing when it ends exactly where the document begins', () => {
    const paper = createMarkupPaper(DOCUMENT)

    expect(createRangesOver(getRuns(paper), [{ start: 10, end: 40 }])).toStrictEqual([])
  })

  it('highlights the first run when its start and end are the offset of that run', () => {
    const paper = createMarkupPaper(DOCUMENT)

    const [range] = createRangesOver(getRuns(paper), [{ start: 40, end: 40 }])

    expect(range?.toString()).toBe('Один')
  })
})
