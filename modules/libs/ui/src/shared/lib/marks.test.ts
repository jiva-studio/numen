/** The wikilink as markdown-it reads it. */
import MarkdownIt from 'markdown-it'
import { describe, expect, it } from 'vitest'
import { wikilinks } from './marks'

const md = new MarkdownIt({ html: false }).use(wikilinks)

describe('a wikilink in markdown', () => {
  it('is a link carrying the address it points at', () => {
    expect(md.renderInline('see [[Entropy]] now')).toBe(
      'see <a href="name://Entropy">Entropy</a> now',
    )
  })

  it('is read in the sentence by its alias', () => {
    expect(md.renderInline('[[Entropy|the other way]]')).toBe(
      '<a href="name://Entropy">the other way</a>',
    )
  })

  it('is made of the tokens every link is made of', () => {
    const tokens = md.parseInline('[[Entropy]]', {})[0]?.children ?? []

    expect(tokens.map((token) => [token.type, token.tag, token.nesting])).toStrictEqual([
      ['link_open', 'a', 1],
      ['text', '', 0],
      ['link_close', 'a', -1],
    ])
    expect(tokens[0]?.attrGet('href')).toBe('name://Entropy')
    expect(tokens[1]?.content).toBe('Entropy')
  })

  it('is found after closing brackets that belong to nothing', () => {
    expect(md.renderInline('x]][[a]]')).toBe('x]]<a href="name://a">a</a>')
  })

  it('is not brackets holding nothing', () => {
    expect(md.renderInline('[[ ]]')).toBe('[[ ]]')
  })

  it('is not brackets that never close', () => {
    expect(md.renderInline('[[abc')).toBe('[[abc')
  })

  it('begins at the brackets that close it', () => {
    expect(md.renderInline('[[a [[b]]')).toBe('[[a <a href="name://b">b</a>')
  })

  it('pushes no token while only being skipped over', () => {
    const defaultMd = new MarkdownIt()
    const recognized = new Set(defaultMd.inline.ruler.getRules(''))
    const rule = md.inline.ruler.getRules('').find((each) => !recognized.has(each))
    const src = '[[Entropy]]'
    const state = new md.inline.State(src, md, {}, [])

    expect(rule?.(state, true)).toBe(true)
    expect(state.tokens).toHaveLength(0)
    expect(state.pos).toBe(src.length)
  })
})
