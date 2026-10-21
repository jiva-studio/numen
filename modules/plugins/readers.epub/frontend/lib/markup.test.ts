/**
 * One document of a book with its pictures pointed at addresses this window
 * loads them from.
 */
import { describe, expect, it } from 'vitest'
import spine from '../../../../libs/protocol/testdata/spine.html?raw'
import { resolveImageUrls } from './markup'

const entry = (name: string) =>
  `/assets/book.epub/${name.split('/').map(encodeURIComponent).join('/')}?size=1`

describe('a picture in a book', () => {
  it('is pointed at the address its entry of the archive is served from', () => {
    const rendered = resolveImageUrls(
      '<p><img src="OEBPS/pictures/plate.png" alt="A plate"></p>',
      entry,
    )

    expect(rendered).toContain('src="/assets/book.epub/OEBPS/pictures/plate.png?size=1"')
    expect(rendered).toContain('alt="A plate"')
  })

  it('is left at its current position when it is written into the markup itself', () => {
    const content = '<img src="data:image/png;base64,iVBORw0KGgo=">'

    expect(resolveImageUrls(content, entry)).toContain('src="data:image/png;base64,iVBORw0KGgo="')
  })
})

describe('a picture with no name', () => {
  it('is left currently and asks for no address', () => {
    const request: string[] = []

    const rendered = resolveImageUrls('<img src="" alt="Blank">', (name) => {
      request.push(name)
      return entry(name)
    })

    expect(request).toStrictEqual([])
    expect(rendered).toContain('src=""')
  })
})

describe('the runs of the text', () => {
  it('carry the offsets they arrived with', () => {
    const rendered = resolveImageUrls(
      '<span data-offset="1200">सत्यं</span><span data-offset="1215">Слово</span>',
      entry,
    )

    expect(rendered).toContain('data-offset="1200"')
    expect(rendered).toContain('data-offset="1215"')
    expect(rendered).toContain('सत्यं')
  })
})

describe('a document of a book as it arrives', () => {
  it('draws its picture from the entry of the archive the markup names', () => {
    const target = new DOMParser().parseFromString(spine, 'text/html')
    const named = [...target.body.querySelectorAll('img[src]')].map((one) => one.getAttribute('src'))
    expect(named).not.toHaveLength(0)

    const rendered = resolveImageUrls(spine, entry)

    for (const name of named) {
      expect(name).toMatch(/^[^:]+\/[^:]+$/)
      expect(rendered).toContain(`src="${entry(name!)}"`)
    }
  })

  it('leaves every offset the application counted at its current position', () => {
    const offsets = (markup: string) =>
      [
        ...new DOMParser()
          .parseFromString(markup, 'text/html')
          .body.querySelectorAll('[data-offset]'),
      ].map((one) => one.getAttribute('data-offset'))

    expect(offsets(spine)).not.toHaveLength(0)
    expect(offsets(resolveImageUrls(spine, entry))).toEqual(offsets(spine))
  })
})
