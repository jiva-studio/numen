/**
 * One document of a book with its pictures pointed at addresses this window
 * loads them from.
 */
import { describe, expect, it } from 'vitest'
import spine from '../../../../../../libs/protocol/testdata/spine.html?raw'
import { resolveImageUrls } from './markup'

const entry = (name: string) =>
  `/assets/book.epub/${name.split('/').map(encodeURIComponent).join('/')}?size=1`

describe('a picture in a book', () => {
  it('is pointed at the address its entry of the archive is served from', () => {
    const drawn = resolveImageUrls('<p><img src="OEBPS/pictures/plate.png" alt="A plate"></p>', entry)

    expect(drawn).toContain('src="/assets/book.epub/OEBPS/pictures/plate.png?size=1"')
    expect(drawn).toContain('alt="A plate"')
  })

  it('is left where it stands when it is written into the markup itself', () => {
    const written = '<img src="data:image/png;base64,iVBORw0KGgo=">'

    expect(resolveImageUrls(written, entry)).toContain('src="data:image/png;base64,iVBORw0KGgo="')
  })
})

describe('the runs of the text', () => {
  it('carry the offsets they arrived with', () => {
    const drawn = resolveImageUrls('<span data-offset="1200">सत्यं</span><span data-offset="1215">Слово</span>', entry)

    expect(drawn).toContain('data-offset="1200"')
    expect(drawn).toContain('data-offset="1215"')
    expect(drawn).toContain('सत्यं')
  })
})

describe('a document of a book as it arrives', () => {
  it('draws its picture from the entry of the archive the markup names', () => {
    const held = new DOMParser().parseFromString(spine, 'text/html')
    const named = [...held.body.querySelectorAll('img[src]')].map((one) => one.getAttribute('src'))
    expect(named).not.toHaveLength(0)

    const drawn = resolveImageUrls(spine, entry)

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
    expect(offsets(resolveImageUrls(spine, entry))).toEqual(offsets(spine))
  })
})
