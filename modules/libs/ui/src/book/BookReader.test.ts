import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import BookReader from './BookReader.vue'
import { chapterOf } from '@/fixtures/book'

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
})

describe('turning past the end of a document', () => {
  it('asks for the offset the next document begins at', async () => {
    const held = await reader()

    await held.find('.book__area').trigger('keydown', { key: 'ArrowRight' })

    expect(asked(held)).toEqual([CHAPTER.span.ends])
  })

  it('asks for the offset before this document, turning back', async () => {
    const held = await reader()

    await held.find('.book__area').trigger('keydown', { key: 'ArrowLeft' })

    expect(asked(held)).toEqual([CHAPTER.span.begins - 1])
  })

  it('asks for nothing past either end of the book itself', async () => {
    const held = await reader(CHAPTER.span)

    await held.find('.book__area').trigger('keydown', { key: 'ArrowRight' })
    await held.find('.book__area').trigger('keydown', { key: 'ArrowLeft' })

    expect(asked(held)).toHaveLength(0)
  })

  it('turns nothing on a key a book is not read with', async () => {
    const held = await reader()

    await held.find('.book__area').trigger('keydown', { key: 'Enter' })
    await held.find('.book__area').trigger('keydown', { key: 'a' })

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
    expect(held.findAll('[data-at]')).toHaveLength(0)
    expect(asked(held)).toHaveLength(0)
  })
})
