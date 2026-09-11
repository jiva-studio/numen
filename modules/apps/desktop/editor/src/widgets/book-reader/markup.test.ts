/**
 * One document of a book with its pictures pointed at addresses this window
 * loads them from.
 */
import { describe, expect, it } from 'vitest'
import spine from '../../../../../libs/protocol/testdata/spine.html?raw'
import { pointedAt } from './markup'

/** Where one entry of the archive is served, as the test reads an address. */
const entry = (name: string) =>
  `/assets/book.epub/${name.split('/').map(encodeURIComponent).join('/')}?size=1`

describe('a picture in a book', () => {
  it('is pointed at the address its entry of the archive is served from', () => {
    const drawn = pointedAt('<p><img src="OEBPS/pictures/plate.png" alt="A plate"></p>', entry)

    expect(drawn).toContain('src="/assets/book.epub/OEBPS/pictures/plate.png?size=1"')
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

/**
 * The corpus is one spine document as the application writes it, held to what
 * that writer emits by a test beside it. What is asked here is what this window
 * does to markup that really arrived, rather than to markup it wrote itself.
 */
describe('a document of a book as it arrives', () => {
  it('draws its picture from the entry of the archive the markup names', () => {
    const held = new DOMParser().parseFromString(spine, 'text/html')
    const named = [...held.body.querySelectorAll('img[src]')].map((one) => one.getAttribute('src'))
    expect(named).not.toHaveLength(0)

    const drawn = pointedAt(spine, entry)

    for (const name of named) {
      expect(name).toMatch(/^[^:]+\/[^:]+$/)
      expect(drawn).toContain(`src="${entry(name!)}"`)
    }
  })

  it('leaves every offset the application counted where it stands', () => {
    const offsets = (markup: string) =>
      [...new DOMParser().parseFromString(markup, 'text/html').body.querySelectorAll('[data-offset]')]
        .map((one) => one.getAttribute('data-offset'))

    expect(offsets(spine)).not.toHaveLength(0)
    expect(offsets(pointedAt(spine, entry))).toEqual(offsets(spine))
  })
})
