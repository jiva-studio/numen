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

/**
 * A book reader in a document with no layout. jsdom measures nothing, so the
 * text has no spread count and the tests cover the reader's behavior without one.
 */
const reader = async (book = BOOK) => {
  const wrapper = mount(Book, {
    props: { markup: CHAPTER.markup, span: CHAPTER.span, book, at: CHAPTER.span.start },
    attachTo: document.body,
  })
  const area = wrapper.find('.book__area').element as HTMLElement
  area.scrollTo = () => {}
  await wrapper.vm.$nextTick()
  return wrapper
}

/** The offsets the reader requested to move to, in request order. */
const getMoves = (wrapper: Awaited<ReturnType<typeof reader>>) =>
  (wrapper.emitted('move') ?? []).map((one) => (one as [number])[0])

/**
 * A key handled by the tab and passed down, returning whether it turned the
 * page. The tab dispatches the key; the reader is queried.
 */
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

    expect((target.find('.book__paper').element as HTMLElement).style.display).toBe('none')
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

/** A reader drawing that document, and the press a link in it was given. */
const mountPointing = async () => {
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
  area.scrollTo = () => {}
  await wrapper.vm.$nextTick()
  return wrapper
}

/** One link of the document pressed, and the press as the page left it. */
const press = async (wrapper: Awaited<ReturnType<typeof mountPointing>>, label: string) => {
  const link = wrapper.findAll('a').find((one) => one.text() === label)!
  const event = new MouseEvent('click', { bubbles: true, cancelable: true })
  link.element.dispatchEvent(event)
  await wrapper.vm.$nextTick()
  return event
}

describe('a link inside a book', () => {
  it('is taken by the reader, and never reaches the browser', async () => {
    const target = await mountPointing()

    for (const says of ['the second parva', 'the note', 'elsewhere']) {
      expect((await press(target, says)).defaultPrevented).toBe(true)
    }
  })

  it('asks for the document it names, where that is another of the book', async () => {
    const target = await mountPointing()

    await press(target, 'the second parva')

    expect(target.emitted('follow')).toEqual([['OEBPS/second.xhtml']])
  })

  it('asks for the offset the place remains at, inside the document being read', async () => {
    const target = await mountPointing()

    await press(target, 'the note')

    expect(getMoves(target)).toEqual([POINTING.span.end - countBytes('The note pointed down to.')])
  })

  it('asks for nothing of the book, where the link leads out of it', async () => {
    const target = await mountPointing()

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

/** Sets the reading area size that a browser would report and jsdom does not. */
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

/** A reader the browser has measured: an area of two columns, three countSpreads of text. */
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
  area.scrollTo = () => {}
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

/** How far the text is carried sideways, as the reader set it last. */
const getTranslate = (wrapper: Awaited<ReturnType<typeof mountBook>>): string =>
  (wrapper.find('.book__paper').element as HTMLElement).style.translate

describe('a document the browser has laid out', () => {
  it('is drawn, and says how many countSpreads it is read in', async () => {
    const target = await mountBook()

    expect((target.find('.book__paper').element as HTMLElement).style.display).not.toBe('none')
    expect(target.find('.book__count').text()).toMatch(/^\d+ of \d+$/)
    expect(target.find('.book__left').text()).toMatch(/left in chapter$/)

    target.unmount()
  })

  it('sets the columns against the room it was measured', async () => {
    const target = await mountBook()

    const style = (target.find('.book__paper').element as HTMLElement).style
    expect(style.getPropertyValue('--book-column')).toBe('478px')
    expect(style.getPropertyValue('--book-height')).toBe('600px')

    target.unmount()
  })

  it('turns a spread at a time while countSpreads are left', async () => {
    const target = await mountBook()

    expect(pressKey(target, 'ArrowRight')).toBe(true)

    expect(getTranslate(target)).toBe('-1100px 0')
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
    expect(getTranslate(target)).toBe('-1100px 0')

    press(1090)
    expect(getTranslate(target)).toBe('-2200px 0')

    press(20)
    expect(getTranslate(target)).toBe('-1100px 0')

    press(550)
    expect(getTranslate(target)).toBe('-1100px 0')

    target.unmount()
  })

  it('is stood where an offset from outside asks, while the offset is inside it', async () => {
    const target = await mountBook()

    await target.setProps({ at: LAID.span.start + countBytes('Первая строка.Вторая.') })
    await target.vm.$nextTick()

    expect(getTranslate(target)).toBe('0px 0')

    target.unmount()
  })
})

/** A second document of the book, which a turn past the end of the first opens. */
const NEXT = createChapter([{ tag: 'p', text: 'Следующая глава.' }], LAID.span.end)

describe('a document replaced while a later spread of the one before is in front', () => {
  it('stays out of sight until its spread is stood', async () => {
    const target = await mountBook()
    pressKey(target, 'ArrowRight')
    expect(getTranslate(target)).toBe('-1100px 0')

    await target.setProps({ markup: NEXT.markup, span: NEXT.span, at: NEXT.span.start })

    // The frame that applies the new document has not run yet, and the text
    // positioned for the old spread is hidden in the meantime.
    expect(target.find('.book__paper').classes()).toContain('book__paper--unset')

    await new Promise((resolve) => setTimeout(resolve, 50))
    await target.vm.$nextTick()

    expect(target.find('.book__paper').classes()).not.toContain('book__paper--unset')
    expect(getTranslate(target)).toBe('0px 0')

    target.unmount()
  })
})
