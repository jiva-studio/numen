import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import BookReader from './BookReader.vue'
import { chapterOf } from '@/fixtures/book'
import { bytesIn } from './spread'

/**
 * A chapter standing in the middle of a book, with a document before it and a
 * document after it.
 */
const CHAPTER = chapterOf([{ tag: 'p', text: 'Один абзац, и глава окончена.' }], 400)
const BOOK = { begins: 0, ends: CHAPTER.span.ends + 500 }

/**
 * A book reader in a room nothing has laid out. jsdom measures nothing, so the
 * text comes to no spreads at all and what is asked here is what the reader
 * does when it has none.
 */
const reader = async (book = BOOK) => {
  const held = mount(BookReader, {
    props: { markup: CHAPTER.markup, span: CHAPTER.span, book, at: CHAPTER.span.begins },
    attachTo: document.body,
  })
  const area = held.find('.book__area').element as HTMLElement
  area.scrollTo = () => {}
  await held.vm.$nextTick()
  return held
}

/** The offsets the reader has asked to be sent to, in the order it asked. */
const asked = (held: Awaited<ReturnType<typeof reader>>) =>
  (held.emitted('go') ?? []).map((one) => (one as [number])[0])

/**
 * A key the tab caught and handed down, answered with whether it turned the
 * page. The tab listens; the reader is asked.
 */
const keyed = (held: Awaited<ReturnType<typeof reader>>, key: string): boolean =>
  (held.vm as unknown as { pressed(event: KeyboardEvent): boolean }).pressed(
    new KeyboardEvent('keydown', { key }),
  )

describe('a document nothing has laid out', () => {
  it('draws no controls, because it has come to no spreads', async () => {
    const held = await reader()

    expect(held.find('input[type="number"]').exists()).toBe(false)
  })

  it('says nothing about what is in front', async () => {
    // Where a run of the text stands is what the browser laid out, and a
    // reading area nothing has measured has no run in front of anybody.
    const held = await reader()

    expect(asked(held)).toHaveLength(0)
  })

  it('shows nothing at all', async () => {
    // A book is turned and never scrolled, so its text is drawn only against an
    // area of a known size.
    const held = await reader()

    expect((held.find('.book__paper').element as HTMLElement).style.display).toBe('none')
  })
})

describe('turning past the end of a document', () => {
  it('asks for the offset the next document begins at', async () => {
    const held = await reader()

    expect(keyed(held, 'ArrowRight')).toBe(true)

    expect(asked(held)).toEqual([CHAPTER.span.ends])
  })

  it('asks for the offset before this document, turning back', async () => {
    const held = await reader()

    expect(keyed(held, 'ArrowLeft')).toBe(true)

    expect(asked(held)).toEqual([CHAPTER.span.begins - 1])
  })

  it('asks for nothing past either end of the book itself', async () => {
    const held = await reader(CHAPTER.span)

    keyed(held, 'ArrowRight')
    keyed(held, 'ArrowLeft')

    expect(asked(held)).toHaveLength(0)
  })

  it('turns nothing on a key a book is not read with, and says so', async () => {
    // The tab hands every key down and stops only the ones the book took, so a
    // key the book does not read is left to whatever else is listening.
    const held = await reader()

    expect(keyed(held, 'Enter')).toBe(false)
    expect(keyed(held, 'a')).toBe(false)

    expect(asked(held)).toHaveLength(0)
  })
})

describe('a document with no text at all', () => {
  it('is drawn, and says nothing', async () => {
    const held = mount(BookReader, {
      props: { markup: '', span: { begins: 0, ends: 0 }, book: BOOK },
      attachTo: document.body,
    })
    await held.vm.$nextTick()

    expect(held.find('.book__paper').exists()).toBe(true)
    expect(held.findAll('[data-offset]')).toHaveLength(0)
    expect(asked(held)).toHaveLength(0)
  })
})

/** A document whose text points inside the book and once out of it. */
const POINTING = chapterOf(
  [
    {
      tag: 'p',
      text: 'Onward: ',
      inside: '<a href="OEBPS/second.xhtml#alpha">the second parva</a>',
    },
    { tag: 'p', text: 'Down: ', inside: '<a href="#beta">the note</a>' },
    { tag: 'p', text: 'Away: ', inside: '<a href="https://example.invalid/away">elsewhere</a>' },
    { tag: 'p', text: 'The note pointed down to.', inside: '<span id="beta"></span>' },
  ],
  400,
)

/** A reader drawing that document, and the press a link in it was given. */
const pointing = async () => {
  const held = mount(BookReader, {
    props: {
      markup: POINTING.markup,
      path: 'OEBPS/first.xhtml',
      span: POINTING.span,
      book: BOOK,
      at: POINTING.span.begins,
    },
    attachTo: document.body,
  })
  const area = held.find('.book__area').element as HTMLElement
  area.scrollTo = () => {}
  await held.vm.$nextTick()
  return held
}

/** One link of the document pressed, and the press as the page left it. */
const press = async (held: Awaited<ReturnType<typeof pointing>>, says: string) => {
  const link = held.findAll('a').find((one) => one.text() === says)!
  const event = new MouseEvent('click', { bubbles: true, cancelable: true })
  link.element.dispatchEvent(event)
  await held.vm.$nextTick()
  return event
}

describe('a link inside a book', () => {
  it('is taken by the reader, and never reaches the browser', async () => {
    const held = await pointing()

    for (const says of ['the second parva', 'the note', 'elsewhere']) {
      expect((await press(held, says)).defaultPrevented).toBe(true)
    }
  })

  it('asks for the document it names, where that is another of the book', async () => {
    const held = await pointing()

    await press(held, 'the second parva')

    expect(held.emitted('follow')).toEqual([['OEBPS/second.xhtml']])
  })

  it('asks for the offset the place stands at, inside the document being read', async () => {
    const held = await pointing()

    await press(held, 'the note')

    expect(asked(held)).toEqual([POINTING.span.ends - bytesIn('The note pointed down to.')])
  })

  it('asks for nothing of the book, where the link leads out of it', async () => {
    const held = await pointing()

    await press(held, 'elsewhere')

    expect(asked(held)).toHaveLength(0)
    expect(held.emitted('follow')).toBeUndefined()
  })
})
