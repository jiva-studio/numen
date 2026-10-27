/**
 * The document of the spine being drawn, and the markup it is drawn from.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useBookDocument, type MessageWriter } from './useBookDocument'
import type { Books, SpineDocument } from '../types'

const FIRST: SpineDocument = { path: 'text/one.xhtml', span: { start: 0, end: 100 } }
const SECOND: SpineDocument = { path: 'text/two.xhtml', span: { start: 100, end: 200 } }

type Pending = { resolve: (markup: string) => void; reject: (reason: Error) => void }

const create = (open = { value: true }) => {
  const pending = new Map<string, Pending>()
  const urls: string[] = []
  const content: Array<[string, string | undefined]> = []
  const books: Books = {
    getBook: vi.fn(),
    getCfi: vi.fn(),
    findOffset: vi.fn(),
    readMarkup: (_path, document) =>
      new Promise<string>((resolve, reject) => pending.set(document, { resolve, reject })),
    getEntryUrl: (path, name, stat) => {
      urls.push(`${path}|${name}|${stat}`)
      return `/served/${name}`
    },
  }
  const writeMessage: MessageWriter = (text, kind) => content.push([text, kind])
  const target = useBookDocument(books, 'library/b.epub', {
    stat: ref('print-1'),
    writeMessage,
    isOpen: () => open.value,
  })
  return { doc: target, pending, urls, messages: content }
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

describe('a book with nothing drawn', () => {
  it('reads nowhere and names no document', () => {
    const { doc } = create()

    expect(doc.documentSpan.value).toStrictEqual({ start: 0, end: 0 })
    expect(doc.renderedChapter.value).toBe('')
    expect(doc.markup.value).toBe('')
  })
})

describe('a document drawn', () => {
  it('remains as the one being read, its pictures pointed where the book serves them', async () => {
    const { doc, pending, urls } = create()

    const drawTask = doc.draw(FIRST)
    pending.get(FIRST.path)?.resolve('<img src="pics/a.png">')
    await drawTask

    expect(doc.renderedChapter.value).toBe(FIRST.path)
    expect(doc.documentSpan.value).toStrictEqual(FIRST.span)
    expect(doc.markup.value).toContain('src="/served/pics/a.png"')
    expect(urls).toStrictEqual(['library/b.epub|pics/a.png|print-1'])
  })

  it('is dropped when a newer document was requested for meanwhile', async () => {
    const { doc, pending } = create()

    const older = doc.draw(FIRST)
    const newer = doc.draw(SECOND)
    pending.get(SECOND.path)?.resolve('<p>second</p>')
    await newer
    pending.get(FIRST.path)?.resolve('<p>first</p>')
    await older

    expect(doc.renderedChapter.value).toBe(SECOND.path)
    expect(doc.markup.value).toContain('second')
  })

  it('is dropped once the tab is closed', async () => {
    const open = { value: true }
    const { doc, pending } = create(open)

    const drawTask = doc.draw(FIRST)
    open.value = false
    pending.get(FIRST.path)?.resolve('<p>first</p>')
    await drawTask

    expect(doc.renderedChapter.value).toBe('')
    expect(doc.markup.value).toBe('')
  })
})

describe('a document that will not be read', () => {
  it('is returned as an error', async () => {
    const { doc, pending, messages } = create()

    const drawTask = doc.draw(FIRST)
    pending.get(FIRST.path)?.reject(new Error('gone'))
    await drawTask

    expect(messages).toHaveLength(1)
    expect(messages[0]?.[1]).toBe('error')
  })

  it('is not returned when a newer document was requested for meanwhile', async () => {
    const { doc, pending, messages } = create()

    const older = doc.draw(FIRST)
    void doc.draw(SECOND)
    pending.get(FIRST.path)?.reject(new Error('gone'))
    await older
    await settle()

    expect(messages).toStrictEqual([])
  })

  it('is not returned once the tab is closed', async () => {
    const open = { value: true }
    const { doc, pending, messages } = create(open)

    const drawTask = doc.draw(FIRST)
    open.value = false
    pending.get(FIRST.path)?.reject(new Error('gone'))
    await drawTask

    expect(messages).toStrictEqual([])
  })
})
