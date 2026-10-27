/**
 * A link pressed in a book's text: what is followed, what is left to the
 * window, and where a link to another document is held.
 */
import { describe, expect, it, vi } from 'vitest'

import type { SettledBookProps } from '../lib/props'
import { createBookLinks } from './links'

const props = { path: 'text/one.xhtml', span: { start: 40, end: 90 } } as SettledBookProps

const setUp = () => {
  const reportMove = vi.fn<(pos: number) => void>()
  const reportFollow = vi.fn<(path: string) => void>()
  const links = createBookLinks(props, { reportMove, reportFollow })
  const findFragmentOffset = vi.fn<(fragment: string) => number | undefined>(() => 12)
  return { links, reportMove, reportFollow, findFragmentOffset }
}

/** A press landing on the text inside a link, or on a bare paragraph. */
const pressOn = (markup: string, selector: string) => {
  const root = document.createElement('div')
  root.innerHTML = markup
  const preventDefault = vi.fn<() => void>()
  const press = { target: root.querySelector(selector), preventDefault } as unknown as MouseEvent
  return { press, preventDefault }
}

describe('a press on a link', () => {
  it('moves within the document to the place the link names', () => {
    const { links, reportMove, reportFollow, findFragmentOffset } = setUp()
    const { press, preventDefault } = pressOn('<a href="#n3"><b>note</b></a>', 'b')

    links.follow(press, findFragmentOffset)

    expect(preventDefault).toHaveBeenCalledOnce()
    expect(findFragmentOffset).toHaveBeenCalledWith('n3')
    expect(reportMove).toHaveBeenCalledExactlyOnceWith(12)
    expect(reportFollow).not.toHaveBeenCalled()
  })

  it('moves to the start of the document where the place is not drawn', () => {
    const { links, reportMove, findFragmentOffset } = setUp()
    findFragmentOffset.mockReturnValue(undefined)

    links.follow(pressOn('<a href="#gone">x</a>', 'a').press, findFragmentOffset)

    expect(reportMove).toHaveBeenCalledExactlyOnceWith(40)
  })

  it('moves within the document where the link names the document itself', () => {
    const { links, reportMove, reportFollow, findFragmentOffset } = setUp()

    links.follow(pressOn('<a href="text/one.xhtml#n3">x</a>', 'a').press, findFragmentOffset)

    expect(reportMove).toHaveBeenCalledExactlyOnceWith(12)
    expect(reportFollow).not.toHaveBeenCalled()
  })

  it('reports another document and holds where the link led', () => {
    const { links, reportMove, reportFollow, findFragmentOffset } = setUp()

    links.follow(pressOn('<a href="text/two.xhtml#n3">x</a>', 'a').press, findFragmentOffset)

    expect(reportFollow).toHaveBeenCalledExactlyOnceWith('text/two.xhtml')
    expect(reportMove).not.toHaveBeenCalled()
    expect(links.takeLed()).toStrictEqual({ path: 'text/two.xhtml', fragment: 'n3' })
    expect(links.takeLed()).toBeUndefined()
  })

  it('is left alone where the link leads out of the book', () => {
    const { links, reportMove, reportFollow, findFragmentOffset } = setUp()
    const { press, preventDefault } = pressOn('<a href="https://example.org/page">x</a>', 'a')

    links.follow(press, findFragmentOffset)

    expect(preventDefault).toHaveBeenCalledOnce()
    expect(reportMove).not.toHaveBeenCalled()
    expect(reportFollow).not.toHaveBeenCalled()
  })
})

describe('a press that is not on a link', () => {
  it('does nothing on text outside any link', () => {
    const { links, reportMove, reportFollow, findFragmentOffset } = setUp()
    const { press, preventDefault } = pressOn('<p>plain</p>', 'p')

    links.follow(press, findFragmentOffset)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(reportMove).not.toHaveBeenCalled()
    expect(reportFollow).not.toHaveBeenCalled()
  })

  it('does nothing on an anchor with no href', () => {
    const { links, reportMove, findFragmentOffset } = setUp()
    const { press, preventDefault } = pressOn('<a name="n3">x</a>', 'a')

    links.follow(press, findFragmentOffset)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(reportMove).not.toHaveBeenCalled()
  })

  it('does nothing where the press has no target', () => {
    const { links, reportMove, findFragmentOffset } = setUp()
    const preventDefault = vi.fn<() => void>()

    links.follow({ target: null, preventDefault } as unknown as MouseEvent, findFragmentOffset)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(reportMove).not.toHaveBeenCalled()
  })

  it('does nothing where the target is no element', () => {
    const { links, reportMove, findFragmentOffset } = setUp()
    const preventDefault = vi.fn<() => void>()

    links.follow({ target: {}, preventDefault } as unknown as MouseEvent, findFragmentOffset)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(reportMove).not.toHaveBeenCalled()
  })
})
