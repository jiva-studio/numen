/**
 * One document of a book with its pictures pointed at addresses this window
 * loads them from.
 */
import { describe, expect, it } from 'vitest'
import { pointedAt } from './markup'

/** Where one entry of the archive is served, as the test reads an address. */
const entry = (name: string) => `/assets/book.epub/entries/${encodeURIComponent(name)}?size=1`

describe('a picture in a book', () => {
  it('is pointed at the address its entry of the archive is served from', () => {
    const drawn = pointedAt('<p><img src="OEBPS/pictures/plate.png" alt="A plate"></p>', entry)

    expect(drawn).toContain('src="/assets/book.epub/entries/OEBPS%2Fpictures%2Fplate.png?size=1"')
    expect(drawn).toContain('alt="A plate"')
  })

  it('is left where it stands when it is written into the markup itself', () => {
    const written = '<img src="data:image/png;base64,iVBORw0KGgo=">'

    expect(pointedAt(written, entry)).toContain('src="data:image/png;base64,iVBORw0KGgo="')
  })
})

describe('the runs of the text', () => {
  it('carry the offsets they arrived with', () => {
    // The offsets are bytes the application counted, and nothing here counts
    // one: a letter of Devanagari is three bytes and one of Cyrillic is two.
    const drawn = pointedAt('<span data-offset="1200">सत्यं</span><span data-offset="1215">Слово</span>', entry)

    expect(drawn).toContain('data-offset="1200"')
    expect(drawn).toContain('data-offset="1215"')
    expect(drawn).toContain('सत्यं')
  })
})
