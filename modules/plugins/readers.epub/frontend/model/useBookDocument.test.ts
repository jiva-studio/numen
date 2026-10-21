/**
 * The document of the spine being drawn, and the markup it is drawn from.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useBookDocument, type MessageWriter } from './useBookDocument'
import type { Books, SpineDocument } from '../types'

const FIRST: SpineDocument = { path: 'text/one.xhtml', span: { from: 0, to: 100 } }
const SECOND: SpineDocument = { path: 'text/two.xhtml', span: { from: 100, to: 200 } }

type Pending = { resolve: (markup: string) => void; reject: (reason: Error) => void }

const create = (open = { value: true }) => {
  const pending = new Map<string, Pending>()
  const urls: string[] = []
  const content: Array<[string, string | undefined]> = []
  const books: Books = {
    getBook: vi.fn(),
    readMarkup: (_path, document) =>
      new Promise<string>((resolve, reject) => pending.set(document, { resolve, reject })),
    getEntryUrl: (path, name, fingerprint) => {
      urls.push(`${path}|${name}|${fingerprint}`)
      return `/served/${name}`
    },
  }
  const writeMessage: MessageWriter = (text, kind) => content.push([text, kind])
  const target = useBookDocument(books, 'library/b.epub', {
    fingerprint: ref('print-1'),
    writeMessage,
    isOpen: () => open.value,
  })
  return { held: target, pending, urls, written: content }
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

describe('a book with nothing drawn', () => {
  it('reads nowhere and names no document', () => {
    const { held } = create()

    expect(held.reading.value).toStrictEqual({ from: 0, to: 0 })
    expect(held.drawn.value).toBe('')
    expect(held.markup.value).toBe('')
  })
})

describe('a document drawn', () => {
  it('remains as the one being read, its pictures pointed where the book serves them', async () => {
    const { held, pending, urls } = create()

    const drawing = held.draw(FIRST)
    pending.get(FIRST.path)?.resolve('<img src="pics/a.png">')
    await drawing

    expect(held.drawn.value).toBe(FIRST.path)
    expect(held.reading.value).toStrictEqual(FIRST.span)
    expect(held.markup.value).toContain('src="/served/pics/a.png"')
    expect(urls).toStrictEqual(['library/b.epub|pics/a.png|print-1'])
  })

  it('is dropped when a newer document was requested for meanwhile', async () => {
    const { held, pending } = create()

    const older = held.draw(FIRST)
    const newer = held.draw(SECOND)
    pending.get(SECOND.path)?.resolve('<p>second</p>')
    await newer
    pending.get(FIRST.path)?.resolve('<p>first</p>')
    await older

    expect(held.drawn.value).toBe(SECOND.path)
    expect(held.markup.value).toContain('second')
  })

  it('is dropped once the tab is closed', async () => {
    const open = { value: true }
    const { held, pending } = create(open)

    const drawing = held.draw(FIRST)
    open.value = false
    pending.get(FIRST.path)?.resolve('<p>first</p>')
    await drawing

    expect(held.drawn.value).toBe('')
    expect(held.markup.value).toBe('')
  })
})

describe('a document that will not be read', () => {
  it('is returned as an error', async () => {
    const { held, pending, messages } = create()

    const drawing = held.draw(FIRST)
    pending.get(FIRST.path)?.reject(new Error('gone'))
    await drawing

    expect(messages).toHaveLength(1)
    expect(messages[0]?.[1]).toBe('error')
  })

  it('is not returned when a newer document was requested for meanwhile', async () => {
    const { held, pending, messages } = create()

    const older = held.draw(FIRST)
    void held.draw(SECOND)
    pending.get(FIRST.path)?.reject(new Error('gone'))
    await older
    await settle()

    expect(messages).toStrictEqual([])
  })

  it('is not returned once the tab is closed', async () => {
    const open = { value: true }
    const { held, pending, messages } = create(open)

    const drawing = held.draw(FIRST)
    open.value = false
    pending.get(FIRST.path)?.reject(new Error('gone'))
    await drawing

    expect(messages).toStrictEqual([])
  })
})
