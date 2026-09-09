import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Book from './Book.vue'
import { chapterOf } from '@/features/book/fixtures/book'
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
  const held = mount(Book, {
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
    const held = mount(Book, {
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
  const held = mount(Book, {
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

/** A document of three runs, the middle of them a spread along from the first. */
const LAID = chapterOf(
  [{ tag: 'p', text: 'Первая строка.' }, { tag: 'p', text: 'Вторая.' }, { tag: 'p', text: 'Третья.' }],
  400,
)

/** What the browser is told about the reading area, which jsdom never says. */
const measured = (area: HTMLElement, wide: number, high: number): void => {
  Object.defineProperty(area, 'clientWidth', { value: wide, configurable: true })
  Object.defineProperty(area, 'clientHeight', { value: high, configurable: true })
  Object.defineProperty(area, 'getBoundingClientRect', {
    value: () => ({ left: 0 }),
    configurable: true,
  })
}

/** What the browser is told about the laid-out text: how far it runs, where each run stands. */
const laid = (paper: HTMLElement, along: number, lefts: readonly number[]): void => {
  Object.defineProperty(paper, 'scrollWidth', { value: along, configurable: true })
  Object.defineProperty(paper, 'getBoundingClientRect', {
    value: () => ({ left: 0 }),
    configurable: true,
  })
  const runs = paper.querySelectorAll<HTMLElement>('[data-offset]')
  runs.forEach((run, at) => {
    Object.defineProperty(run, 'getClientRects', {
      value: () => [{ left: lefts[at] }],
      configurable: true,
    })
  })
}

/** A reader the browser has measured: an area of two columns, three spreads of text. */
const drawn = async () => {
  const held = mount(Book, {
    props: {
      markup: LAID.markup,
      span: LAID.span,
      book: BOOK,
      at: LAID.span.begins,
    },
    attachTo: document.body,
  })
  const area = held.find('.book__area').element as HTMLElement
  area.scrollTo = () => {}
  measured(area, 1100, 600)
  laid(held.find('.book__paper').element as HTMLElement, 3300, [10, 1150, 580])

  ;(held.vm as unknown as { measure(): void }).measure()
  // The columns are measured after the browser has laid them out, which is the
  // frame after the one that asked, and a frame jsdom draws on its own clock.
  await new Promise((wake) => setTimeout(wake, 50))
  await new Promise((wake) => setTimeout(wake, 50))
  await held.vm.$nextTick()
  return held
}

/** How far the text is carried sideways, as the reader set it last. */
const carried = (held: Awaited<ReturnType<typeof drawn>>): string =>
  (held.find('.book__paper').element as HTMLElement).style.translate

describe('a document the browser has laid out', () => {
  it('is drawn, and says how many spreads it is read in', async () => {
    const held = await drawn()

    expect((held.find('.book__paper').element as HTMLElement).style.display).not.toBe('none')
    expect(held.find('.book__count').text()).toMatch(/^\d+ of \d+$/)
    expect(held.find('.book__left').text()).toMatch(/left in chapter$/)

    held.unmount()
  })

  it('sets the columns against the room it was measured', async () => {
    const held = await drawn()

    const style = (held.find('.book__paper').element as HTMLElement).style
    expect(style.getPropertyValue('--book-column')).toBe('478px')
    expect(style.getPropertyValue('--book-high')).toBe('600px')

    held.unmount()
  })

  it('turns a spread at a time while spreads are left', async () => {
    const held = await drawn()

    expect(keyed(held, 'ArrowRight')).toBe(true)

    expect(carried(held)).toBe('-1100px 0')
    expect(asked(held)).toEqual([LAID.span.begins + bytesIn('Первая строка.')])

    held.unmount()
  })

  it('is turned by a press near either edge, and by the edges alone', async () => {
    const held = await drawn()
    const area = held.find('.book__area').element as HTMLElement
    const press = (at: number) => {
      area.dispatchEvent(new MouseEvent('pointerdown', { button: 0, clientX: at }))
      area.dispatchEvent(new MouseEvent('pointerup', { button: 0, clientX: at }))
    }

    press(1090)
    expect(carried(held)).toBe('-1100px 0')

    press(1090)
    expect(carried(held)).toBe('-2200px 0')

    press(20)
    expect(carried(held)).toBe('-1100px 0')

    press(550)
    expect(carried(held)).toBe('-1100px 0')

    held.unmount()
  })

  it('is stood where an offset from outside asks, while the offset is inside it', async () => {
    const held = await drawn()

    await held.setProps({ at: LAID.span.begins + bytesIn('Первая строка.Вторая.') })
    await held.vm.$nextTick()

    expect(carried(held)).toBe('0px 0')

    held.unmount()
  })
})
