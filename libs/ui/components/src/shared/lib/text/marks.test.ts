/** Pieces of a text the caller names, as markdown-it reads them. */
import MarkdownIt from 'markdown-it'
import { describe, expect, it } from 'vitest'
import { indexLinkPieces, linkPieces, type LinkPiece } from './marks'

const md = new MarkdownIt({ html: false }).use(linkPieces)

const renderWith = (text: string, pieces: readonly LinkPiece[]) =>
  md.renderInline(text, { pieces: indexLinkPieces(pieces) })

const ENTROPY: LinkPiece = { raw: '[[Entropy]]', href: 'x://Entropy', text: 'Entropy' }

describe('a piece named as a link', () => {
  it('is a link carrying the href it was named with', () => {
    expect(renderWith('see [[Entropy]] now', [ENTROPY])).toBe(
      'see <a href="x://Entropy">Entropy</a> now',
    )
  })

  it('reads as the text it was named with', () => {
    const piece = { raw: '[[E|other]]', href: 'x://E', text: 'the other way' }
    expect(renderWith('[[E|other]]', [piece])).toBe('<a href="x://E">the other way</a>')
  })

  it('is made of the tokens every link is made of', () => {
    const tokens =
      md.parseInline('[[Entropy]]', { pieces: indexLinkPieces([ENTROPY]) })[0]?.children ?? []

    expect(tokens.map((token) => [token.type, token.tag, token.nesting])).toStrictEqual([
      ['link_open', 'a', 1],
      ['text', '', 0],
      ['link_close', 'a', -1],
    ])
    expect(tokens[0]?.attrGet('href')).toBe('x://Entropy')
    expect(tokens[1]?.content).toBe('Entropy')
  })

  it('links every occurrence of the piece', () => {
    expect(renderWith('[[Entropy]], [[Entropy]]', [ENTROPY])).toBe(
      '<a href="x://Entropy">Entropy</a>, <a href="x://Entropy">Entropy</a>',
    )
  })

  it('is matched by the longest piece that starts there', () => {
    const short = { raw: 'ab', href: 'x://short', text: 'S' }
    const long = { raw: 'abc', href: 'x://long', text: 'L' }
    expect(renderWith('abc', [short, long])).toBe('<a href="x://long">L</a>')
  })

  it('is found in the middle of a word', () => {
    expect(renderWith('xyz', [{ raw: 'y', href: 'x://y', text: 'Y' }])).toBe(
      'x<a href="x://y">Y</a>z',
    )
  })

  it('is not drawn inside the text of another link, and is drawn after it', () => {
    expect(renderWith('[a [[Entropy]] b](https://example.com) [[Entropy]]', [ENTROPY])).toBe(
      '<a href="https://example.com">a [[Entropy]] b</a> <a href="x://Entropy">Entropy</a>',
    )
    expect(
      renderWith(
        '![a [[Entropy]] b](https://example.com/img.png) [c [[Entropy]] d](https://example.com)',
        [ENTROPY],
      ),
    ).toBe(
      '<img src="https://example.com/img.png" alt="a Entropy b"> <a href="https://example.com">c [[Entropy]] d</a>',
    )
    expect(
      renderWith('![see a]b here](https://example.com/img.png)', [
        { raw: 'a]b', href: 'x://ab', text: 'AB' },
      ]),
    ).toBe('<img src="https://example.com/img.png" alt="see AB here">')
  })

  it('ignores empty pieces and leaves text alone where nothing is named', () => {
    expect(indexLinkPieces([{ raw: '', href: 'x://empty', text: 'E' }]).size).toBe(0)
    expect(md.renderInline('see [[Entropy]] now')).toBe('see [[Entropy]] now')
    expect(renderWith('see [[Entropy]] now', [])).toBe('see [[Entropy]] now')
  })

  it('pushes no token while only being skipped over and respects posMax', () => {
    const defaultMd = new MarkdownIt()
    const recognized = new Set(defaultMd.inline.ruler.getRules(''))
    const rule = md.inline.ruler
      .getRules('')
      .find((each) => !recognized.has(each) && each.name === 'linkPiece')
    const textRule = md.inline.ruler.getRules('').find((each) => each.name === 'text')
    const src = '[[Entropy]]'
    const state = new md.inline.State(src, md, { pieces: indexLinkPieces([ENTROPY]) }, [])

    expect(rule?.(state, true)).toBe(true)
    expect(state.tokens).toHaveLength(0)
    expect(state.pos).toBe(src.length)

    const truncated = new md.inline.State(src, md, { pieces: indexLinkPieces([ENTROPY]) }, [])
    truncated.posMax = 5
    expect(rule?.(truncated, false)).toBe(false)

    const noEnv = new md.inline.State(src, md, undefined as never, [])
    expect(rule?.(noEnv, false)).toBe(false)

    const textState = new md.inline.State(
      'hello [[Entropy]]',
      md,
      { pieces: indexLinkPieces([ENTROPY]) },
      [],
    )
    expect(textRule?.(textState, false)).toBe(true)
    expect(textState.pending).toBe('hello ')
    expect(textState.pos).toBe(6)

    const boundedText = new md.inline.State('hello world', md, {}, [])
    boundedText.posMax = 5
    expect(textRule?.(boundedText, false)).toBe(true)
    expect(boundedText.pending).toBe('hello')
    expect(boundedText.pos).toBe(5)
    expect(textRule?.(textState, false)).toBe(false)

    const silentText = new md.inline.State(
      'hello!',
      md,
      { pieces: indexLinkPieces([{ raw: '!', href: 'x://bang', text: '!' }]) },
      [],
    )
    expect(textRule?.(silentText, true)).toBe(true)
    expect(silentText.pending).toBe('')
    expect(silentText.pos).toBe(5)
  })
})
