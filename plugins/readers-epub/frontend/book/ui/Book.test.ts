import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Book from './Book.vue'
import { createChapter } from '../fixtures/book'
import { countBytes } from '../lib/bytes'

/**
 * A chapter standing in the middle of a book, with a document before it and a
 * document after it.
 */
const CHAPTER = createChapter([{ tag: 'p', text: 'Один абзац, и глава окончена.' }], 400)
const BOOK = { start: 0, end: CHAPTER.span.end + 500 }

const reader = async (book = BOOK) => {
  const wrapper = mount(Book, {
    props: { markup: CHAPTER.markup, span: CHAPTER.span, book, at: CHAPTER.span.start },
    attachTo: document.body,
  })
  const area = wrapper.find('.book__area').element as HTMLElement
  area.scrollTo = () => undefined
  await wrapper.vm.$nextTick()
  return wrapper
}

const getMoves = (wrapper: Awaited<ReturnType<typeof reader>>) =>
  (wrapper.emitted('move') ?? []).map((one) => (one as [number])[0])

const pressKey = (wrapper: Awaited<ReturnType<typeof reader>>, key: string): boolean =>
  (wrapper.vm as unknown as { handleKey(event: KeyboardEvent): boolean }).handleKey(
    new KeyboardEvent('keydown', { key }),
  )

describe('a document nothing has laid out', () => {
  it('draws no controls, because it has come to no countSpreads', async () => {
    const target = await reader()

    expect(target.find('input[type="number"]').exists()).toBe(false)
  })

  it('says nothing about what is in front', async () => {
    // The position of a text run comes from the browser layout, and an
    // unmeasured reading area has no run at the current position.
    const target = await reader()

    expect(getMoves(target)).toHaveLength(0)
  })

  it('shows nothing at all', async () => {
    // A book is turned and never scrolled, so its text is drawn only against an
    // area of a known size.
    const target = await reader()

    expect(target.find('.book__paper').isVisible()).toBe(false)
  })
})

describe('turning past the end of a document', () => {
  it('asks for the offset the next document begins at', async () => {
    const target = await reader()

    expect(pressKey(target, 'ArrowRight')).toBe(true)

    expect(getMoves(target)).toEqual([CHAPTER.span.end])
  })

  it('asks for the offset before this document, turning back', async () => {
    const target = await reader()

    expect(pressKey(target, 'ArrowLeft')).toBe(true)

    expect(getMoves(target)).toEqual([CHAPTER.span.start - 1])
  })

  it('asks for nothing past either end of the book itself', async () => {
    const target = await reader(CHAPTER.span)

    pressKey(target, 'ArrowRight')
    pressKey(target, 'ArrowLeft')

    expect(getMoves(target)).toHaveLength(0)
  })

  it('turns nothing on a key a book is not read with, and says so', async () => {
    // The tab hands every key down and stops only the ones the book took, so a
    // key the book does not read is left to whatever else is listening.
    const target = await reader()

    expect(pressKey(target, 'Enter')).toBe(false)
    expect(pressKey(target, 'a')).toBe(false)

    expect(getMoves(target)).toHaveLength(0)
  })
})

describe('a document with no text at all', () => {
  it('is drawn, and says nothing', async () => {
    const wrapper = mount(Book, {
      props: { markup: '', span: { start: 0, end: 0 }, book: BOOK },
      attachTo: document.body,
    })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.book__paper').exists()).toBe(true)
    expect(wrapper.findAll('[data-offset]')).toHaveLength(0)
    expect(getMoves(wrapper)).toHaveLength(0)
  })
})

/** A document whose text points inside the book and once out of it. */
const POINTING = createChapter(
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

const mountPointingBook = async () => {
  const wrapper = mount(Book, {
    props: {
      markup: POINTING.markup,
      path: 'OEBPS/first.xhtml',
      span: POINTING.span,
      book: BOOK,
      at: POINTING.span.start,
    },
    attachTo: document.body,
  })
  const area = wrapper.find('.book__area').element as HTMLElement
  area.scrollTo = () => undefined
  await wrapper.vm.$nextTick()
  return wrapper
}

const press = async (wrapper: Awaited<ReturnType<typeof mountPointingBook>>, label: string) => {
  const link = wrapper.findAll('a').find((one) => one.text() === label)!
  const event = new MouseEvent('click', { bubbles: true, cancelable: true })
  link.element.dispatchEvent(event)
  await wrapper.vm.$nextTick()
  return event
}

describe('a link inside a book', () => {
  it('is taken by the reader, and never reaches the browser', async () => {
    const target = await mountPointingBook()

    for (const says of ['the second parva', 'the note', 'elsewhere']) {
      expect((await press(target, says)).defaultPrevented).toBe(true)
    }
  })

  it('asks for the document it names, where that is another of the book', async () => {
    const target = await mountPointingBook()

    await press(target, 'the second parva')

    expect(target.emitted('follow')).toEqual([['OEBPS/second.xhtml']])
  })

  it('asks for the offset the place remains at, inside the document being read', async () => {
    const target = await mountPointingBook()

    await press(target, 'the note')

    expect(getMoves(target)).toEqual([POINTING.span.end - countBytes('The note pointed down to.')])
  })

  it('asks for nothing of the book, where the link leads out of it', async () => {
    const target = await mountPointingBook()

    await press(target, 'elsewhere')

    expect(getMoves(target)).toHaveLength(0)
    expect(target.emitted('follow')).toBeUndefined()
  })
})

/** A document of three runs, the middle of them a spread along from the first. */
const LAID = createChapter(
  [
    { tag: 'p', text: 'Первая строка.' },
    { tag: 'p', text: 'Вторая.' },
    { tag: 'p', text: 'Третья.' },
  ],
  400,
)

const setSize = (area: HTMLElement, width: number, high: number): void => {
  Object.defineProperty(area, 'clientWidth', { value: width, configurable: true })
  Object.defineProperty(area, 'clientHeight', { value: high, configurable: true })
  Object.defineProperty(area, 'getBoundingClientRect', {
    value: () => ({ left: 0 }),
    configurable: true,
  })
}

const setLayout = (paper: HTMLElement, scrollWidth: number, lefts: readonly number[]): void => {
  Object.defineProperty(paper, 'scrollWidth', { value: scrollWidth, configurable: true })
  Object.defineProperty(paper, 'getBoundingClientRect', {
    value: () => ({ left: 0 }),
    configurable: true,
  })
  const runs = paper.querySelectorAll<HTMLElement>('[data-offset]')
  runs.forEach((run, index) => {
    Object.defineProperty(run, 'getClientRects', {
      value: () => [{ left: lefts[index] }],
      configurable: true,
    })
  })
}

const mountBook = async () => {
  const wrapper = mount(Book, {
    props: {
      markup: LAID.markup,
      span: LAID.span,
      book: BOOK,
      at: LAID.span.start,
    },
    attachTo: document.body,
  })
  const area = wrapper.find('.book__area').element as HTMLElement
  area.scrollTo = () => undefined
  setSize(area, 1100, 600)
  setLayout(wrapper.find('.book__paper').element as HTMLElement, 3300, [10, 1150, 580])

  ;(wrapper.vm as unknown as { measure(): void }).measure()
  // The columns are measured after the browser has laid them out, which is the
  // frame after the requesting one, on a frame clock that jsdom drives itself.
  await new Promise((resolve) => setTimeout(resolve, 50))
  await new Promise((resolve) => setTimeout(resolve, 50))
  await wrapper.vm.$nextTick()
  return wrapper
}

const getFront = (wrapper: Awaited<ReturnType<typeof mountBook>>): string =>
  `${wrapper.find('.book__count').text()} / ${wrapper.find('.book__left').text()}`

describe('a document the browser has laid out', () => {
  it('is drawn, and says how many countSpreads it is read in', async () => {
    const target = await mountBook()

    expect(target.find('.book__paper').isVisible()).toBe(true)
    expect(target.find('.book__count').text()).toMatch(/^\d+ of \d+$/)
    expect(target.find('.book__left').text()).toMatch(/left in chapter$/)

    target.unmount()
  })

  it('turns a spread at a time while countSpreads are left', async () => {
    const target = await mountBook()
    expect(pressKey(target, 'ArrowRight')).toBe(true)

    expect(getMoves(target)).toEqual([LAID.span.start + countBytes('Первая строка.')])

    target.unmount()
  })

  it('is turned by a press near either edge, and by the edges alone', async () => {
    const target = await mountBook()
    const area = target.find('.book__area').element as HTMLElement
    const press = (pos: number) => {
      area.dispatchEvent(new MouseEvent('pointerdown', { button: 0, clientX: pos }))
      area.dispatchEvent(new MouseEvent('pointerup', { button: 0, clientX: pos }))
    }

    press(1090)
    expect(getMoves(target)).toHaveLength(1)

    press(550)
    expect(getMoves(target)).toHaveLength(1)

    target.unmount()
  })

  it('is stood where an offset from outside asks, while the offset is inside it', async () => {
    const target = await mountBook()
    const first = getFront(target)

    await target.setProps({ at: LAID.span.start + countBytes('Первая строка.Вторая.') })
    await target.vm.$nextTick()

    expect(getFront(target)).toBe(first)

    target.unmount()
  })
})

/** A second document of the book, which a turn past the end of the first opens. */
const NEXT = createChapter([{ tag: 'p', text: 'Следующая глава.' }], LAID.span.end)

describe('a document replaced while a later spread of the one before is in front', () => {
  it('stays out of sight until its spread is stood', async () => {
    const target = await mountBook()
    pressKey(target, 'ArrowRight')

    await target.setProps({ markup: NEXT.markup, span: NEXT.span, at: NEXT.span.start })

    // The frame that applies the new document has not run yet, and the text
    // positioned for the old spread is hidden in the meantime.
    expect(target.find('.book__paper').classes()).toContain('book__paper--unset')

    await new Promise((resolve) => setTimeout(resolve, 50))
    await target.vm.$nextTick()

    expect(target.find('.book__paper').classes()).not.toContain('book__paper--unset')

    target.unmount()
  })
})
