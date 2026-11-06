/**
 * Stretches of the book at the edge of a drawn document whose first run begins past the start of
 * the book. The text is Cyrillic, two bytes a letter.
 */
import { describe, expect, it } from 'vitest'
import { getRuns, rangesOver } from './runs'

const DOCUMENT =
  '<h2><span data-offset="40">Один</span><span data-offset="48">\n</span></h2>' +
  '<p><span data-offset="49">два</span></p>'

const createMarkupPaper = (markup: string): HTMLElement => {
  const paper = document.createElement('div')
  paper.innerHTML = markup
  document.body.append(paper)
  return paper
}

describe('a stretch at the start of a document drawn from the middle of the book', () => {
  it('lights nothing where it ends exactly where the document begins', () => {
    const paper = createMarkupPaper(DOCUMENT)

    expect(rangesOver(getRuns(paper), [{ start: 10, end: 40 }])).toStrictEqual([])
  })

  it('lights the first run where its start and end name the offset of that run', () => {
    const paper = createMarkupPaper(DOCUMENT)

    const [range] = rangesOver(getRuns(paper), [{ start: 40, end: 40 }])

    expect(range?.toString()).toBe('Один')
  })
})
